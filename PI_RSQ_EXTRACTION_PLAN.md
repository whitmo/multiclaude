# pi-rsq Extraction Plan

Move pi-rsq from `github.com/dlorenc/multiclaude` into a standalone repository at `github.com/whitmo/pi-rsq`.

## Current state

pi-rsq lives inside the multiclaude monorepo:

| Location | What | Lines |
|---|---|---|
| `cmd/pi-rsq/main.go` | Entry point | ~20 |
| `internal/pirsq/` | All packages (cli, config, daemon, pirpc, proc, prompts, publisher, repos, runtime, service, state, worktree) | ~16k |
| `docs/pi-rsq/` | README, ARCHITECTURE, ROADMAP, STATE_SCHEMA, QUEUE, NEXT_STEPS, PARITY_TOMLIN_PROPOSAL, RALPH_* docs, skills/ | ~13 files |
| `PROMPT.pi.impl.md` | Ralph implementation prompt | ~30 |
| `ralph.pi.impl.yml` | Ralph config for pi-rsq work | ~10 |
| `scripts/ralph-pi-night-*.sh` | Ralph nightly loop scripts | ~3 files |
| `.gitignore` line 41 | `/pi-rsq` (compiled binary) | 1 line |
| `.ralph/night-runs/` | Runtime artifacts (not tracked) | N/A |

Git history: ~63 commits touch pi-rsq files across `internal/pirsq/`, `cmd/pi-rsq/`, and `docs/pi-rsq/`.

### Shared dependencies (pi-rsq imports from multiclaude)

pi-rsq imports two non-pirsq multiclaude packages:

1. **`internal/socket`** (~170 lines) — Unix socket client/server with JSON request/response. Used by `pirsq/cli`, `pirsq/daemon/server`, `pirsq/runtime/supervisor`, `pirsq/service`.

2. **`internal/worktree`** (~900 lines) — Git worktree manager. Used by `pirsq/repos` (for `DetectDefaultBranch`) and `pirsq/worktree` (wraps `basewt.Manager`).

No multiclaude code imports pirsq (one-way dependency, clean cut).

### External Go dependencies

From `go.mod`: `fatih/color`, `google/uuid`, plus their transitive deps. pi-rsq may not use all of these; the new repo's `go.mod` should only declare what pi-rsq actually imports.

---

## What moves to whitmo/pi-rsq

### Must move

| Source path | Destination path | Notes |
|---|---|---|
| `cmd/pi-rsq/main.go` | `cmd/pi-rsq/main.go` | Rewrite import path |
| `internal/pirsq/*` | `internal/*` | Drop the `pirsq/` nesting; e.g. `internal/pirsq/cli` -> `internal/cli` |
| `docs/pi-rsq/*` | `docs/*` | Promote to top-level docs |
| `docs/pi-rsq/skills/` | `docs/skills/` or `skills/` | Skill templates |
| `internal/socket/` | `internal/socket/` | Copy (or vendor) — shared package |
| `internal/worktree/worktree.go` | `internal/worktree/worktree.go` | Copy — shared package |

### Should move

| Source path | Destination | Notes |
|---|---|---|
| `PROMPT.pi.impl.md` | `PROMPT.pi.impl.md` | Ralph prompt — pi-rsq specific |
| `ralph.pi.impl.yml` | `ralph.pi.impl.yml` | Ralph config — pi-rsq specific |
| `scripts/ralph-pi-night-loop.sh` | `scripts/ralph-pi-night-loop.sh` | Ralph nightly driver |
| `scripts/ralph-pi-night-status.sh` | `scripts/ralph-pi-night-status.sh` | Ralph nightly status |
| `scripts/ralph-pi-night-repeat.sh` | `scripts/ralph-pi-night-repeat.sh` | Ralph repeat loop |

### Stays in multiclaude

