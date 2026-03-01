# CLI Restructure Proposal

> **Status**: Draft proposal for discussion
> **Author**: cool-wolf worker
> **Aligns with**: ROADMAP P2 - Better onboarding

## Problem Statement

The multiclaude CLI has grown organically and now suffers from:

1. **Too many top-level commands** (28 entries)
2. **Redundant aliases** that pollute `--help` output
3. **Inconsistent naming** (`agent`/`agents`, `work`/`worker`)
4. **Unclear mental model** - is the tool repo-centric or agent-centric?

### Evidence: Current `--help` Output

```
Subcommands:
  repair          ← maintenance
  claude          ← agent context
  logs            ← agent ops
  status          ← system overview
  daemon          ← daemon management
  worker          ← agent creation
  work            ← ALIAS for worker
  agent           ← agent ops
  attach          ← ALIAS for agent attach
  review          ← agent creation
  config          ← repo config
  start           ← ALIAS for daemon start
  list            ← ALIAS for repo list
  workspace       ← agent creation
  refresh         ← agent ops
  docs            ← meta
  diagnostics     ← meta
  version         ← meta
  agents          ← agent definitions (confusing: singular vs plural)
  init            ← ALIAS for repo init
  cleanup         ← maintenance
  bug             ← meta
  stop-all        ← daemon control
  repo            ← repo management
  history         ← ALIAS for repo history
  message         ← agent comms
```

A new user sees 28 commands and has no idea where to start.

## Current Command Tree (Actual)

```
multiclaude
├── daemon
│   ├── start
│   ├── stop
│   ├── status
│   └── logs
├── repo
│   ├── init
│   ├── list
│   ├── rm
│   ├── use
│   ├── current
│   ├── unset
│   ├── history
│   └── hibernate
├── worker
│   ├── create (default)
│   ├── list
│   └── rm
├── workspace
│   ├── add
│   ├── rm
│   ├── list
│   └── connect (default)
├── agent
│   ├── attach
│   ├── complete
│   ├── restart
│   ├── send-message (alias)
│   ├── list-messages (alias)
│   ├── read-message (alias)
│   └── ack-message (alias)
├── agents
│   ├── list
│   ├── spawn
│   └── reset
├── message
│   ├── send
│   ├── list
│   ├── read
│   └── ack
├── logs
│   ├── list
│   ├── search
│   └── clean
├── start (alias → daemon start)
├── init (alias → repo init)
├── list (alias → repo list)
├── history (alias → repo history)
├── attach (alias → agent attach)
├── work (alias → worker)
├── status
├── stop-all
├── cleanup
├── repair
├── refresh
├── claude
├── review
├── config
├── docs
├── diagnostics
├── version
└── bug
```

**Issues by category:**

| Issue | Count | Examples |
|-------|-------|----------|
| Top-level aliases | 7 | `start`, `init`, `list`, `history`, `attach`, `work` |
| Nested aliases | 4 | `agent send-message` → `message send` |
| Singular/plural confusion | 1 | `agent` vs `agents` |
| Unclear grouping | 5 | `logs`, `refresh`, `status`, `claude`, `review` |

## Proposed Solutions

### Option A: Documentation-Only (Minimal Change)

**Change**: Improve help text and docs, no code changes.

**Approach**:
1. Rewrite `--help` to show "Getting Started" section first
2. Group commands visually in help output
3. Add `multiclaude quickstart` command that shows common workflows
4. Update COMMANDS.md with clearer structure

**Pros**: No breaking changes, fast to implement
**Cons**: Still confusing command surface, doesn't fix root cause

### Option B: Deprecation Warnings (Medium Change)

**Change**: Add deprecation warnings to aliases, document preferred commands.

**Approach**:
1. Aliases print `DEPRECATED: Use 'multiclaude repo init' instead`
2. Hide aliases from `--help` output (still work, just not shown)
3. Document migration path in COMMANDS.md
4. Remove aliases in v2.0

**Pros**: Gradual migration, preserves backward compat
**Cons**: Two releases needed, some user friction

### Option C: Restructure Verbs (Breaking Change)

**Change**: Consolidate commands under clear noun groups.

**Proposed structure**:
```
multiclaude
├── daemon (start, stop, status, logs)
├── repo (init, list, rm, use, config, history, hibernate)
├── agent (create, list, rm, attach, restart, complete)  ← merges worker+workspace
├── message (send, list, read, ack)
├── logs (view, list, search, clean)
├── status          ← comprehensive overview
├── refresh         ← sync all worktrees
├── cleanup         ← maintenance
├── repair          ← maintenance
├── version
├── help            ← enhanced help
```

