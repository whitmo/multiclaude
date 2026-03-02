# Event Architecture for Claude Code: A Proposal

**Author**: proud-bear (multiclaude worker)
**Date**: 2026-03-01
**Status**: Draft proposal for discussion
**Context**: Informed by multiclaude orchestration experience and Gemini's analysis of event notification options

---

## Executive Summary

Claude Code has excellent *reactive* patterns (hooks responding to its own lifecycle) but no *proactive* patterns (external systems pushing events into sessions). This document proposes a minimal event architecture that would enable orchestrators like multiclaude to communicate with Claude Code sessions natively, replacing the current tmux send-keys hack.

The core idea: **Claude Code sessions should be able to receive and react to external events between agentic turns**, enabling a "director" pattern where a long-lived session acts as an intelligent event processor.

---

## The Problem

### Current State

Claude Code's interaction model is purely request-response:

| Mechanism | Direction | Limitation |
|-----------|-----------|------------|
| **Tools** | Claude -> external | Synchronous, Claude-initiated only |
| **MCP servers** | Claude -> server | Pull-only; servers provide tools Claude calls |
| **Hooks** | Claude lifecycle -> scripts | React to internal events only; cannot be triggered externally |
| **Skills** | User/Claude -> skill | Cannot be triggered by external processes |

### The tmux Hack

Multiclaude's entire agent communication system is built on injecting text into Claude Code sessions via tmux:

```go
// The only way to "push" to Claude Code today
d.tmux.SendKeysLiteralWithEnter(ctx, session, window, messageText)
```

