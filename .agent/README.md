# .agent Project Format

`.agent` is the model-agnostic project context format for this repository. It is intended to work across agents, IDEs, CLI tools, and model providers.

Use `.agent` for the durable rules every agent should understand before making changes. Use optional agent folders, such as `.kiro`, for specialized workflows, generated specs, or tool-specific state.

## Load Order

Agents should read project context in this order:

1. `AGENTS.md` - repository-wide operating rules
2. `.agent/project.md` - project identity, architecture, and commands
3. `.agent/workflow.md` - default AI-DLC workflow
4. Optional workflow folders, such as `.kiro/`, when a task needs their specs or protocols

## Directory Contract

```text
.agent/
  README.md              # format contract
  project.md             # project-specific context
  workflow.md            # default AI-DLC workflow
  templates/
    feature.md           # feature/spec template
    decision.md          # architecture decision template
```

## Scope Rules

- `.agent` contains stable project memory and agent instructions.
- `.kiro` remains an optional spec-driven agent workflow for this repository.
- Generated tool state should not be placed in `.agent` unless every agent needs it.
- Secrets, credentials, and machine-local paths do not belong in `.agent`.

## Portability Rule

For another project, copy the `.agent` directory and rewrite `project.md`. Keep `workflow.md` unless that project needs a different development lifecycle.