**Key changes**:
- Merge `worker`, `workspace`, `agents` under `agent`
- Remove all top-level aliases
- `agent create "task"` replaces `worker create`
- `agent create --workspace` replaces `workspace add`

**Pros**: Clean, learnable, consistent
**Cons**: Breaking change, migration required

### Option D: Hybrid (Recommended)

**Change**: Implement Option B now, plan Option C for v2.0.

**Phase 1 (Now)**:
1. Hide aliases from `--help` (still work)
2. Group help output by category
3. Add `multiclaude guide` command with interactive walkthrough
4. Rename `agents` → `templates` (avoids `agent`/`agents` confusion)

**Phase 2 (v2.0)**:
1. Remove deprecated aliases
2. Optionally merge `worker`/`workspace` under `agent`

## Recommended Immediate Actions

### 1. Improve Help Output

Current:
```
Subcommands:
  repair          Repair state after crash
  claude          Restart Claude in current agent context
  ...
```

Proposed:
```
Multiclaude - orchestrate multiple Claude Code agents

QUICK START:
  multiclaude repo init <github-url>    Initialize a repository
  multiclaude worker "task"             Create a worker for a task
  multiclaude status                    See what's running

DAEMON:
  daemon start/stop/status/logs         Manage background process

REPOSITORIES:
  repo init/list/rm/use/history         Track and manage repos

AGENTS:
  worker create/list/rm                 Task-focused workers
  workspace add/list/rm/connect         Persistent workspaces
  agent attach/restart/complete         Agent operations

COMMUNICATION:
  message send/list/read/ack            Inter-agent messaging

MAINTENANCE:
  cleanup, repair, refresh              Fix and sync state
  logs, config, diagnostics             Inspect and configure

Run 'multiclaude <command> --help' for details.
```

### 2. Add `guide` Command

```bash
$ multiclaude guide

Welcome to multiclaude! Here's how to get started:

1. INITIALIZE A REPO
   multiclaude repo init https://github.com/you/repo

2. START THE DAEMON
   multiclaude start

3. CREATE A WORKER
   multiclaude worker "Fix the login bug"

4. WATCH IT WORK
   multiclaude agent attach <worker-name>

Need more? See: multiclaude docs
```

### 3. Hide Aliases from Help

In `cli.go`, add a `Hidden` field to Command:

```go
type Command struct {
    Name        string
    Description string
    Hidden      bool  // Don't show in --help
    ...
}

// Mark aliases as hidden
c.rootCmd.Subcommands["init"] = repoCmd.Subcommands["init"]
c.rootCmd.Subcommands["init"].Hidden = true
```

### 4. Rename `agents` → `templates`

The current naming creates confusion:
- `agent attach` - operate on running agent
- `agents list` - list agent definitions (templates)

Rename to:
- `templates list` - list agent templates
- `templates spawn` - spawn from template
- `templates reset` - reset to defaults

## Migration Path

| Current | Deprecated In | Removed In | Replacement |
|---------|---------------|------------|-------------|
| `multiclaude init` | v1.x | v2.0 | `multiclaude repo init` |
| `multiclaude list` | v1.x | v2.0 | `multiclaude repo list` |
| `multiclaude start` | v1.x | v2.0 | `multiclaude daemon start` |
| `multiclaude attach` | v1.x | v2.0 | `multiclaude agent attach` |
| `multiclaude work` | v1.x | v2.0 | `multiclaude worker` |
| `multiclaude history` | v1.x | v2.0 | `multiclaude repo history` |
| `multiclaude agents` | v1.x | v2.0 | `multiclaude templates` |

## Implementation Checklist

- [ ] Add `Hidden` field to Command struct
- [ ] Mark aliases as hidden
- [ ] Restructure help output with categories
- [ ] Add `guide` command
- [ ] Rename `agents` → `templates`
- [ ] Update COMMANDS.md
- [ ] Update embedded prompts
- [ ] Add deprecation warnings to aliases
- [ ] Update tests

## Questions for Review

1. **Keep `start` alias?** It's the most commonly used shortcut.
2. **Merge worker/workspace?** They're conceptually similar (agent instances).
3. **Add `quickstart` or `guide`?** Which name is clearer?
4. **Timeline for v2.0?** When do we remove deprecated aliases?

---

*Generated by cool-wolf worker analyzing CLI structure.*
