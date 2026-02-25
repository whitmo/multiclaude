# Quickstart Guide

Get multiclaude running in 5 minutes.

## Prerequisites

You need:
- Go 1.21+
- tmux
- git
- GitHub CLI (`gh`) - authenticated via `gh auth login`

## Install

```bash
go install github.com/dlorenc/multiclaude/cmd/multiclaude@latest
```

## Start the Daemon

multiclaude runs a background daemon that coordinates agents.

```bash
multiclaude start
```

Check it's running:

```bash
multiclaude daemon status
```

## Initialize a Repository

Point multiclaude at a GitHub repository you want to work on:

```bash
multiclaude init https://github.com/your/repo
```

This:
- Clones the repository
- Creates a tmux session (`mc-repo`)
- Spawns a supervisor agent
- Spawns a merge-queue agent
- Creates a default workspace for you

## Spawn a Worker

Create a worker to do a task:

```bash
multiclaude work "Add unit tests for the auth module"
```

The worker gets a fun name like `happy-platypus` and starts working
immediately.

## Watch Agents Work

Attach to the tmux session to see all agents:

```bash
tmux attach -t mc-repo
```

Navigate between agent windows:
- `Ctrl-b n` - next window
- `Ctrl-b p` - previous window
- `Ctrl-b w` - window picker
- `Ctrl-b d` - detach (agents keep running)

Or attach to a specific agent:

```bash
multiclaude attach happy-platypus --read-only
```

## Connect to Your Workspace

Your workspace is a persistent Claude session where you interact with the
codebase:

```bash
multiclaude workspace connect default
```

From here you can spawn more workers, check status, and manage your
development flow.

## Check Status

```bash
multiclaude status          # Overall status
multiclaude work list       # List active workers
multiclaude repo history    # What workers have done
```

## Stop Everything

When you're done:

```bash
multiclaude stop-all        # Stop daemon and all agents
```

Or just stop the daemon (agents keep running in tmux):

```bash
multiclaude daemon stop
```

## Next Steps

- Read the [README](../README.md) for the full philosophy and feature list
- See [CRASH_RECOVERY.md](CRASH_RECOVERY.md) for what to do when things go
  wrong
- Check [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) to understand where
  files live

## Common Issues

**"daemon is not running"**

Start it: `multiclaude start`

**"repository already initialized"**

You've already set up this repo. Use `multiclaude list` to see tracked repos.

**"gh auth error"**

Authenticate GitHub CLI: `gh auth login`

**tmux not found**

Install tmux: `brew install tmux` (macOS) or `apt install tmux` (Linux)
