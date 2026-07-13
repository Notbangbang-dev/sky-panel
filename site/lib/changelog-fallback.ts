// Build-time bundled copy of the latest changelog entry, used as a fallback
// when the live fetch of CHANGELOG.md from GitHub fails or returns empty.
// This guarantees the changelog page never renders an empty timeline.
// Keep the newest 1-2 entries in sync with the top of ../../CHANGELOG.md.
export const CHANGELOG_FALLBACK = `## [0.23.0] - 2026-07-08

### ✨ New Features

- **Multiple ports per server (admin-assigned).** A server can now hold more than one port — handy for a Minecraft query port, a proxy, a voice-chat mod, dynmap, and the like. Manage them from **Admin → Servers → Ports** on any server: attach a free port from that server's node, or remove an additional one. Changing ports recreates the container (the server restarts, but its files are preserved), and every published port is opened in the node firewall automatically (see below). The primary port can't be removed. Owners can see their server's extra ports on the server page (\`node-ip:port\` connect address unchanged) but only an admin can add or remove them. The container also gets \`SERVER_ADDITIONAL_PORTS\` (comma-separated) and \`SERVER_PORT_1..n\` env vars so eggs can reference the extra ports.
- **Automatic firewall port-forwarding.** Published ports are now opened in the node's \`ufw\` firewall automatically when a container is created, so an allocated port is reachable without hand-writing a rule. This lives in **sky-daemon v0.6.0** — best-effort, configurable (\`SKY_MANAGE_FIREWALL=auto|sudo|off\`), and Linux-only. On a cloud host you still need the port open in your provider's security group.

### 🔒 Hardening

Attaching a port is an atomic, node-scoped claim: a specific allocation can only be attached if it's genuinely free (a concurrent claim can't double-book it) and on the same node as the server; a failed re-provision rolls the claim back so the database never shows a port attached to a server that isn't using it; and the primary port is guarded against removal.

### 🔗 Requires

- **sky-daemon v0.6.0** on each node for the automatic firewall opening. Multi-port publishing itself works with any current daemon (the wire protocol already carries a list of ports); older daemons simply won't open the firewall for you.
`;
