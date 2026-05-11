# AI-DLC Workflow

## Default Loop

Use this loop for non-trivial work:

1. **Understand**: Read `AGENTS.md`, `.agent/project.md`, and relevant code.
2. **Resolve**: Define the problem, affected layers, invariants, candidate approaches, selected approach, and validation path.
3. **Implement**: Make the smallest coherent change that fits existing code.
4. **Verify**: Run focused tests first, then broader tests when risk warrants it.
5. **Record**: Update docs, specs, tasks, or implementation notes when behavior or architecture changes.

## Resolve-Before-Run Rule

For complex or high-risk tasks, do not start implementation by running broad commands or editing files. First resolve:

- What is the actual problem?
- Which packages and architecture layers are affected?
- What must remain invariant?
- What are at least two plausible approaches?
- Why is the selected approach the best fit?
- What command or test proves the change?

If the task involves `.kiro` specs, apply `.kiro/AI-DLC-HEAVYSKILL.md`.

## Task Sizing

- Small: one package, obvious behavior, focused test.
- Medium: multiple files or APIs, requires a written plan and focused tests.
- Large: cross-layer behavior, persistence, concurrency, planner/executor/storage semantics, or public API changes. Use heavy thinking and update specs.

## Reporting

Final reports should include:

- Selected approach
- Files changed
- Tests run
- Remaining risk or follow-up

Do not include raw private reasoning transcripts. Summarize decisions and evidence.
