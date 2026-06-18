# Agent Workflow Rules

Rules that apply to every agent working on this codebase.

## Orchestrator Agents

If you are acting as an orchestrator, invoke the `subagent-driven-development` skill before starting implementation work. Delegate independent tasks to specialist subagents rather than doing all work in the orchestrator thread.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) format for all commit messages:

```
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`.

## Local Deployment

If the local stack is running when you change application code, rebuild and redeploy so the changes take effect. In the `deployments/` directory:

```bash
make docker-build
make docker-up
```

Or restart the full stack:

```bash
make docker-restart
```

Verify the new containers are healthy before continuing.
