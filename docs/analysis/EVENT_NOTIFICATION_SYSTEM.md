# Event Notification System: Critical Analysis

> Research document evaluating approaches for advertising multiclaude events to external consumers.

## Context

Multiclaude currently has no mechanism for external consumers to observe system events in real time. Internal event flow is handled through two mechanisms: (1) filesystem-based JSON message files polled every 2 minutes by the daemon's `messageRouterLoop`, and (2) direct tmux pane text injection for immediate agent notifications. Neither is designed for external consumption.

This document evaluates five candidate approaches for an event notification system, assessing each against multiclaude's design philosophy, existing architecture, and practical constraints.

## Events Under Consideration

| Event | Source | Current Visibility |
|-------|--------|-------------------|
| `agent_complete` | `complete_agent` handler in daemon | State update + message to supervisor |
| `pr_merged` | merge-queue agent / daemon cleanup | Merge-queue agent detects via `gh` |
| `health_check_failed` | `checkAgentHealth()` loop | Daemon log + crash message to supervisor |
| `agent_crashed` | `checkAgentHealth()` loop | State update (`CrashedAt`) + supervisor message |
| `message_sent` | `messages.Manager.Send()` | JSON file created on disk |
| `worktree_refreshed` | `refreshWorktrees()` loop | Daemon log + agent message |
| `worker_spawned` | `add_agent` handler | State update |

## Governing Constraints

Before evaluating options, these constraints from ROADMAP.md and CLAUDE.md must be acknowledged:

1. **Explicitly out of scope**: "Notification systems (Slack, Discord, webhooks)" — ROADMAP.md lists this as something users should build themselves.
2. **No web interfaces**: WebSocket servers contradict the "no web dashboards" principle.
3. **No plugin/extension systems**: Any event system must be a simple, integrated primitive — not a framework.
4. **Terminal-native**: The interface must work naturally from a terminal.
5. **Local-first**: No external service dependencies.
6. **Simple > clever**: "Prefer deleting code over complexity."
7. **Filesystem as database**: The existing patterns use JSON files on disk, not in-memory state.
8. **Current phase is Stabilization**: Adding an event system is additive feature work while the roadmap says to focus on reliability.

**The tension**: The roadmap says notification systems are out of scope, but the question of how to *expose* events is distinct from building notification integrations. A low-level event primitive (like a log file) enables users to build their own notification systems — which is exactly what the roadmap suggests they do. The key distinction is: **event emission is infrastructure; notification routing is application.**

---

## Option 1: Subscribe Command on Existing Unix Socket

**Description**: Add a `subscribe` command to the existing request/response socket protocol. After sending `{"command": "subscribe", "args": {"events": ["agent_complete", "agent_crashed"]}}`, the connection stays open and the server pushes newline-delimited JSON events until the client disconnects.

### Assessment

**Requires**:
- Rewriting `handleConnection()` in `internal/socket/socket.go` to support persistent connections
- Adding client tracking (subscription registry, connection lifecycle management)
- A publish/subscribe mechanism in the daemon (event bus → fan-out to subscribers)
- Graceful handling of slow consumers, disconnects, backpressure
- Changing the `Handler` interface — currently returns a single `Response`

**Strengths**:
- Single socket for everything (no new files/ports)
- Clients can filter events at subscription time
- Real-time delivery (no polling)

**Weaknesses**:
- **Fundamental protocol mismatch**: The current socket is strictly request/response — one request, one response, connection closed. This isn't an incremental change; it's a protocol redesign.
- **Complexity explosion**: Persistent connections require client tracking, heartbeats, subscription management, fan-out buffering, and graceful degradation. The current `socket.go` is 156 lines. This would triple it minimum.
- **Breaks simplicity contract**: The socket server currently handles 19 commands via a simple switch statement. Adding streaming creates two fundamentally different code paths.
- **No replay**: If a consumer disconnects and reconnects, events during the gap are lost.
- **Testing burden**: Concurrent streaming connections are significantly harder to test than request/response.

**Verdict**: **Reject.** The architectural cost is disproportionate to the value. This turns a simple RPC socket into a message broker. The existing socket protocol is clean and working — upgrading it to support streaming pollutes a good design.

