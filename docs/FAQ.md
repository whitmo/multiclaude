# Frequently Asked Questions

## General

### What is multiclaude?

multiclaude is a lightweight orchestrator for running multiple Claude Code
agents on GitHub repositories. Each agent runs in its own tmux window with
an isolated git worktree.

### How is it different from Gastown?

Both solve multi-agent orchestration for Claude Code. multiclaude aims for
Unix-style simplicity with fewer concepts: 3 agent roles vs 7, filesystem
state vs git-backed hooks, minimal dependencies. Gastown offers more
sophisticated features like work swarming and the Beads framework. See the
[README comparison](../README.md#gastown-and-multiclaude) for details.

### What are the prerequisites?

- Go 1.21+
- tmux (terminal multiplexer)
- git
- GitHub CLI (`gh`) authenticated via `gh auth login`

## Agents

### What types of agents are there?

- **Supervisor**: Coordinates all agents, answers status questions, helps
  stuck workers
- **Workers**: Execute specific tasks, create PRs when done
- **Merge Queue**: Monitors PRs, merges when CI passes
- **Workspace**: Your interactive Claude session for spawning workers and
  managing flow

### How do agents communicate?

Via filesystem-based JSON messages in `~/.multiclaude/messages/`. The daemon
routes messages and periodically nudges agents to check their inbox.

### What happens when an agent crashes?

The daemon's health check (every 2 minutes) detects dead agents and attempts
to restart them with `--resume` to preserve session context. See
[CRASH_RECOVERY.md](CRASH_RECOVERY.md) for details.

### Can I have multiple workers?

Yes. Spawn as many as you want:

```bash
multiclaude work "Task 1"
multiclaude work "Task 2"
multiclaude work "Task 3"
```

They work in parallel, each in their own git worktree.

## Workflow

### How do I see what agents are doing?

```bash
# Attach to the tmux session (all agents)
tmux attach -t mc-repo

# Attach to a specific agent
multiclaude attach happy-platypus --read-only
```

### How do I check worker status?

```bash
multiclaude work list          # Active workers
multiclaude repo history       # Completed tasks and PRs
multiclaude status             # Overall system status
```

### How do I stop a worker?

```bash
multiclaude work rm happy-platypus
```

This warns if there's uncommitted work.

### How do I iterate on an existing PR?

Use `--push-to` to have a worker push to an existing branch:

```bash
multiclaude work "Fix review comments" --branch origin/work/fox --push-to work/fox
```

## Troubleshooting

### "daemon is not running"

Start it:

```bash
multiclaude start
```

### "repository already initialized"

The repo is already tracked. Check with:

```bash
multiclaude list
```

### "permission denied" on socket

The daemon socket may have wrong permissions. Try:

```bash
multiclaude stop-all
multiclaude start
```

### Agent seems stuck

1. Check if Claude is waiting for input:
   ```bash
   multiclaude attach <agent> --read-only
   ```

2. Send it a nudge message:
   ```bash
   multiclaude agent send-message <agent> "Status update?"
   ```

3. The supervisor also periodically nudges stuck agents.

### Uncommitted work in a dead worker

```bash
# Check the worktree
cd ~/.multiclaude/wts/<repo>/<worker-name>
git status

# Save the work
git add . && git commit -m "WIP"
git push -u origin work/<worker-name>
```

### How do I reset everything?

```bash
multiclaude stop-all --clean
```

This stops all agents and removes state files (but preserves repo clones and
worktrees).

## Configuration

### Can I customize agent behavior?

Yes. Add files to `.multiclaude/` in your repository:

- `SUPERVISOR.md` - Additional instructions for supervisor
- `WORKER.md` - Additional instructions for workers
- `REVIEWER.md` - Additional instructions for merge queue
- `hooks.json` - Claude Code hooks configuration

### Where does multiclaude store data?

Everything lives in `~/.multiclaude/`:

```
~/.multiclaude/
├── daemon.pid          # Daemon process ID
├── daemon.sock         # Unix socket
├── daemon.log          # Daemon logs
├── state.json          # All state
├── repos/<repo>/       # Cloned repositories
├── wts/<repo>/         # Git worktrees
└── messages/<repo>/    # Agent messages
```

See [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) for full details.

### How do I view daemon logs?

```bash
multiclaude daemon logs -f     # Follow logs
tail -f ~/.multiclaude/daemon.log
```

## Philosophy

### Why "Brownian Ratchet"?

In physics, a Brownian ratchet converts random motion into directed movement
through a mechanism that only allows forward motion. multiclaude applies
this: multiple agents create "chaos" (parallel, potentially overlapping
work), but CI acts as the ratchet - only passing code merges, ensuring
permanent forward progress.

### Why embrace chaos instead of coordination?

Trying to perfectly coordinate agent work is expensive and fragile. Instead:
- Redundant work is cheaper than blocked work
- Failed attempts cost nothing; only successful attempts matter
- CI is the arbiter - if it passes, the code is good
- More agents mean more chaos but also more forward motion

### Why can't agents merge without CI?

CI is King. Agents are forbidden from weakening CI or bypassing checks.
This ensures the ratchet never slips backward.