| Item | Reason |
|---|---|
| Everything under `internal/cli/`, `internal/daemon/`, `internal/state/`, `pkg/`, etc. | Core multiclaude |
| `internal/socket/`, `internal/worktree/` | Still used by multiclaude proper; pi-rsq gets a copy |
| `Makefile`, `.github/workflows/`, `go.mod`, `CLAUDE.md`, `ROADMAP.md` | multiclaude infra |
| Other `scripts/` (non-pi-rsq Ralph scripts, `pre-commit.sh`, etc.) | multiclaude tooling |
| `.ralph/` runtime artifacts | Ephemeral; not tracked |

### Cleanup in multiclaude after extraction

- Remove `cmd/pi-rsq/`, `internal/pirsq/`, `docs/pi-rsq/`
- Remove `PROMPT.pi.impl.md`, `ralph.pi.impl.yml`
- Remove `scripts/ralph-pi-night-*.sh`
- Remove `/pi-rsq` line from `.gitignore`
- Remove stale worktree artifacts from `.claude/worktrees/` that reference pi-rsq

---

## Module and import path changes

### New Go module

```
module github.com/whitmo/pi-rsq
```

### Import path rewrites

All import paths change from `github.com/dlorenc/multiclaude/internal/pirsq/<pkg>` to `github.com/whitmo/pi-rsq/internal/<pkg>`.

The shared-package imports change from `github.com/dlorenc/multiclaude/internal/socket` and `github.com/dlorenc/multiclaude/internal/worktree` to `github.com/whitmo/pi-rsq/internal/socket` and `github.com/whitmo/pi-rsq/internal/worktree`.

Summary of rewrites:

| Old import | New import |
|---|---|
| `github.com/dlorenc/multiclaude/internal/pirsq/cli` | `github.com/whitmo/pi-rsq/internal/cli` |
| `github.com/dlorenc/multiclaude/internal/pirsq/config` | `github.com/whitmo/pi-rsq/internal/config` |
| `github.com/dlorenc/multiclaude/internal/pirsq/daemon` | `github.com/whitmo/pi-rsq/internal/daemon` |
| `github.com/dlorenc/multiclaude/internal/pirsq/pirpc` | `github.com/whitmo/pi-rsq/internal/pirpc` |
| `github.com/dlorenc/multiclaude/internal/pirsq/proc` | `github.com/whitmo/pi-rsq/internal/proc` |
| `github.com/dlorenc/multiclaude/internal/pirsq/prompts` | `github.com/whitmo/pi-rsq/internal/prompts` |
| `github.com/dlorenc/multiclaude/internal/pirsq/publisher` | `github.com/whitmo/pi-rsq/internal/publisher` |
| `github.com/dlorenc/multiclaude/internal/pirsq/repos` | `github.com/whitmo/pi-rsq/internal/repos` |
| `github.com/dlorenc/multiclaude/internal/pirsq/runtime` | `github.com/whitmo/pi-rsq/internal/runtime` |
| `github.com/dlorenc/multiclaude/internal/pirsq/service` | `github.com/whitmo/pi-rsq/internal/service` |
| `github.com/dlorenc/multiclaude/internal/pirsq/state` | `github.com/whitmo/pi-rsq/internal/state` |
| `github.com/dlorenc/multiclaude/internal/pirsq/worktree` | `github.com/whitmo/pi-rsq/internal/worktree` |
| `github.com/dlorenc/multiclaude/internal/socket` | `github.com/whitmo/pi-rsq/internal/socket` |
| `github.com/dlorenc/multiclaude/internal/worktree` | `github.com/whitmo/pi-rsq/internal/worktree` |

The `gsocket` alias used in several files (e.g. `gsocket "github.com/dlorenc/multiclaude/internal/socket"`) and the `basewt` alias (e.g. `basewt "github.com/dlorenc/multiclaude/internal/worktree"`) can be dropped since there will no longer be naming conflicts.

---

## History preservation options

### Primary: full-history fork/relocate

Fork (or push) the entire multiclaude repository to `whitmo/pi-rsq`, then delete everything that isn't pi-rsq.