---

## Option 2: Dedicated Second Unix Domain Socket

**Description**: Create a second Unix socket (`~/.multiclaude/events.sock`) exclusively for event streaming. Clients connect and receive a stream of newline-delimited JSON events. The main socket remains unchanged for command/response.

### Assessment

**Requires**:
- New socket server (can reuse `net` primitives, but different protocol)
- Event publishing mechanism in daemon
- Client connection tracking for the event socket
- New daemon loop or goroutine for event fan-out

**Strengths**:
- Clean separation: command socket stays simple, event socket is purpose-built
- Real-time delivery
- Main protocol untouched

**Weaknesses**:
- **Same complexity, different address**: All the streaming problems from Option 1 still exist (client tracking, backpressure, disconnects, fan-out). Moving them to a second socket doesn't reduce complexity — it just moves it.
- **Two sockets to manage**: Daemon startup/shutdown must handle both. Health checks must verify both. Cleanup must remove both.
- **No replay**: Same gap problem as Option 1.
- **Harder to discover**: Users must know about two sockets.
- **Overkill for the use case**: Multiclaude runs 3-5 agents generating events every few minutes. A streaming socket is infrastructure for a problem that produces maybe 20 events per hour.

**Verdict**: **Reject.** This is the right architecture for a high-throughput event system, but multiclaude's event volume doesn't justify the complexity. The separation of concerns is good in principle but adds operational burden without proportional benefit.

---

## Option 3: Append-Only JSONL Event Log File

**Description**: The daemon appends one JSON object per line to `~/.multiclaude/events.jsonl`. External consumers read events via `tail -f ~/.multiclaude/events.jsonl`. Optional: rotate the file periodically.

### Assessment

**Requires**:
- An `EventLogger` that appends JSON lines to a file (trivial: `json.Marshal` + `\n` + `file.Write`)
- Emit calls at event sites in the daemon (e.g., after `complete_agent`, in `checkAgentHealth`, etc.)
- Log rotation (reuse existing `rotateLogsIfNeeded()` pattern from daemon.go)
- Documented JSON schema for each event type

**Strengths**:
- **Terminal-native**: `tail -f events.jsonl` is the consumption model. Every Unix user knows this.
- **Zero client complexity**: No connections, no subscriptions, no protocol. It's a file.
- **Full replay**: The entire event history is on disk. Consumers can `cat` or `jq` the full log.
- **Composable**: `tail -f events.jsonl | jq 'select(.type == "agent_complete")'` — instant filtering.
- **Scriptable**: Shell scripts, Python, Go — anything that can read a file can consume events.
- **Crash-safe**: Events survive daemon restarts. Partially written lines are detectable (incomplete JSON).
- **Matches existing patterns**: The daemon already writes `daemon.log` with rotation. This is the same pattern with structured output.
- **Minimal implementation**: ~50-80 lines of Go. An `EventLog` struct with `Emit(event Event)` and `Rotate()`.
- **Inspectable**: `cat events.jsonl | jq .` shows everything. Matches multiclaude's "filesystem is the database" philosophy.
- **Users build their own notifications**: `tail -f events.jsonl | while read line; do echo "$line" | jq -r 'select(.type == "agent_crashed") | .agent' | xargs -I{} notify-send "Agent {} crashed"; done`

**Weaknesses**:
- **Not real-time**: `tail -f` has a small latency (typically <1 second, but filesystem-dependent). Acceptable for multiclaude's event frequency.
- **File growth**: Without rotation, the file grows unbounded. But rotation is already a solved problem in the codebase (`rotateLogsIfNeeded()`).
- **No selective subscription**: Consumers get all events and must filter client-side. At multiclaude's event volume (~20/hour), this is a non-issue.
- **Atomic line writes**: Must ensure each `Emit()` writes a complete line atomically. Go's `os.File.Write()` is atomic for writes ≤ pipe buffer size (4KB on macOS/Linux), and event JSON will be well under this.
- **Rotation coordination**: When rotating, consumers following with `tail -f` lose their position. Mitigated by: (a) rotating infrequently, (b) using `tail -F` (capital F, follows by name), or (c) numbered rotation files.

