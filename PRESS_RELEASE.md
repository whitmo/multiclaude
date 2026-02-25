# multiclaude: A Brownian Ratchet for AI-Assisted Development

**Lightweight orchestration for multiple Claude Code agents**

## The Problem

Modern AI coding assistants are powerful, but they work one task at a time.
When you have a backlog of issues, tests to write, and documentation to
update, you wait. Each task queues behind the last.

What if you could parallelize?

## The Solution

**multiclaude** is a lightweight orchestrator that runs multiple Claude Code
agents simultaneously on your GitHub repository. Each agent works in its own
isolated environment, and you can spawn as many as you need.

```bash
multiclaude init https://github.com/your/repo
multiclaude work "Add unit tests for auth"
multiclaude work "Fix issue #42"
multiclaude work "Update API documentation"
```

Three tasks. Three agents. Running in parallel.

## The Philosophy: Brownian Ratchet

In physics, a Brownian ratchet converts random molecular motion into
directed movement through a mechanism that allows motion in only one
direction.

multiclaude applies this principle to software development:

**The Chaos**: Multiple agents work simultaneously. They may duplicate
effort, create conflicts, or produce suboptimal solutions. This is fine.
More attempts mean more chances for progress.

**The Ratchet**: CI is the arbiter. If tests pass, the code merges. Every
merged PR clicks forward one notch. Progress is permanent.

This approach optimizes for throughput of successful changes, not efficiency
of individual agents. Redundant work is cheaper than blocked work.

## Key Features

**Observable**: All agents run in tmux windows. Attach anytime to watch them
work or intervene when needed.

**Isolated**: Each agent gets its own git worktree. No interference between
parallel tasks.

**Self-Healing**: The daemon monitors agent health, restarts crashed
processes, and cleans up finished work.

**Simple**: Filesystem for state. Tmux for visibility. Git for isolation. No
databases, no cloud dependencies, no complex setup.

## How It Works

multiclaude spawns three types of agents:

1. **Supervisor** - Coordinates agents, helps stuck workers, tracks progress
2. **Workers** - Execute tasks, create pull requests
3. **Merge Queue** - Monitors PRs, merges when CI passes

Workers communicate via filesystem-based messages. The supervisor nudges
stuck agents. The merge queue ensures only passing code lands.

## Remote-First Design

Unlike tools designed for solo development, multiclaude treats software
engineering as an MMORPG:

- Your **workspace** is your persistent home base
- **Workers** are party members you spawn for quests
- The **supervisor** coordinates the guild
- The **merge queue** is the raid boss guarding main

The system keeps running when you're away. Spawn workers before lunch.
Review their PRs when you return.

## Comparison with Gastown

multiclaude was developed independently but shares goals with Steve Yegge's
Gastown project. Both orchestrate multiple Claude Code instances using tmux
and git worktrees.

**Where they differ:**

| Aspect | multiclaude | Gastown |
|--------|-------------|---------|
| Agent model | 3 roles | 7 roles |
| Philosophy | Minimal, Unix-style | Comprehensive orchestration |
| State | JSON + filesystem | Git-backed hooks |
| Target | Remote-first teams | Solo development |

Choose multiclaude for simplicity. Choose Gastown for sophisticated features
like work swarming and structured work units.

## Getting Started

```bash
# Install
go install github.com/dlorenc/multiclaude/cmd/multiclaude@latest

# Start the daemon
multiclaude start

# Initialize a repository
multiclaude init https://github.com/your/repo

# Spawn a worker
multiclaude work "Your task here"

# Watch it work
tmux attach -t mc-repo
```

## Requirements

- Go 1.21+
- tmux
- git
- GitHub CLI (authenticated)

## Open Source

multiclaude is MIT licensed and available on GitHub:

https://github.com/dlorenc/multiclaude

Contributions welcome. Issues and PRs are the ratchet - if CI passes, it
ships.

## About

multiclaude embraces a counterintuitive truth: in AI-assisted development,
chaos is fine as long as you ratchet forward. Perfect coordination is
expensive and fragile. Multiple imperfect attempts that occasionally succeed
beat waiting for one perfect solution.

Let the agents work. Let CI judge. Click the ratchet forward.

---

*For more information, see the project documentation or open an issue on
GitHub.*