**Steps:**

1. `git clone github.com/dlorenc/multiclaude pi-rsq`
2. `cd pi-rsq`
3. Remove all non-pi-rsq files and directories
4. Restructure (flatten `internal/pirsq/` -> `internal/`, promote `docs/pi-rsq/` -> `docs/`, etc.)
5. Rewrite `go.mod`, fix all import paths
6. Commit the restructure as one clear "Extract pi-rsq as standalone repo" commit
7. Push to `github.com/whitmo/pi-rsq`

**Pros:**
- Full git history preserved — `git log --follow` works for all files
- Simplest to execute
- No tooling risk
- All commit hashes and author attribution preserved

**Cons:**
- Repo contains multiclaude history for files that are now deleted (extra weight)
- `git log` without `--follow` shows unrelated multiclaude commits
- The repo will be larger than necessary (but not dramatically — multiclaude is not huge)

**This is the recommended approach** per operator guidance.

### Secondary alternative: `git filter-repo` path extraction

Use `git filter-repo` to rewrite history so only pi-rsq paths are preserved.

```bash
git filter-repo \
  --path internal/pirsq/ \
  --path cmd/pi-rsq/ \
  --path docs/pi-rsq/ \
  --path internal/socket/ \
  --path internal/worktree/ \
  --path PROMPT.pi.impl.md \
  --path ralph.pi.impl.yml \
  --path scripts/ralph-pi-night-loop.sh \
  --path scripts/ralph-pi-night-status.sh \
  --path scripts/ralph-pi-night-repeat.sh
```

Then do the same rename/restructure commit on top.

**Pros:**
- Cleaner history — only pi-rsq-related commits
- Smaller repo

**Cons:**
- Rewrites all commit hashes
- `git filter-repo` can be finicky with path renames
- Shared packages (`socket`, `worktree`) bring in their own commit history which may include unrelated multiclaude changes
- More error-prone, harder to validate

### Tertiary alternative: `git subtree split`

Not recommended. pi-rsq spans multiple top-level directories (`cmd/`, `internal/`, `docs/`), so `git subtree split` cannot produce a single coherent subtree.

---

## Bootstrap steps for whitmo/pi-rsq

### 1. Create the repository

```bash
gh repo create whitmo/pi-rsq --public --description "Repo-centric multi-agent orchestrator built around pi RPC"
```

### 2. Clone multiclaude and prepare the extraction

```bash
git clone git@github.com:dlorenc/multiclaude.git pi-rsq-staging
cd pi-rsq-staging
```

### 3. Remove non-pi-rsq content

```bash
# Keep these, remove everything else
# cmd/pi-rsq/
# internal/pirsq/
# internal/socket/
# internal/worktree/worktree.go (and worktree_test.go, refresh_test.go)
# docs/pi-rsq/
# PROMPT.pi.impl.md
# ralph.pi.impl.yml
# scripts/ralph-pi-night-*.sh

# Remove multiclaude-specific content
rm -rf cmd/multiclaude cmd/generate-docs cmd/verify-docs
rm -rf internal/cli internal/daemon internal/state internal/messages
rm -rf internal/prompts internal/hooks internal/names internal/errors
rm -rf internal/bugreport internal/templates internal/agents internal/fork
rm -rf internal/redact
rm -rf pkg/
rm -rf test/
rm -rf docs/AGENTS.md docs/extending/ docs/CLI_RESTRUCTURE_PROPOSAL.md
rm -f Makefile ROADMAP.md CLAUDE.md ARCHITECTURE.md
rm -f .github/workflows/*  # will replace with pi-rsq CI
rm -f ralph.yml ralph.pi.review.yml PROMPT.md PROMPT.pi.review.md
rm -f scripts/pre-commit.sh scripts/ralph-cycle*.sh scripts/review-ralph-worktree.sh
rm -f scripts/watch-ralph-run.sh scripts/ralph-task-rollup.py scripts/ralph_task_rollup_test.py
```

