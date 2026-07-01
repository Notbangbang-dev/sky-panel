# Architecture: panel ↔ daemon protocol

`panel-api` (Go) and [`sky-daemon`](https://github.com/Notbangbang-dev/sky-daemon)
(Rust) talk over a single long-lived WebSocket per node, opened outbound by
the daemon so no inbound ports need to be exposed on the node. This document
describes that wire protocol — implemented twice, once per language, in
`panel-api/internal/agenthub` and `sky-daemon/protocol`, and kept
byte-compatible on purpose so either side can be read as the spec for the
other.

## Envelope

Every message on the wire, in both directions, is a JSON `Envelope`:

```json
{
  "type": "command",
  "timestamp": 1782900000,
  "nonce": "3f9a1c2e7b4d5a6f8091a2b3c4d5e6f7",
  "payload": { "...": "..." },
  "sig": "…hex HMAC-SHA256…"
}
```

- `type` — one of `hello`, `heartbeat`, `event`, `ack`, `command`.
- `timestamp` — unix seconds when the envelope was signed.
- `nonce` — 16 random bytes, hex-encoded, freshly generated per envelope.
- `payload` — the type-specific body, transported as raw JSON (not
  re-encoded) so both sides can hash the exact bytes that were signed.
- `sig` — hex HMAC-SHA256 over the canonical string below, keyed by the
  node's raw token.

### Signing

```
canonical = type + "." + timestamp + "." + nonce + "." + sha256hex(payload_bytes)
sig       = hex(HMAC_SHA256(node_token, canonical))
```

`payload_bytes` is the exact byte sequence of the payload as it appears on
the wire — hashing it (rather than signing it directly) keeps the canonical
string a fixed, short size regardless of payload size, and avoids any
ambiguity from JSON re-serialization changing field order or whitespace.

### Hello is the one unsigned message

The daemon's first message after connecting is `hello`, carrying its raw
node token in the payload:

```json
{ "type": "hello", "timestamp": ..., "nonce": "...", "payload": { "node_token": "...", "agent_version": "..." }, "sig": "" }
```

`hello` isn't signed because the token isn't a shared secret *yet* from the
panel's point of view at that point in the handshake — presenting the raw
token *is* the authentication. The panel looks it up by hash
(`sha256(token)` as the DB index, so the raw token is never queried
directly), confirms it hasn't expired, and from that point on both sides
share the same secret (the node's raw token) as the HMAC key.

**Every message after hello must carry a valid signature.** The panel
verifies signature, timestamp freshness, and nonce freshness before doing
anything else with an incoming envelope (see below) — any failure closes
the connection immediately. This is a hard fail, not a soft/logged one: an
invalid signature at this point means either a bug or an attacker, and
there's no safe partial-trust state to fall back to.

### Freshness and replay protection

- **Timestamp**: rejected if more than `MAX_CLOCK_SKEW_SECS` (30s) away from
  the receiver's clock, in either direction.
- **Nonce**: each side keeps a process-local cache of nonces it has seen
  recently (`agenthub.nonceCache` in Go, `agent::nonce_cache::NonceCache` in
  Rust), swept on a TTL of 120s. 120s is deliberately wider than the ±30s
  skew window, so a message can never simultaneously (a) pass the freshness
  check and (b) have its nonce already evicted from the cache — closing the
  gap that would otherwise let a captured envelope be replayed just after
  its nonce ages out.
- A replayed or out-of-window envelope is rejected the same way a bad
  signature is: connection closed.

The nonce cache is intentionally process-local and in-memory, not persisted
or shared across a panel restart — a restart briefly reopens the replay
window for envelopes signed in the few seconds around the restart, which is
an accepted tradeoff for not needing a shared store between panel replicas.

### Rate limiting

The panel applies a token-bucket rate limiter (20/sec, burst 40) per
connection in its read loop, checked before signature verification (so a
flood of garbage doesn't even reach the crypto). Exceeding it closes the
connection.

## Message types

| type        | direction       | payload                                                              |
|-------------|-----------------|-----------------------------------------------------------------------|
| `hello`     | daemon → panel  | `{ node_token, agent_version }` — unsigned, see above                |
| `heartbeat` | daemon → panel  | per-container stats snapshot (cpu/mem/net) for every tracked server  |
| `event`     | daemon → panel  | `{ server_id, kind, message }` — console line or state-change events |
| `command`   | panel → daemon  | `{ command_id, action, server_id, ...action-specific fields }`       |
| `ack`       | daemon → panel  | `{ command_id, ok, error?, result? }` — always sent for a `command`  |

Every `command` gets exactly one `ack` back, matched by `command_id`, so the
panel's dispatch call can `await` it. `event` is fire-and-forget: console
output and start/stop/kill state transitions stream to the panel as they
happen rather than being polled.

### Command actions

`create`, `start`, `stop`, `kill`, `remove`, `console_input` drive the
container lifecycle. `list_files`, `read_file`, `write_file`, `rename_file`,
`delete_file`, `mkdir` are the file-manager actions — the daemon runs on the
host (not inside the container), so these are plain filesystem calls scoped
to `{volumes_root}/{server_id}/{relative_path}`, guarded lexically against
path traversal (rejecting any `..`, absolute path, or path-prefix escape
component before touching the filesystem — no `canonicalize`, since a
write/mkdir target may not exist yet). File content moves as base64 inside
the JSON payload, capped at 10MB per file — there's no separate streaming
binary channel yet, which is a known limitation for larger files.

## Node token lifecycle

The `nodes` table stores the token's hash (`token_hash`, used as the lookup
index on `hello`) **and** the raw token (`token`) — the raw value doubles as
the HMAC signing key for every message after hello, so unlike a normal
credential it can't be one-way-hashed at rest. This is a deliberate
tradeoff, documented here rather than silently: treat panel database access
as sensitive in a way a purely-hashed-credentials table wouldn't require.

Tokens default to a 90-day expiry (`expires_at`), checked on every `hello`.
An admin can rotate a node's token (`POST /api/v1/admin/nodes/{id}/rotate-token`)
without deleting and recreating the node — this issues a fresh token/expiry
and immediately invalidates the old one, since `hello` re-authenticates
against the current `token_hash` on every connection.

## Why signed-symmetric instead of mTLS

Both directions sign every message with the same shared secret (the node's
token) rather than each side holding its own TLS client certificate. This
was a deliberate scope choice for this pass: it closes the actual gap that
existed (an authenticated-but-unverified connection could have any message
injected or replayed on it) with a change contained entirely to the
application layer — no certificate authority, issuance, or rotation
infrastructure to stand up. mTLS remains a reasonable future upgrade if the
threat model expands to include compromised transport (e.g. a
misconfigured/legacy TLS-terminating proxy in front of the panel).
