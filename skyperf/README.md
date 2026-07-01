# skyperf

`skyperf` is a small Rust CLI used by the Sky Panel Go backend for the
handful of operations that are genuinely perf-sensitive: recursive
directory sizing, streaming tar+zstd backups, and log tailing. It is
intentionally tiny — everything else in Sky Panel lives in Go, and this
tool is meant to be shelled out to as a subprocess, with each subcommand
printing a single well-defined JSON shape to stdout.

All commands print machine-readable JSON. On success, exit code is `0`.
On failure, exactly one JSON line of the form `{"error": "<message>"}` is
printed to stdout and the process exits with a non-zero status.

## Build

```
cargo build --release
```

The resulting binary is at `target/release/skyperf` (`skyperf.exe` on
Windows).

## Subcommands

### `skyperf dirsize <path>`

Recursively computes the total size in bytes of every regular file under
`<path>`. Symlinks are never followed (safe against symlink cycles).

```
skyperf dirsize /var/lib/sky-panel/servers/1
```

Success (stdout, exit 0):

```json
{"path": "/var/lib/sky-panel/servers/1", "bytes": 1048576}
```

Failure (stdout, exit non-zero), e.g. path does not exist or permission
denied:

```json
{"error": "The system cannot find the file specified. (os error 2)"}
```

### `skyperf backup create <src_dir> <dest.tar.zst>`

Streams a zstd-compressed tar archive of everything under `<src_dir>` to
`<dest.tar.zst>`. Never holds the whole archive in memory.

```
skyperf backup create /var/lib/sky-panel/servers/1 /backups/server-1.tar.zst
```

Success (stdout, exit 0):

```json
{"created": "/backups/server-1.tar.zst", "bytes": 5242880}
```

`bytes` is the size in bytes of the resulting archive file on disk.

Failure (stdout, exit non-zero), e.g. `src_dir` does not exist:

```json
{"error": "source directory not found: /var/lib/sky-panel/servers/1"}
```

### `skyperf backup restore <archive.tar.zst> <dest_dir>`

Extracts a zstd+tar archive (as produced by `backup create`) into
`<dest_dir>`, creating it if it doesn't exist.

```
skyperf backup restore /backups/server-1.tar.zst /var/lib/sky-panel/servers/1-restored
```

Success (stdout, exit 0):

```json
{"restored": "/var/lib/sky-panel/servers/1-restored"}
```

Failure (stdout, exit non-zero):

```json
{"error": "<message>"}
```

**Path traversal guard:** any archive entry whose path is absolute or
contains a `..` component (i.e. would escape `dest_dir` once joined) is
rejected. Rejected entries are skipped (not extracted) and a warning is
written to **stderr** — stdout still only ever contains the single
success/error JSON line described above. This makes `backup restore` safe
to run against archives from an untrusted or corrupted source.

### `skyperf tail <path> [--follow]`

Without `--follow`: prints the last 200 lines of `<path>`, one JSON line
per input line, to stdout, then exits 0.

```
skyperf tail /var/log/server-1/latest.log
```

Output (stdout, one JSON object per line, exit 0 when done):

```json
{"line": "[12:00:01] Server started"}
{"line": "[12:00:02] Loading world..."}
```

With `--follow`: prints the last 200 lines as above, then keeps running
and polls the file every ~250ms for appended content. Each time a new
complete line appears, a `{"line": "<text>"}` JSON line is written to
stdout and flushed immediately, so a parent process reading the pipe sees
it in real time.

```
skyperf tail /var/log/server-1/latest.log --follow
```

In `--follow` mode there is no separate "done" marker — the process is
expected to run indefinitely until one of:

- **stdin reaches EOF** (i.e. the pipe skyperf's stdin is connected to is
  closed). This is the recommended clean-shutdown signal: keep a pipe open
  on the child's stdin for as long as you want it to keep following, and
  close it when you want skyperf to stop. Note that if the parent process
  does not explicitly wire up stdin (e.g. Go's `exec.Command` defaults an
  unset `Stdin` to the null device), skyperf will see immediate EOF and
  exit right after printing the initial lines — so the Go side must
  attach a real pipe to stdin if it wants `--follow` to keep running.
- the file being tailed disappears (`fs::metadata` starts failing), or
- the process receives SIGINT or is killed outright (the expected
  shutdown path when the Go parent is done with it).

In all of these cases skyperf exits with status 0 and prints nothing
further; it does not attempt graceful in-flight cleanup beyond flushing
already-written lines, since it is designed to be a disposable subprocess
under the Go process's control.

If the file is truncated or replaced while following (e.g. log rotation),
skyperf detects that the file has shrunk and restarts reading from the
beginning of the new file.
