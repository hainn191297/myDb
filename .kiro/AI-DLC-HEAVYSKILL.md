# AI-DLC Heavy Thinking Protocol

## Purpose

This repository uses `.kiro` as an AI-assisted development lifecycle (AI-DLC) system. For complex or high-risk work, agents should use a heavy-thinking protocol before implementation: explore several independent solution paths, synthesize them critically, then execute the selected plan.

The goal is not more documentation for its own sake. The goal is to reduce wrong turns before code is changed, tests are run, or architectural direction hardens.

## When to Activate

Use this protocol for:

- New feature design or implementation across multiple packages
- Query planner, executor, storage, WAL, transaction, lock, or catalog changes
- Bugs where the root cause is unclear
- Refactors that affect public APIs, data formats, concurrency, persistence, or query semantics
- Tasks where correctness can be tested but the path is not obvious
- Any change where a bad first implementation would create misleading tests or architecture drift

Do not use the full protocol for:

- Small typo fixes
- Mechanical formatting
- Simple one-file changes with obvious behavior
- Pure information lookup
- Running an already-known verification command

## Resolve-Before-Run Gate

Before running implementation commands or editing code for an activated task, resolve the problem on paper:

1. Define the actual problem in one or two sentences.
2. Identify affected layers from `RDBMS-ARCHITECTURE.md`.
3. Read the relevant `.kiro/specs/{feature}/` documents.
4. State the invariants that must remain true.
5. Generate at least two plausible implementation approaches.
6. Compare approaches by correctness, simplicity, testability, and fit with existing code.
7. Select one approach and name the validation command or test cases.

Only then move to code edits or command execution.

## Parallel Reasoning Stage

For complex tasks, produce independent reasoning tracks. In a harness with subagents, use separate agents. Without subagents, simulate separate tracks explicitly and keep them independent until synthesis.

Recommended tracks for this repository:

- **Architecture track**: What layer owns this behavior? What interfaces should change?
- **Correctness track**: What invariants, edge cases, and failure modes matter?
- **Implementation track**: What is the smallest code path that fits existing Go patterns?
- **Testing track**: What tests prove the behavior and prevent regressions?

Each track should return:

- Proposed approach
- Files likely affected
- Main risk
- Validation needed

## Sequential Deliberation Stage

After collecting tracks, synthesize rather than vote.

Ask:

- Which track has the strongest evidence from existing code and specs?
- Are any assumptions contradicted by current implementation?
- Is the majority approach actually simpler, or just more familiar?
- Can the selected plan be tested locally and traced from SQL entrypoint to storage or execution behavior?
- Does the plan preserve the learning-oriented style of this repository?

If all tracks are weak, restart from the repo/spec context before editing.

## AI-DLC Feature Workflow

For each substantial feature, keep this lifecycle:

1. **Intent**: Update or create requirements with user-visible behavior.
2. **Design**: Define component ownership, data flow, interfaces, and invariants.
3. **Tasks**: Break implementation into dependency-ordered, testable units.
4. **Resolve**: Apply heavy thinking to the next task before code changes.
5. **Implement**: Make the smallest coherent change.
6. **Verify**: Run focused tests first, then broader tests when risk warrants it.
7. **Record**: Update task status or implementation notes when behavior, scope, or architecture changes.

## Output Expectations

When reporting work, keep the heavy-thinking details internal unless the user asks for them. The user-facing result should include:

- The selected approach
- Files changed or planned
- Tests run or planned
- Any unresolved risk

For code changes, do not expose raw parallel reasoning transcripts. Summarize the decision and evidence.

## Model-Agnostic Rule

This protocol is for all models and agents working in this repository. It does not depend on a specific provider, hidden chain-of-thought, or a particular agent framework. A weaker model should use fewer tracks with stricter checklists; a stronger model may use more tracks or real subagents. In all cases, the required behavior is the same: resolve the problem before running the implementation.
