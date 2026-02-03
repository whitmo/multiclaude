# Multiclaude Diagrams

Visual diagrams for understanding multiclaude's architecture and data flows.

## System Overview

```mermaid
flowchart TB
    subgraph User["User's Machine"]
        CLI[CLI Client]

        subgraph Daemon["Daemon Process"]
            Socket[Socket Server]
            Health[Health Check<br/>2 min cycle]
            Router[Message Router<br/>2 min cycle]
            Wake[Wake/Nudge<br/>2 min cycle]
            State[(state.json)]
        end

        subgraph Tmux["tmux session: mc-repo"]
            Sup[supervisor]
            MQ[merge-queue]
            W1[worker-1]
            W2[worker-2]
        end

        subgraph Worktrees["Git Worktrees"]
            WT1[wts/repo/supervisor]
            WT2[wts/repo/merge-queue]
            WT3[wts/repo/worker-1]
            WT4[wts/repo/worker-2]
        end
    end

    CLI <-->|Unix Socket| Socket
    Socket <--> State
    Health --> Tmux
    Router --> Tmux
    Wake --> Tmux

    Sup --> WT1
    MQ --> WT2
    W1 --> WT3
    W2 --> WT4
```

## The Brownian Ratchet

The core philosophy: chaos creates progress when filtered through CI.

```mermaid
flowchart LR
    subgraph Chaos["Agent Activity (Chaotic)"]
        WA[Worker A<br/>auth feature]
        WB[Worker B<br/>auth feature]
        WC[Worker C<br/>bugfix #42]
    end

    subgraph Ratchet["CI Gate"]
        CI{CI Passes?}
    end

    subgraph Progress["Main Branch"]
        Main[████████████<br/>irreversible progress]
    end

    WA -->|PR| CI
    WB -->|PR| CI
    WC -->|PR| CI

    CI -->|pass| Main
    CI -->|fail| Retry[retry or<br/>spawn fixup]
    Retry --> Chaos
```

## Agent Types and Relationships

```mermaid
flowchart TB
    Human[Human User]

    subgraph Agents
        WS[Workspace<br/>user interactive]
        SUP[Supervisor<br/>coordination]
        MQ[Merge Queue<br/>the ratchet]
        W1[Worker 1]
        W2[Worker 2]
        REV[Review Agent]
    end

    subgraph External
        GH[(GitHub)]
        CI[CI System]
    end

    Human <-->|direct input| WS
    Human -->|spawns| SUP
    Human -->|spawns| MQ

    WS -->|spawn| W1
    SUP -->|guidance| W1
    SUP -->|guidance| W2

    W1 -->|PR| GH
    W2 -->|PR| GH

    MQ -->|monitor| GH
    MQ -->|spawn| REV
    MQ -->|merge| GH

    REV -->|comments| GH

    GH --> CI
    CI -->|status| GH
```

## Worker Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: multiclaude work "task"

    Created --> Working: Claude starts

    state Working {
        [*] --> Coding
        Coding --> Testing
        Testing --> Coding: tests fail
        Testing --> PR: tests pass
        PR --> [*]
    }

    Working --> Completing: multiclaude agent complete
    Completing --> MarkedForCleanup: Daemon marks ReadyForCleanup
    MarkedForCleanup --> NotifySupervisor: Daemon notifies
    NotifySupervisor --> NotifyMergeQueue
    NotifyMergeQueue --> HealthCheckCleanup: Health check runs
    HealthCheckCleanup --> [*]: Kill window, remove worktree
```

## Worker Creation Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Daemon
    participant Git
    participant Tmux
    participant Claude

    User->>CLI: multiclaude work "Add tests"
    CLI->>Daemon: list_agents (socket)
    Daemon-->>CLI: agent list

    CLI->>Git: worktree add
    Git-->>CLI: worktree created

    CLI->>Tmux: new-window
    Tmux-->>CLI: window created

    CLI->>CLI: write prompt file

    CLI->>Tmux: send-keys (start Claude)
    Tmux->>Claude: launch

    CLI->>Tmux: send-keys (task message)

    CLI->>Daemon: add_agent (worker)
    Daemon->>Daemon: save state
    Daemon-->>CLI: success

    CLI-->>User: Worker 'clever-fox' created
```

## Message Routing

```mermaid
sequenceDiagram
    participant AgentA as Worker A
    participant FS as Filesystem
    participant Daemon
    participant Tmux
    participant AgentB as Supervisor

    AgentA->>FS: write message.json<br/>(status: pending)

    Note over Daemon: 2 min later<br/>message router runs

    Daemon->>FS: scan messages dir
    FS-->>Daemon: pending messages

    Daemon->>Tmux: send-keys (literal mode)
    Tmux->>AgentB: paste message

    Daemon->>FS: update status<br/>(pending → delivered)
```

## Message Status Flow

```mermaid
stateDiagram-v2
    [*] --> pending: Message created
    pending --> delivered: Daemon sends via tmux
    delivered --> read: Agent sees message
    read --> acked: Agent acknowledges
    acked --> [*]: Message deleted
```

