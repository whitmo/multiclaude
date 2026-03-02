# Proposal: Incremental Context Compaction for Claude Code

**Author:** bright-squirrel (multiclaude worker)
**Date:** 2026-03-01
**Status:** Draft / Research Proposal

---

## Executive Summary

Claude Code currently performs context compaction as a **single foreground operation** when the context window reaches ~80-83.5% capacity. This creates a latency spike — a "compaction wall" — that interrupts the developer's flow at the worst possible time: deep into a complex task. This proposal argues for **incremental background compaction**, where older context is progressively compressed in small batches at natural boundaries (hook cycles, tool call completions), smoothing latency and preserving flow state.

---

## 1. How Compaction Works Today

### Trigger Mechanism

Compaction triggers when input tokens exceed approximately **80-83.5% of the 200K context window** (~160-167K tokens). At the API level, this threshold is configurable via the `trigger` parameter (minimum 50K tokens, default ~150K).

Two paths to compaction:
- **Automatic:** Fires when the threshold is crossed during normal conversation
- **Manual:** The `/compact` command lets users trigger it proactively, with optional focus instructions (e.g., `/compact focus on API changes`)

### The Compaction Process

When triggered, the system:

1. Sends the entire conversation to the model with a summarization prompt
2. The model generates a `<summary>` block capturing key state, decisions, code changes, and next steps
3. All message blocks **prior to the compaction block are dropped**
4. The conversation continues from the summary forward
5. Achieves ~85% token compression (167K → ~25K tokens)

### The Default Summarization Prompt

```
You have written a partial transcript for the initial task above.
Please write a summary of the transcript. The purpose of this summary
is to provide continuity so you can continue to make progress towards
solving the task in a future context, where the raw history above may
not be accessible and will be replaced with this summary. Write down
anything that would be helpful, including the state, next steps,
learnings etc.
```

### Why It's Foreground/Blocking

The compaction is effectively **foreground** because:

1. **Atomic consistency:** The summary must capture the full conversation state at a single point in time. Doing it in one pass avoids race conditions where new messages arrive mid-summarization.
2. **Simplicity:** One-shot summarization is architecturally simple — no bookkeeping about what has/hasn't been summarized.
3. **API design:** The Messages API processes compaction as part of the response generation cycle. The model generates the summary, then the API drops old messages and continues.
4. **Correctness guarantee:** A single comprehensive pass over the full context ensures nothing is missed. Incremental approaches risk "summarizing the summary" and losing fidelity.

These are reasonable engineering tradeoffs for a v1, but they create a **predictable pain point** at scale.

---

## 2. The Problem: The Compaction Wall

### User Experience

When compaction fires automatically:
- The conversation **pauses** while the model processes the entire context into a summary
- For a near-full 200K window, this can take **10-30 seconds** of model processing time
- The developer is mid-thought, waiting, and the compacted context may lose nuances they were relying on
- There's no warning — it fires when you've been working longest and are most deeply engaged

### For Multiclaude / Agentic Workloads

The problem is amplified for autonomous agents:
- Agents hit compaction walls during long autonomous tasks
- A supervisor waiting on a worker gets delayed by the worker's compaction
- Multiple agents compacting simultaneously creates resource spikes
- Post-compaction, agents may lose important inter-agent context (message history, coordination state)

### The Fundamental Issue

Compaction is a **batch operation in a streaming world.** It's the garbage-collection pause of LLM context management — theoretically necessary, practically disruptive, and potentially avoidable with better architecture.

---

## 3. Proposal: Incremental Background Compaction

### Core Idea

Instead of waiting for the context to fill up and then performing a large compaction, perform **small, continuous compressions** of older context at natural boundaries. Think of it as a **generational garbage collector** for conversation context.

### Architecture: Rolling Summary with Generations

Divide the context into logical "generations" based on age:

```
┌─────────────────────────────────────────────────────────┐
│ CONTEXT WINDOW (200K tokens)                            │
│                                                         │
│ ┌──────────┐  ┌──────────────┐  ┌────────────────────┐  │
│ │ Gen 2    │  │ Gen 1        │  │ Gen 0 (Live)       │  │
│ │ (Deep    │  │ (Lightly     │  │ (Raw messages,     │  │
│ │  summary │  │  compressed) │  │  full fidelity)    │  │
│ │  ~5K)    │  │  ~15K)       │  │  ~80K)             │  │
│ └──────────┘  └──────────────┘  └────────────────────┘  │
│                                                         │
│ ◄─── older ──────────────────────────── newer ────────► │
└─────────────────────────────────────────────────────────┘
```

**Gen 0 (Live):** Raw, uncompressed messages. Full fidelity. This is the recent working context.

**Gen 1 (Lightly Compressed):** Messages from 5-20 turns ago. Verbose tool outputs trimmed, exploratory discussions summarized, but code changes and decisions preserved verbatim.

**Gen 2 (Deep Summary):** Everything older than ~20 turns. A running summary that captures architectural decisions, file modifications, test results, and key learnings. Updated incrementally.

### When Compression Happens

**At natural boundaries**, not at arbitrary thresholds:

1. **After each tool call completes:** Before returning the result to the model, check if any Gen 0 content has aged past the generation boundary. If so, compress the oldest Gen 0 block into Gen 1 format. This adds ~1-2 seconds but is amortized across many operations.

2. **At hook cycle boundaries:** If Claude Code hooks fire (e.g., post-tool hooks), use that processing time to compress one block from Gen 1 into Gen 2.

3. **During model "thinking" time:** While the model is processing its next response, a background thread could compress older generations.

4. **On idle:** If the user hasn't sent input for N seconds, proactively compress.

### Compression Operations by Generation

| Transition | What Happens | Cost |
|-----------|-------------|------|
| Gen 0 → Gen 1 | Trim verbose tool outputs (file listings, search results). Replace with summaries. Keep code diffs and decisions verbatim. | ~1-2s per block |
| Gen 1 → Gen 2 | Merge into rolling summary. Extract key facts (files changed, decisions made, errors encountered). Discard procedural detail. | ~2-3s per block |
| Gen 2 maintenance | Periodically re-summarize Gen 2 if it grows beyond budget (e.g., 10K tokens). | ~3-5s, infrequent |

### Implementation Sketch

```
// Pseudocode for incremental compaction

on_tool_call_complete(result):
    context.append(result)

    // Check if oldest Gen 0 blocks should graduate
    if context.gen0_age_exceeded(threshold=10_turns):
        oldest_block = context.gen0.oldest()
        compressed = lightweight_compress(oldest_block)  // trim outputs, keep decisions
        context.gen1.append(compressed)
        context.gen0.remove(oldest_block)

    // Check if Gen 1 is over budget
    if context.gen1.token_count > GEN1_BUDGET:
        oldest_gen1 = context.gen1.oldest()
        context.gen2.rolling_summary = merge_into_summary(
            context.gen2.rolling_summary,
            oldest_gen1
        )
        context.gen1.remove(oldest_gen1)

on_idle(seconds_idle):
    if seconds_idle > 5 and context.needs_compression():
        compress_one_block()  // background, non-blocking
```

---

## 4. Tradeoffs Analysis

### Latency Smoothing vs. Total Compute

| Dimension | Current (Batch) | Proposed (Incremental) |
|-----------|----------------|----------------------|
| **Peak latency** | 10-30s compaction wall | 1-3s micro-compressions |
| **Total compute** | Lower (one pass) | Higher (~20-40% more total summarization tokens) |
| **User experience** | Jarring but infrequent | Smooth but constant overhead |
| **Predictability** | Unpredictable timing | Predictable, bounded latency |

**The core tradeoff:** Incremental compaction trades **total compute** (more model calls for summarization) for **latency smoothing** (no single large pause). This is the same tradeoff that generational GC makes versus stop-the-world GC, and for interactive workloads, it's almost always worth it.