This works but is:
- **Fragile**: Race conditions between paste-buffer and Enter key (Issue #63 required atomic shell chaining)
- **Unstructured**: Messages are raw text, not typed events
- **Blind**: No acknowledgment that Claude received or processed the message
- **Platform-dependent**: Requires tmux; doesn't work in IDE integrations
- **Timing-dependent**: Messages can arrive mid-thought, corrupting Claude's reasoning

### What We Actually Need

An orchestrator needs to:
1. **Push structured events** to a Claude session (agent completed, PR merged, health check failed)
2. **Know the event was received** (acknowledgment)
3. **Have Claude react appropriately** (dispatch work, review results, escalate)
4. **Do this without tmux** (works in any Claude Code environment)

---

## Proposed Architecture

### Design Principles

1. **Minimal surface area**: One new primitive, not a framework
2. **Pull semantics for push delivery**: Events queue; Claude processes them between turns
3. **File-based where possible**: Leverage the filesystem (Claude Code's natural habitat)
4. **No persistent connections**: Avoid WebSocket/SSE complexity
5. **Works with existing hooks**: Extend, don't replace

### The Primitive: Session Inbox

A session inbox is a directory where external processes drop event files that Claude Code picks up between agentic turns.

```
~/.claude/sessions/<session-id>/inbox/
  ├── 001-1709312400-agent_complete.json
  ├── 002-1709312460-pr_merged.json
  └── 003-1709312520-message.json
```

Each event file:
```json
{
  "id": "evt_abc123",
  "type": "agent_complete",
  "source": "multiclaude/supervisor",
  "timestamp": "2026-03-01T12:00:00Z",
  "payload": {
    "agent": "worker-1",
    "task": "Fix authentication bug",
    "pr": "https://github.com/org/repo/pull/42",
    "status": "success"
  },
  "priority": "normal"
}
```

**Claude Code behavior**:
- Between agentic turns (after tool results, before next reasoning step), check inbox
- Present queued events as structured context: "You have N pending events"
- Claude decides how to react based on its system prompt and skills
- Processed events are moved to `inbox/processed/` or deleted

### Three Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Layer 3: Director Pattern                  │
│   Long-lived Claude session + skills = intelligent reactor   │
│   Skills register as event handlers via frontmatter          │
└──────────────────────────┬──────────────────────────────────┘
                           │ uses
┌──────────────────────────▼──────────────────────────────────┐
│                    Layer 2: Event Routing                     │
│   MCP servers push events; hooks bridge external sources     │
│   Events are typed, routed to skills or conversation         │
└──────────────────────────┬──────────────────────────────────┘
                           │ writes to
┌──────────────────────────▼──────────────────────────────────┐
│                    Layer 1: Session Inbox                     │
│   Filesystem directory; external processes write event JSON  │
│   Claude Code reads between turns                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Layer 1: Session Inbox (The Foundation)

### Specification

**Directory**: `~/.claude/sessions/<session-id>/inbox/`

**Event file format**: `<sequence>-<timestamp>-<type>.json`

**Claude Code changes**:
1. After each tool use result (and before generating the next response), scan inbox
2. If events exist, prepend them to Claude's context as a structured block:

```
<pending-events count="2">
<event id="evt_abc" type="agent_complete" source="multiclaude" priority="normal">
Worker "worker-1" completed task "Fix auth bug". PR: #42 (merged).
</event>
<event id="evt_def" type="message" source="multiclaude/supervisor" priority="high">
New task available: "Add rate limiting to API endpoints"
</event>
</pending-events>
```

3. Claude processes events as part of its normal reasoning
4. A new built-in tool `AcknowledgeEvent` marks events as processed

**Why filesystem**:
- No daemon, no socket, no persistent connection
- Works in any environment (terminal, IDE, SSH)
- Crash-safe: unprocessed events survive restarts
- Debuggable: `ls` and `cat` are your monitoring tools
- Aligns with Claude Code's existing patterns (session JSONL files are already filesystem-based)

### External Process API

Any process can write to the inbox:

```bash
# Simple: just write a JSON file
cat > ~/.claude/sessions/$SESSION_ID/inbox/001-$(date +%s)-task_assigned.json << 'EOF'
{
  "id": "evt_001",
  "type": "task_assigned",
  "source": "multiclaude",
  "timestamp": "2026-03-01T12:00:00Z",
  "payload": {"task": "Fix the login bug", "priority": "high"}
}
EOF
```

Or via a helper:
```bash
claude-event send --session $SESSION_ID --type task_assigned --payload '{"task": "Fix login"}'
```

### Session ID Discovery

For orchestrators to find the right inbox, Claude Code should expose session IDs:

```bash
# New: query running sessions
claude --list-sessions
# Output: session_id  pid  working_dir  started_at
# abc123    42000  /path/to/repo  2026-03-01T12:00:00Z
```

Or via a well-known file:
```
~/.claude/sessions/active.json
[
  {"session_id": "abc123", "pid": 42000, "cwd": "/path/to/repo"}
]
```

---

## Layer 2: Event Routing via MCP and Hooks

### MCP Server Event Push

The MCP specification includes a `notifications` concept. Claude Code could support a new notification type from MCP servers:

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/event",
  "params": {
    "type": "agent_complete",
    "payload": {
      "agent": "worker-1",
      "status": "success"
    }
  }
}
```

**Claude Code behavior**: When an MCP server sends `notifications/event`, write it to the session inbox. This converts MCP push notifications into the inbox primitive.

**Why this matters**: MCP servers are already long-lived processes connected to Claude Code. They're the natural bridge for external event sources. An orchestrator MCP server could:

```
┌─────────────┐     MCP stdio      ┌─────────────┐
│ Claude Code  │◄──────────────────►│ Orchestrator │
│   session    │   notifications/   │  MCP Server  │
│              │   event            │              │
└─────────────┘                     └──────┬──────┘
                                           │
                                    watches/polls
                                           │
                                    ┌──────▼──────┐
                                    │ Orchestrator │
                                    │   daemon     │
                                    └─────────────┘
```

### Hook-Based Event Bridge

A new hook event `InboxEvent` fires when events arrive in the inbox:

```json
{
  "hooks": {
    "InboxEvent": [
      {
        "type": "command",
        "command": "~/.claude/hooks/route-event.sh",
        "description": "Route events to skills or filter them"
      }
    ]
  }
}
```

The hook receives the event JSON on stdin and can:
- **Allow**: Event reaches Claude (exit 0)
- **Block**: Event is silently discarded (exit 2, like PreToolUse)
- **Transform**: Modify event content via stdout
- **Route**: Set `"skill": "handle-deployment"` in output to auto-invoke a skill

---

## Layer 3: Skills as Event Handlers (The Director Pattern)

### Skill Event Registration

Skills declare which events they handle via frontmatter:

```yaml
---
name: handle-agent-complete
description: Process completed agent work
handles-events:
  - agent_complete
  - agent_failed
user-invocable: false
---

# Handle Agent Completion

When an agent completes its task:

1. Read the PR details from the event payload
2. Check if the PR was merged successfully
3. If merged, check ROADMAP.md for next priority items
4. If there are available tasks, assign one to the next idle worker
5. If failed, analyze the failure and decide whether to retry or escalate

## Event Schema

- `event.payload.agent`: Name of the completed agent
- `event.payload.pr`: PR URL (if created)
- `event.payload.status`: "success" | "failure" | "cancelled"
- `event.payload.summary`: Agent's completion summary
```

**Claude Code behavior**: When an inbox event's type matches a skill's `handles-events`, auto-invoke that skill with the event as context. This is analogous to how skills are currently auto-invoked based on conversation context, but triggered by events instead.

### The Director Session

A "director" is just a long-lived Claude Code session with:
1. A system prompt defining its orchestration role
2. Skills registered as event handlers
3. An MCP server connecting it to the orchestrator

```
Director Session
├── System prompt: "You are an orchestrator directing parallel agents..."
├── Skills:
│   ├── handle-agent-complete (handles: agent_complete, agent_failed)
│   ├── handle-pr-merged (handles: pr_merged, pr_closed)
│   ├── handle-health-alert (handles: health_check_failed)
│   └── assign-task (handles: task_available)
└── MCP Server: orchestrator-bridge
    └── Pushes events from daemon into session inbox
```

**How it replaces tmux injection**:

| Current (multiclaude) | Proposed (director) |
|---|---|
| Daemon writes message JSON to filesystem | Daemon writes event JSON to filesystem |
| Daemon router loop sends via `tmux send-keys` | MCP server (or inbox watcher) delivers to Claude |
| Claude sees raw text injected at cursor | Claude sees structured event between turns |
| No acknowledgment mechanism | `AcknowledgeEvent` tool confirms processing |
| Timing-dependent, can corrupt reasoning | Queue-based, processed at safe boundaries |
| tmux-only | Works in any Claude Code environment |

### Director Lifecycle

```
1. Orchestrator starts director Claude session
   claude --session-id director-001 \
     --append-system-prompt-file director.md \
     --mcp-server orchestrator-bridge

2. Director session starts, loads skills
   Skills register event handlers

3. Event arrives (e.g., worker completed)
   → MCP server receives from daemon
   → Pushes notification/event to Claude Code
   → Claude Code writes to session inbox
   → Between turns, Claude sees event
   → Matching skill auto-invoked
   → Director reasons about next action
   → Calls tools (assign task, send message, etc.)

4. Director acknowledges event
   → AcknowledgeEvent tool marks as processed
   → Orchestrator knows event was handled
```

---

## Implementation Phases

### Phase 1: Session Inbox (Minimal, standalone)

**Changes to Claude Code**:
- Create inbox directory on session start
- Scan inbox between agentic turns
- Present events as structured context
- Add `AcknowledgeEvent` built-in tool
- Expose session IDs via `active.json` or `--list-sessions`

**Effort**: Small. No new protocols, no persistent connections.
**Value**: External processes can already push events to Claude Code without tmux.

### Phase 2: MCP Event Notifications

**Changes to Claude Code**:
- Handle `notifications/event` from MCP servers
- Write received notifications to session inbox
- Standard event schema

**Effort**: Medium. Requires MCP notification handling.
**Value**: MCP servers become bidirectional; orchestrators get a clean integration point.

### Phase 3: Skill Event Handlers

**Changes to Claude Code**:
- `handles-events` frontmatter in skills
- Event-to-skill routing logic
- Event context injection into skill invocation

**Effort**: Medium. Extends existing skill auto-invocation.
**Value**: Enables the full director pattern; skills become reactive components.

---

## What This Enables for Multiclaude

### Today: Daemon-Driven Polling Loops

```
Daemon (Go)
├── Health check loop (2 min) → tmux send-keys "/status"
├── Message router loop (2 min) → tmux send-keys "<message text>"
├── Wake loop (2 min) → tmux send-keys "/status"
└── All coordination logic in Go daemon
```

### Tomorrow: Director-Driven Event Processing

```
Director Session (Claude Code)
├── Receives: agent_complete → assigns next task
├── Receives: pr_merged → updates roadmap tracking
├── Receives: health_alert → investigates and repairs
├── Receives: message → routes to appropriate agent
└── All coordination logic in natural language (skills)
```

**What gets simpler in multiclaude**:
- Daemon becomes a thin event emitter, not a coordination engine
- No more tmux send-keys for communication
- No more polling loops; events are push-based
- Agent intelligence moves from Go code to Claude's reasoning
- Works in IDE integrations (VS Code, JetBrains) not just terminal

**What stays the same**:
- Filesystem-based state (JSONL events instead of JSON messages)
- Local-first (no cloud dependencies)
- Claude-only (events processed by Claude Code)

---

## Open Questions

1. **Turn boundaries**: When exactly between turns should Claude check the inbox? After every tool result? Only when idle? Configurable?

2. **Priority and ordering**: Should high-priority events interrupt current work? Or always queue?

3. **Event schema standardization**: Should Claude Code define standard event types, or leave it fully custom?

4. **Backpressure**: What happens when events arrive faster than Claude can process them? Queue limits? Oldest-dropped?

5. **Multi-session events**: Can an event target multiple sessions? Broadcast patterns?

6. **Security**: Who can write to a session's inbox? Should there be an allowlist of event sources?

7. **IDE integration**: How does this work in VS Code / JetBrains extensions? The inbox directory approach is environment-agnostic, but event presentation UX differs.

8. **Stop hook interaction**: Currently multiclaude uses the `Stop` hook to inject messages when Claude tries to stop. With an inbox, the Stop hook could check for pending events and say "don't stop, you have events to process." How should these interact?

---

## Relationship to Existing Analysis

The Gemini analysis at `/tmp/gemini-events-analysis.md` evaluated event notification options **for multiclaude's daemon**. That analysis correctly identified the append-only JSONL log as the best fit for multiclaude's constraints.

This proposal is complementary but different: it's about **Claude Code itself** gaining event-receiving capability. The two work together:

```
multiclaude daemon → events.jsonl (Gemini's recommendation: daemon emits events)
                  ↓
orchestrator MCP server → reads events.jsonl, pushes to Claude Code inbox
                        ↓
Claude Code session → receives structured events, processes via skills
```

The JSONL log is the daemon's output format. The session inbox is Claude Code's input format. The MCP server bridges them.

---

## Summary

| Layer | What | Who Changes | Effort |
|-------|------|------------|--------|
| Session Inbox | File-based event queue per session | Claude Code (Anthropic) | Small |
| MCP Event Push | `notifications/event` from MCP servers | Claude Code + MCP spec | Medium |
| Skill Event Handlers | `handles-events` in skill frontmatter | Claude Code (Anthropic) | Medium |
| Director Pattern | Long-lived session + skills + events | Orchestrators (multiclaude) | Medium |

The key insight: Claude Code doesn't need a complex event bus or pub-sub system. It needs **one simple primitive** - a file-based inbox that it checks between turns - and the rest builds naturally on existing mechanisms (MCP, hooks, skills).

This is the minimum viable event architecture: filesystem-native, no persistent connections, no new protocols, and it turns Claude Code from a request-response tool into an event-driven agent.