**Event Schema Example**:
```json
{"ts":"2026-03-01T10:30:45Z","type":"agent_complete","repo":"multiclaude","agent":"calm-fox","data":{"summary":"Fixed parser bug","pr_url":"https://github.com/...","pr_number":42}}
{"ts":"2026-03-01T10:32:01Z","type":"health_check_failed","repo":"multiclaude","agent":"bold-eagle","data":{"reason":"tmux_window_missing","had_uncommitted":true}}
```

**Verdict**: **Strong recommend.** This is the right answer for multiclaude. It's terminal-native, composable, inspectable, crash-safe, and trivial to implement. It enables users to build their own notification systems (which is what the roadmap says they should do) without multiclaude itself becoming a notification system.

---

## Option 4: Filesystem Event Directory

**Description**: Mirror the existing `messages/` pattern — create `~/.multiclaude/events/<repo>/` with individual JSON files per event (e.g., `evt-<uuid>.json`). Consumers poll the directory or use filesystem watchers.

### Assessment

**Requires**:
- Event directory structure under `~/.multiclaude/events/`
- Event creation function (similar to `messages.Manager.Send()`)
- Cleanup/rotation mechanism (events accumulate without garbage collection)
- Consumer-side tooling to watch or poll the directory

**Strengths**:
- **Familiar pattern**: Mirrors `messages/` exactly. Developers who understand messages understand events.
- **Per-event atomicity**: Each event is a complete, independent file.
- **Per-repo organization**: Events naturally scoped to repositories.
- **Individual file reads**: No parsing a stream — each file is self-contained.

**Weaknesses**:
- **Directory listing as event stream is terrible UX**: `ls -lt events/multiclaude/ | head` is the consumption model. Compare this to `tail -f events.jsonl`. The file-per-event model is worse in every interactive scenario.
- **Filesystem overhead**: Creating, listing, and deleting thousands of small files is expensive. The `messages/` system works because message volume is low and messages are deleted after acking. Events are never "acked" — they accumulate.
- **Polling required**: No built-in way to "follow" a directory. You'd need `inotifywait` (Linux) or `fswatch` (macOS), which are external dependencies.
- **Cleanup complexity**: Who deletes old events? Messages have a clear lifecycle (pending → acked → deleted). Events don't. You'd need TTL-based garbage collection — more code, more configuration.
- **Ordering**: Filesystem directory listings don't guarantee insertion order. You'd need to parse timestamps from each file to reconstruct event sequence. The JSONL log preserves order by construction.
- **Already proven suboptimal**: The `messages/` system's 2-minute polling latency is acceptable for inter-agent IPC but would be frustrating for event consumers who want to watch events in real time.

**Verdict**: **Reject.** This takes the worst aspects of the messages/ pattern (polling, directory scanning, cleanup) and applies them to a use case that is fundamentally a log stream. A JSONL file is strictly superior: same durability, better ordering, better consumption UX, less filesystem overhead, simpler implementation.

---

## Option 5: WebSocket Server

**Description**: The daemon runs an HTTP/WebSocket server (e.g., on `localhost:9999`). Clients connect via WebSocket and receive JSON event frames.

### Assessment

**Requires**:
- HTTP server with WebSocket upgrade (`gorilla/websocket` or `nhooyr.io/websocket`)
- New dependency (no WebSocket in Go stdlib)
- Port management (conflict avoidance, configuration)
- TLS considerations (even for localhost)
- Client connection management

**Strengths**:
- Rich ecosystem of WebSocket client libraries
- Could serve a future web dashboard
- Familiar to web developers

**Weaknesses**:
- **Directly contradicts ROADMAP.md**: "Out of Scope: Web interfaces or dashboards." A WebSocket server is web infrastructure, even if consumed from a CLI tool.
- **New external dependency**: Multiclaude currently has zero non-stdlib dependencies for networking. Adding a WebSocket library breaks this.
- **Port conflicts**: Unix sockets avoid port management entirely. WebSocket servers need a port, which could conflict with other services.
- **Overkill**: WebSocket is designed for bidirectional, high-frequency communication. Multiclaude events are unidirectional and low-frequency.
- **Security surface**: An HTTP server listening on localhost is accessible to any process on the machine. Unix sockets have filesystem permissions.
- **Doesn't fit the user**: Multiclaude users are terminal-first. `wscat ws://localhost:9999` is a worse experience than `tail -f events.jsonl`.