For pay-per-token pricing, the additional summarization tokens add cost. Rough estimate: ~20-40% more summarization tokens over a long session, but each individual cost is small and amortized.

### Context Consistency

**Risk:** Incremental summaries may be less coherent than a single-pass summary over the full context. The model sees the full picture in batch mode; in incremental mode, it only sees the local block being compressed plus the existing summary.

**Mitigation:**
- Gen 1 compression is mostly mechanical (trimming verbose outputs), not semantic summarization. Low risk of information loss.
- Gen 2 merging uses the existing summary as context when incorporating new information, maintaining narrative coherence.
- Periodic "re-summarization" of Gen 2 provides a consistency checkpoint.
- Custom compaction instructions (from CLAUDE.md) are applied at every compression step, not just once.

### Quality of Compressed Context

**Risk:** "Summarizing summaries" loses fidelity over time — the telephone game problem.

**Mitigation:**
- **Structured extraction over narrative summary:** Instead of asking the model to "summarize," extract structured data: `{files_modified: [...], decisions: [...], errors: [...], current_state: "..."}`. Structured data survives re-compression better than prose.
- **Anchoring to CLAUDE.md:** Persistent context (project rules, patterns, architecture) lives in CLAUDE.md and is re-injected fresh every cycle. Only ephemeral session state needs compression.
- **Compression is lossy by design:** Even batch compaction loses information. The question isn't "is incremental lossier?" but "is the additional loss worth the UX improvement?" For most development tasks, yes.

---

## 5. The Rolling Summary Approach

### How It Would Work

The Gen 2 rolling summary would be a **structured, append-merge document** rather than a narrative summary:

```markdown
## Rolling Context Summary (auto-maintained)

### Modified Files
- `internal/cli/cli.go` - Added /compact command handler (line ~2400)
- `internal/daemon/daemon.go` - Added health check for compaction state
- `pkg/config/paths.go` - New CompactionDir path

### Key Decisions
- Using generational compression (3 levels)
- Structured extraction preferred over narrative summary
- Hook-cycle-aligned compression timing

### Current State
- Implementing Gen 1 → Gen 2 compression
- Tests passing for Gen 0 → Gen 1
- Blocked on: deciding Gen 2 token budget

### Errors & Learnings
- Initial approach of compressing every turn was too aggressive
- Tool outputs > 5K tokens should be trimmed immediately (Gen 0.5)

### Recent Activity (last 5 turns)
- Read internal/daemon/daemon.go
- Edited lines 450-470 to add compression trigger
- Ran tests: 3 passed, 0 failed
```

### Advantages of Structured Rolling Summary

1. **Survives re-compression:** Structured data can be mechanically merged, not narratively re-summarized. The telephone game doesn't apply to append-only structured data.
2. **Queryable:** The model can quickly find what it needs (file list, decisions, errors) without scanning prose.
3. **Bounded growth:** Each section has a natural cap. "Modified Files" won't grow beyond the number of files in the project. "Key Decisions" can be capped at N most recent.
4. **Diffable:** Changes to the summary between compression cycles are easy to audit and debug.

### Challenges

1. **Cross-referencing:** Sometimes context from turn 5 is needed to understand turn 50. Incremental compression may have already discarded that connection. Mitigation: the "Key Decisions" section acts as a persistent cross-reference.
2. **Emergent patterns:** A batch summary can identify patterns across the full conversation ("you keep hitting the same bug in X"). Incremental compression may miss these. Mitigation: periodic full re-summarization of Gen 2 (every ~30 turns).
3. **Implementation complexity:** Significantly more complex than batch compaction. More state to track, more edge cases, more testing required.

---

## 6. Applicability to Multiclaude

For multi-agent orchestration, incremental compaction has additional benefits:

| Scenario | Batch Impact | Incremental Benefit |
|----------|-------------|-------------------|
| Worker on long task | 15s pause mid-implementation | Smooth operation, no coordination delays |
| Supervisor monitoring 5 workers | Random pauses when checking status | Predictable response times |
| Agent-to-agent messaging | Messages arrive during compaction, potentially lost from summary | Messages are compressed gradually with full context |
| Parallel agents | Multiple agents compact simultaneously, resource spike | Compaction load distributed over time |

### Specific Multiclaude Enhancement

Agents could share their Gen 2 summaries as a form of **distributed context**. If worker-A has summarized its changes to `internal/cli/cli.go`, worker-B could ingest that summary instead of re-reading the file, enabling lighter-weight coordination.

---

## 7. Implementation Roadmap (If Pursued)

### Phase 1: Immediate Tool Output Trimming
- Trim tool outputs (file reads, search results, git diffs) older than N turns to their first/last 20 lines + a size annotation
- Zero model cost — purely mechanical
- Estimated impact: 30-50% context savings on tool-heavy sessions

### Phase 2: Gen 0 → Gen 1 Compression
- After each tool call, check if oldest messages exceed age threshold
- Compress using a fast model (Haiku) for low latency
- Preserve code changes verbatim, summarize discussions

### Phase 3: Gen 1 → Gen 2 Rolling Summary
- Implement structured rolling summary format
- Merge compressed blocks into summary at hook boundaries
- Add periodic consistency re-summarization

### Phase 4: Cross-Agent Summary Sharing (Multiclaude-specific)
- Agents publish their Gen 2 summaries
- Other agents can ingest summaries for lightweight coordination
- Supervisor gets automatic context about worker state

---

## 8. Open Questions

1. **What's the right generation boundary?** 10 turns? 20? Should it be token-based or turn-based? Probably token-based with a turn minimum (don't compress anything < 5 turns old).

2. **Which model for compression?** Using the same Opus/Sonnet model for summarization is expensive. Haiku is faster and cheaper but may miss nuance. Could a fine-tuned summarization model work?

3. **How to handle user-specified compaction instructions?** Currently `/compact focus on X` applies to the whole context. In incremental mode, should the focus apply to every micro-compression, or only to Gen 2 summarization?

4. **Interaction with prompt caching:** Incremental compression changes the message structure more frequently, potentially invalidating cached prefixes. Need to ensure compression boundaries align with cache breakpoints.

5. **Can we even do this client-side?** If compaction is an API-level feature (the API drops old messages), the client (Claude Code) may not have direct control over when/how it happens. This proposal may require API changes.

6. **Backwards compatibility:** Existing conversations expect batch compaction. How do you migrate? Probably: incremental for new conversations, batch as fallback.

---

## 9. Conclusion

Current batch compaction is a **correct but unergonomic** solution. It solves the hard problem (context overflow) but creates a secondary problem (latency spikes at the worst time). Incremental compaction trades modest additional compute for dramatically smoother developer experience.

The generational approach is proven in garbage collection, database compaction (LSM trees), and log-structured storage. Context management is fundamentally the same problem: bounded memory, unbounded input, lossy compression required. The same solutions apply.

**Recommendation:** Start with Phase 1 (mechanical tool output trimming) — it's zero-risk, zero-cost, and immediately impactful. Use it to validate the incremental architecture before investing in model-driven compression phases.

---

## References

- [Claude API Compaction Documentation](https://platform.claude.com/docs/en/build-with-claude/compaction)
- [Claude Code Context Management](https://code.claude.com/docs/en/how-claude-code-works)
- [Claude Code Best Practices](https://code.claude.com/docs/en/best-practices)
- [Automatic Context Compaction Cookbook](https://platform.claude.com/cookbook/tool-use-automatic-context-compaction)
- [Context Compaction Research Comparison](https://gist.github.com/badlogic/cd2ef65b0697c4dbe2d13fbecb0a0a5f)
- Generational Garbage Collection — Ungar, 1984
- LSM Trees — O'Neil et al., 1996