## Repository Initialization

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Daemon
    participant Git
    participant Tmux
    participant FS as Filesystem

    User->>CLI: multiclaude init github.com/org/repo

    CLI->>Daemon: ping
    Daemon-->>CLI: pong

    CLI->>Git: clone repo
    Git-->>CLI: cloned

    CLI->>Tmux: new-session mc-repo
    CLI->>Tmux: new-window supervisor
    CLI->>Tmux: new-window merge-queue

    CLI->>FS: write prompt files

    CLI->>Tmux: send-keys (start Claude)

    CLI->>Daemon: add_repo
    Daemon->>FS: save state
    Daemon-->>CLI: success

    CLI->>Daemon: add_agent (supervisor)
    CLI->>Daemon: add_agent (merge-queue)

    CLI-->>User: Repository initialized
```

## Health Check Loop

```mermaid
flowchart TD
    Start[Health Check Starts<br/>every 2 min] --> CheckSession{tmux session<br/>exists?}

    CheckSession -->|no| TryRestore[Attempt Restoration]
    TryRestore -->|success| CheckAgents
    TryRestore -->|fail| MarkCleanup[Mark agents<br/>for cleanup]

    CheckSession -->|yes| CheckAgents[Check each agent]

    CheckAgents --> WindowExists{tmux window<br/>exists?}

    WindowExists -->|no| RemoveAgent[Remove from state]
    WindowExists -->|yes| CheckReady{ReadyForCleanup?}

    CheckReady -->|yes| Cleanup[Kill window<br/>Remove worktree<br/>Clean messages]
    CheckReady -->|no| NextAgent[Check next agent]

    Cleanup --> NextAgent
    RemoveAgent --> NextAgent

    NextAgent --> PruneOrphans[Prune orphaned<br/>worktrees & messages]
    PruneOrphans --> End[Wait 2 min]
    End --> Start
```

## Daemon Goroutines

```mermaid
flowchart LR
    subgraph Daemon
        Main[main goroutine]

        subgraph Loops["Background Loops"]
            Server[serverLoop<br/>continuous]
            Health[healthCheckLoop<br/>2 min]
            Router[messageRouterLoop<br/>2 min]
            Wake[wakeLoop<br/>2 min]
        end

        WG[sync.WaitGroup]
        Ctx[context.Context]
    end

    Main -->|spawn| Server
    Main -->|spawn| Health
    Main -->|spawn| Router
    Main -->|spawn| Wake

    WG -.->|tracks| Loops
    Ctx -.->|cancels| Loops
```

## File System Layout

```mermaid
flowchart TB
    subgraph Home["~/.multiclaude/"]
        PID[daemon.pid]
        Sock[daemon.sock]
        Log[daemon.log]
        State[state.json]

        subgraph Prompts["prompts/"]
            P1[supervisor.md]
            P2[merge-queue.md]
            P3[worker-name.md]
        end

        subgraph Repos["repos/"]
            R1[my-repo/]
        end

        subgraph WTS["wts/"]
            subgraph WTRepo["my-repo/"]
                WT1[supervisor/]
                WT2[merge-queue/]
                WT3[clever-fox/]
            end
        end

        subgraph Msgs["messages/"]
            subgraph MsgRepo["my-repo/"]
                MS[supervisor/]
                MMQ[merge-queue/]
                MW[clever-fox/]
            end
        end

        subgraph Config["claude-config/"]
            subgraph CfgRepo["my-repo/"]
                subgraph Agent["clever-fox/"]
                    Cmds[commands/]
                end
            end
        end
    end
```

## Shutdown Sequence

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Daemon
    participant Goroutines
    participant FS as Filesystem

    User->>CLI: multiclaude stop-all
    CLI->>Daemon: stop (socket)

    Daemon->>Daemon: d.cancel()
    Daemon->>Goroutines: context cancelled

    par All goroutines
        Goroutines->>Goroutines: check ctx.Done()
        Goroutines->>Goroutines: return
    end

    Daemon->>Daemon: d.wg.Wait()
    Note over Daemon: All goroutines stopped

    Daemon->>Daemon: d.server.Stop()
    Daemon->>FS: d.state.Save()
    Daemon->>FS: d.pidFile.Remove()

    Daemon-->>CLI: success
    CLI-->>User: Daemon stopped
```

## State Thread Safety

```mermaid
flowchart TB
    subgraph State["state.State"]
        Mutex[sync.RWMutex]
        Data[Repos map]
    end

    subgraph Readers["Read Operations"]
        R1[GetRepo]
        R2[GetAgent]
        R3[ListAgents]
    end

    subgraph Writers["Write Operations"]
        W1[AddRepo]
        W2[AddAgent]
        W3[RemoveAgent]
    end

    R1 -->|RLock| Mutex
    R2 -->|RLock| Mutex
    R3 -->|RLock| Mutex

    W1 -->|Lock| Mutex
    W2 -->|Lock| Mutex
    W3 -->|Lock| Mutex

    Mutex --> Data

    W1 -->|auto-save| Disk[(state.json)]
    W2 -->|auto-save| Disk
    W3 -->|auto-save| Disk
```