### 4. Restructure directories

```bash
# Flatten internal/pirsq/ -> internal/
mv internal/pirsq/* internal/
rmdir internal/pirsq

# Promote docs
mv docs/pi-rsq/* docs/
rmdir docs/pi-rsq
```

### 5. Create new go.mod

```go
module github.com/whitmo/pi-rsq

go 1.25.1
```

Then `go mod tidy` to discover actual dependencies.

### 6. Rewrite import paths

Global find-and-replace across all `.go` files:

```bash
# Flatten pirsq imports
find . -name '*.go' -exec sed -i '' \
  's|github.com/dlorenc/multiclaude/internal/pirsq/|github.com/whitmo/pi-rsq/internal/|g' {} +

# Shared package imports
find . -name '*.go' -exec sed -i '' \
  's|github.com/dlorenc/multiclaude/internal/socket|github.com/whitmo/pi-rsq/internal/socket|g' {} +

find . -name '*.go' -exec sed -i '' \
  's|github.com/dlorenc/multiclaude/internal/worktree|github.com/whitmo/pi-rsq/internal/worktree|g' {} +
```

### 7. Remove import aliases that are no longer needed

The `gsocket` alias (for `internal/socket`) and `basewt` alias (for `internal/worktree`) exist because the multiclaude repo had naming conflicts with pirsq's own packages. After flattening:

- `pirsq/worktree` becomes `internal/worktree` and wraps the same-package `basewt` — this needs a refactor: either inline the base worktree into pirsq's worktree package, or rename one of them (e.g. `internal/gitworktree` for the base).
- `gsocket` can likely be dropped since there won't be a conflicting `socket` package.

### 8. Update .gitignore

Replace multiclaude-specific entries with pi-rsq ones:

```
/pi-rsq
.ralph/
*.log
```

### 9. Create new CLAUDE.md

Write a project-specific CLAUDE.md for pi-rsq covering:
- Project overview (from existing `docs/pi-rsq/README.md`)
- Build/test commands
- Architecture summary
- Package map
- Contributing checklist

### 10. Verify build and tests

```bash
go build ./cmd/pi-rsq
go test ./internal/...
go test ./cmd/...
```

---

## CI / test / release setup

### CI (GitHub Actions)

Create `.github/workflows/ci.yml`:

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go build ./cmd/pi-rsq
      - run: go test ./...
      - run: go vet ./...
```

### Release

Create `.github/workflows/release.yml` using `goreleaser` or simple `go build` matrix:

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go build -o pi-rsq ./cmd/pi-rsq
      # upload artifact or use goreleaser
```

### Test mode

pi-rsq already supports `PIRSQ_TEST_MODE=1` for deterministic testing without a running daemon. This should carry over unchanged.

---

## Migration risks

### 1. Worktree package naming collision

After flattening, both the base worktree package (`internal/worktree`) and pirsq's worktree wrapper (`internal/pirsq/worktree`) would both want to be `internal/worktree`. 

**Resolution options:**
- (a) Merge them into one package — the pirsq wrapper is only ~50 lines
- (b) Rename the base to `internal/gitworktree` and keep pirsq's as `internal/worktree`
- (c) Keep the wrapper and inline the base functions it uses

**Recommended:** Option (a) — merge. The wrapper is thin enough to fold into the base.

### 2. Socket package may diverge

`internal/socket` is currently shared between multiclaude and pi-rsq. After extraction, the two copies will drift independently. This is acceptable — the socket package is small (~170 lines), stable, and generic.

### 3. Runtime directory collision

Both multiclaude (`~/.multiclaude/`) and pi-rsq (`~/.pi-rsq/`) already use separate runtime directories. No collision risk.

### 4. Ralph script paths

Ralph scripts reference `$ROOT` relative paths. After moving to the new repo, these should work unchanged since they use `$(cd "$(dirname "$0")/.." && pwd)`.

### 5. Skill template installation paths