**Verdict**: **Hard reject.** This contradicts the project's stated philosophy on multiple axes. It adds complexity, dependencies, and a web-shaped interface to a terminal-native tool.

---

## Comparative Summary

| Criterion | Socket Subscribe | Second Socket | JSONL Log | Event Dir | WebSocket |
|-----------|:---:|:---:|:---:|:---:|:---:|
| Terminal-native | Partial | Partial | **Yes** | Partial | No |
| Implementation complexity | High | High | **Low** | Medium | High |
| Matches existing patterns | No | No | **Yes** (daemon.log) | Yes (messages/) | No |
| Event replay | No | No | **Yes** | Yes | No |
| Real-time capable | Yes | Yes | **~Yes** (tail -f) | No (polling) | Yes |
| Crash-safe | No | No | **Yes** | Yes | No |
| Composable (pipes, jq) | No | No | **Yes** | Partial | No |
| Zero dependencies | Yes | Yes | **Yes** | Yes | No |
| Lines of code estimate | ~300-500 | ~300-500 | **~50-80** | ~150-200 | ~400+ |
| Aligns with ROADMAP | Neutral | Neutral | **Yes** | Neutral | **Conflicts** |

## Recommendation

**Option 3: Append-only JSONL event log file.**

It is the only option that:
- Is genuinely terminal-native (consumed via `tail -f` and `jq`)
- Has near-zero implementation cost (~50-80 lines)
- Provides full event replay (the file is the history)
- Requires no protocol changes to the existing socket
- Matches an existing codebase pattern (daemon.log rotation)
- Enables users to build their own notification systems (per ROADMAP.md)
- Respects the stabilization phase (minimal risk, minimal code)

### Suggested Implementation Sketch

```
~/.multiclaude/events.jsonl          # Current event log
~/.multiclaude/events.jsonl.1        # Rotated (if rotation needed)
```

Each line:
```json
{"ts":"...","type":"agent_complete","repo":"multiclaude","agent":"calm-fox","data":{...}}
```

Emit points in daemon.go:
- `complete_agent` handler → `agent_complete`
- `checkAgentHealth()` crash detection → `agent_crashed` / `health_check_failed`
- `add_agent` handler → `agent_spawned`
- `routeMessages()` delivery → `message_delivered`
- `refreshWorktrees()` completion → `worktree_refreshed`
- Merge-queue PR detection → `pr_merged`

Rotation: Reuse the existing `rotateLogsIfNeeded()` pattern from the daemon, triggered in `healthCheckLoop`.

### What This Does NOT Do (Intentionally)

- Does not build Slack/Discord/webhook integrations (out of scope per ROADMAP.md)
- Does not add a subscription protocol to the socket (unnecessary complexity)
- Does not create a plugin system (out of scope per ROADMAP.md)
- Does not add a web interface (out of scope per ROADMAP.md)

Users who want notifications can:
```bash
tail -f ~/.multiclaude/events.jsonl | jq -r 'select(.type == "agent_crashed") | "CRASHED: \(.agent)"' | while read msg; do
  osascript -e "display notification \"$msg\" with title \"multiclaude\""
done
```

That's their code, not ours. The JSONL file is the primitive that makes it possible.

## Open Questions

1. **Rotation policy**: Rotate at what size? 10MB matches daemon.log. Or rotate daily?
2. **Event schema versioning**: Should events include a `v` field for schema version?
3. **Scope**: Should this be a P1 (this quarter) or P2 (backlog) item? It's low-effort but additive.
4. **`multiclaude events` CLI command**: Should there be a built-in `tail -f` wrapper? e.g., `multiclaude events --follow --type agent_complete`. Useful but not essential — the raw file works.
5. **Compatibility with ROADMAP out-of-scope**: Is a JSONL event log a "notification system" or "infrastructure primitive"? This analysis argues the latter, but the distinction should be validated with project maintainers.
