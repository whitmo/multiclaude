# Worker Lifecycle Audit

This document identifies gaps in the worker lifecycle where workers might fail
to start, complete, or clean up properly.

**Audit Date:** 2026-02-03
**Reviewer:** calm-panda (worker agent)
**Files Reviewed:**
- `internal/cli/cli.go` (createWorker, completeWorker, removeWorker)
- `internal/daemon/daemon.go` (health checks, cleanup, restoration)

---

## Summary

The worker lifecycle has several gaps that can result in:
- Orphaned resources (worktrees, tmux windows, branches)
- Lost work (uncommitted changes, unpushed commits)
- Missing task history records
- Inconsistent state between daemon and filesystem

---

## Gap 1: No Rollback on Worker Creation Failure

**Location:** `internal/cli/cli.go:1622-1828`

**Issue:** Worker creation is not atomic. Resources are created sequentially:
1. Worktree created (line 1700/1707)
2. Tmux window created (line 1746)
3. Claude started (line 1783)
4. Worker registered with daemon (line 1796)

If any step after worktree creation fails, earlier resources are orphaned.

**Example scenarios:**
- Daemon registration fails → orphaned worktree + tmux window
- Claude startup fails → orphaned worktree + tmux window
- Tmux window creation fails → orphaned worktree

**Severity:** Medium - orphans accumulate over time, waste disk space

**Suggested fix:** Add rollback logic or deferred cleanup on error path.

---

## Gap 2: Workers Not Cleaned Up After Crash

**Location:** `internal/daemon/daemon.go:300-310`

**Issue:** When a worker's Claude process dies (crash, system restart, etc.),
the worker is NOT auto-restarted or cleaned up. The comment says workers
"complete and clean up" but this only happens if the worker calls
`multiclaude agent complete` before dying.

```go
// For transient agents (workers, review), don't auto-restart - they complete and clean up
```

A crashed worker remains in state with:
- Dead PID
- Existing tmux window (empty shell)
- Existing worktree (possibly with uncommitted work)

The health check doesn't mark them as dead because the window still exists.

**Severity:** High - crashed workers become zombies, blocking cleanup

**Suggested fix:** Detect workers with dead PIDs and either:
- Mark them for cleanup after a grace period, or
- Notify supervisor that worker crashed with uncommitted work

---

## Gap 3: Branch Name Assumption in Cleanup

**Location:** `internal/daemon/daemon.go:1354-1360`

**Issue:** Cleanup assumes worker branch is `work/<agentName>`:
```go
branchName := "work/" + agentName
```

But workers created with `--push-to` flag use custom branch names
(cli.go:1694-1702). This causes:
- Failed branch deletion (branch doesn't exist)
- Wrong branch deleted (if another branch happens to match)

**Severity:** Low - deletion just fails silently with a warning

**Suggested fix:** Store actual branch name in agent state, use it for cleanup.

---

## Gap 4: Session Restoration Loses Worker State

**Location:** `internal/daemon/daemon.go:1614-1629`

**Issue:** When tmux session is lost (e.g., system restart, tmux server crash),
`restoreRepoAgents` removes ALL agents from state before recreating the session:

```go
for agentName := range repo.Agents {
    d.state.RemoveAgent(repoName, agentName)
}
```

Only persistent agents (supervisor, merge-queue, workspace) are recreated.
Workers are silently lost without:
- Task history record
- Notification to user
- Any recovery attempt

**Severity:** High - all in-progress worker state lost silently

**Suggested fix:**
- Record task history before removing workers
- Log warning about lost workers
- Consider storing worktree paths for recovery

---

## Gap 5: Manual Removal Skips Task History

**Location:** `internal/cli/cli.go:2225-2366`

**Issue:** `multiclaude worker rm` removes workers without recording task
history. Only `cleanupDeadAgents` calls `recordTaskHistory`.

Workers removed manually have no record of what they were working on.

**Severity:** Low - task history is for auditing, not critical

**Suggested fix:** Call daemon's complete_agent or add history recording to
removeWorker.

---

## Gap 6: Race Condition in Worker Completion

**Location:** `internal/daemon/daemon.go:984-1046`

**Issue:** `handleCompleteAgent` does three async things:
1. Marks agent as `ReadyForCleanup`
2. Sends messages to supervisor/merge-queue (async delivery via `go d.routeMessages()`)
3. Triggers cleanup check (async via `go d.checkAgentHealth()`)

The worker process might still be running after returning from `complete_agent`.
If health check runs fast, it could:
- Kill the tmux window while worker is still outputting
- Remove worktree while Claude is still writing files

**Severity:** Low - unlikely in practice due to timing

**Suggested fix:** Wait for worker process to exit before cleanup, or add
brief delay.

---

## Gap 7: PID Detection Timing Issues

**Location:** `internal/daemon/daemon.go:1768`

**Issue:** After starting Claude, code sleeps 500ms then gets PID:
```go
time.Sleep(500 * time.Millisecond)
pid, err := d.tmux.GetPanePID(...)
```

This is fragile:
- If Claude takes >500ms to start, might get shell PID instead
- If Claude starts faster, works but wastes time
- No verification that PID is actually Claude process

**Severity:** Low - usually works, but PID might be wrong

**Suggested fix:** Poll for Claude process or verify PID is claude binary.

---

## Gap 8: Orphaned State Without Worktree

**Location:** `internal/daemon/daemon.go:1471-1501`

**Issue:** `cleanupOrphanedWorktrees` only removes worktree directories not
tracked by git. It doesn't handle:
- Agent state without worktree (worktree manually deleted)
- Agent state with corrupted worktree (git refs broken)

The repair command (`handleTriggerCleanup`) does check for missing worktrees,
but this isn't run automatically.

**Severity:** Low - manual repair available

**Suggested fix:** Add worktree existence check to health check loop.

---

## Recommendations

### Immediate (P0)

1. **Add crash detection for workers** - If worker PID is dead and window
   exists (empty shell), mark as crashed and notify supervisor. Don't silently
   leave zombie workers.

2. **Record task history on session loss** - Before clearing agents during
   session restoration, record their task history so we know what was lost.

### Short-term (P1)

3. **Add rollback to worker creation** - Use deferred cleanup to remove
   worktree/window if later steps fail.

4. **Store actual branch name in state** - Don't assume `work/<name>` for
   branch cleanup.

### Nice-to-have (P2)

5. **Record task history on manual removal** - Make `worker rm` record history.

6. **Improve PID detection** - Poll or verify instead of fixed sleep.

---

## Testing Scenarios

To verify fixes, test these scenarios:

1. **Creation failure:** Kill daemon mid-worker-creation
2. **Worker crash:** `kill -9` the Claude process
3. **Tmux crash:** `tmux kill-server` with workers running
4. **Manual deletion:** `rm -rf` a worker worktree while it's in state
5. **Push-to cleanup:** Create worker with `--push-to`, complete it, verify
   correct branch cleanup