The worker and swarm skill templates install to `~/.agents/skills/`. The templates reference `pi-rsq` commands, so they should continue to work. The `docs/pi-rsq/skills/` path changes to `docs/skills/` — update any installation instructions.

### 6. `go.sum` may need regeneration

After rewriting `go.mod`, run `go mod tidy` to regenerate `go.sum`. Some transitive dependencies from multiclaude's `go.mod` may no longer be needed.

### 7. Stale `docs/pi-rsq/` references within docs

Several docs cross-reference each other using relative paths like `[ARCHITECTURE.md](./ARCHITECTURE.md)`. After promotion from `docs/pi-rsq/` to `docs/`, these should still work. But references from README that say "See docs/pi-rsq/" need updating to "See docs/".

---

## Phased PR plan

### Phase 0: Preparation (in multiclaude)

**PR: "Prepare pi-rsq for extraction"**

- Ensure all pi-rsq tests pass: `go test ./internal/pirsq/... ./cmd/pi-rsq/...`
- Verify no multiclaude code imports pirsq (confirmed: none do)
- Fix any doc cross-references that would break after the move
- Tag a commit or note the SHA as the extraction point

### Phase 1: Create whitmo/pi-rsq (new repo)

**Not a PR — repo creation and initial push.**

1. Create `github.com/whitmo/pi-rsq`
2. Clone multiclaude at the tagged extraction point
3. Remove non-pi-rsq content
4. Restructure directories (flatten `internal/pirsq/` -> `internal/`, promote docs)
5. Resolve worktree package naming (merge base into wrapper)
6. Rewrite `go.mod` and all import paths
7. Create new `.gitignore`, `CLAUDE.md`, `README.md` (from docs/pi-rsq/README.md)
8. Verify: `go build ./cmd/pi-rsq && go test ./...`
9. Push to `whitmo/pi-rsq` as the initial state

### Phase 2: CI and project scaffolding (in whitmo/pi-rsq)

**PR: "Add CI, release workflow, and project scaffolding"**

- Add `.github/workflows/ci.yml`
- Add `.github/workflows/release.yml` (if desired)
- Add LICENSE file
- Update `docs/README.md` to reflect standalone repo status
- Update skill template installation docs
- Verify CI passes

### Phase 3: Cleanup multiclaude (in dlorenc/multiclaude)

**PR: "Remove extracted pi-rsq code"**

- Remove `cmd/pi-rsq/`, `internal/pirsq/`, `docs/pi-rsq/`
- Remove `PROMPT.pi.impl.md`, `ralph.pi.impl.yml`
- Remove `scripts/ralph-pi-night-*.sh`
- Remove `/pi-rsq` from `.gitignore`
- Update multiclaude's CLAUDE.md if it references pi-rsq
- Run `go test ./...` to confirm nothing broke

### Phase 4: Post-extraction polish (in whitmo/pi-rsq)

**PR(s) as needed:**

- Clean up any remaining `dlorenc/multiclaude` references in docs or comments
- Drop `gsocket`/`basewt` import aliases where no longer needed
- Add a top-level `Makefile` with `build`, `test`, `install` targets
- Consider whether `internal/socket` should become `pkg/socket` for reusability
- Update `NEXT_STEPS.md` and `QUEUE.md` to reflect standalone context

---

## Decision log

| Decision | Choice | Rationale |
|---|---|---|
| History strategy | Full-history fork, delete non-pi-rsq | Simplest, preserves all attribution, operator preference |
| Worktree collision | Merge base into pirsq wrapper | Wrapper is ~50 lines, avoids rename churn |
| Socket package | Copy, not shared module | Too small to justify a shared Go module |
| Module path | `github.com/whitmo/pi-rsq` | Matches target repo |
| `internal/pirsq/` flattening | Yes, drop `pirsq/` nesting | No longer needed as a namespace when it's the whole repo |
| Ralph scripts | Move with the repo | pi-rsq specific, not multiclaude general |
