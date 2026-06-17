# Refactor Root README into Overview + Canonical Docs

**Date:** 2026-06-17  
**Status:** Pending Approval

## Problem Statement

`README.md` is 451 lines and mixes human-facing project overview with architecture rules, API endpoint inventory, WhatsApp command details, database schema, testing strategy, and error handling. Much of this content duplicates or overlaps with `docs/rules/` and cell `AGENTS.md` files, and some details (WhatsApp commands, testing counts) exist only in `README.md`.

## Solution

Reduce `README.md` to a concise project overview and getting-started guide. Move architecture conventions, API details, WhatsApp behavior, database design decisions, testing strategy, and error handling to their canonical homes in `docs/rules/` and cell `AGENTS.md`.

## Decision: README Scope

`README.md` will keep only human-facing project information:

- Title and one-paragraph description
- Tech stack
- Quick start (Docker and local development)
- Environment variables
- Makefile targets
- Project status and known limitations
- Links to `AGENTS.md` and `docs/rules/`

## Content Mapping

| README Section | Destination | Action |
|---|---|---|
| Architecture principles | `docs/rules/01-architecture.md` | Remove from README; link to existing doc. |
| Project structure tree | `AGENTS.md` | Remove duplicate; root `AGENTS.md` already covers this. |
| API endpoint inventory | Cell `AGENTS.md` files | Remove from README; cells already own endpoints. |
| WhatsApp commands, format, examples, default actions | `apps/api/internal/activity/AGENTS.md` | Add a `## WhatsApp Commands` section there. |
| WhatsApp business rules | `apps/api/internal/activity/AGENTS.md` | Merge into existing business rules; avoid duplication. |
| Database schema tables/indexes | Cell `AGENTS.md` + `docs/rules/03-database.md` | Remove from README; already decentralized. |
| Database design decisions | `docs/rules/03-database.md` | Add a `## Design Decisions` section if not already covered. |
| Testing strategy and test counts | `docs/rules/05-testing.md` | Move layer table and descriptions there. |
| Error handling status code mapping | `docs/rules/04-api-and-webhook.md`, `06-development.md` | Already in `04-api-and-webhook.md`; add `ErrUnauthorized` to `06-development.md` if missing. |
| Tech stack, quick start, env vars, Makefile, status | `README.md` | Keep. |

## Execution Sequence

1. **Move WhatsApp details to `activity/AGENTS.md`**
   - Add `## WhatsApp Commands` with command format, examples, default system actions, and business rules.
   - Skip content already in `activity/AGENTS.md`.

2. **Move testing details to `docs/rules/05-testing.md`**
   - Append test layer table and counts.
   - Rename "Handler tests" to "Controller tests" to match current naming.

3. **Move database design decisions to `docs/rules/03-database.md`**
   - Add `## Design Decisions` covering interleaving, denormalized `company_code`, SQL aggregations, atomic check-out.

4. **Move `ErrUnauthorized` note to `docs/rules/06-development.md`**
   - Add the 401 mapping if missing.

5. **Rewrite `README.md`**
   - Keep title, description, tech stack, quick start, env vars, Makefile targets, project status, links.
   - Remove all sections mapped above.
   - Add links to `AGENTS.md` and relevant `docs/rules/` files.

6. **Verify no content is lost**
   - Cross-check each removed section against its destination.
   - Confirm `README.md` contains no domain-specific details.

## Acceptance Criteria

- [ ] `README.md` is under 120 lines and contains only overview, setup, env vars, Makefile, status, and links.
- [ ] `README.md` links to `AGENTS.md` and to relevant `docs/rules/` files.
- [ ] WhatsApp command details live in `apps/api/internal/activity/AGENTS.md`.
- [ ] Testing strategy details live in `docs/rules/05-testing.md`.
- [ ] Database design decisions live in `docs/rules/03-database.md`.
- [ ] `ErrUnauthorized` mapping lives in `docs/rules/06-development.md`.
- [ ] No API endpoint inventory, database schema, architecture rules, or WhatsApp details remain in `README.md`.
- [ ] No duplicated content between `README.md` and `docs/rules/` or cell `AGENTS.md` files.
- [ ] All internal links resolve correctly.

## Related Documents

- [README.md](../../../../README.md)
- [AGENTS.md](../../../../AGENTS.md)
- [docs/rules/01-architecture.md](../../../../docs/rules/01-architecture.md)
- [docs/rules/03-database.md](../../../../docs/rules/03-database.md)
- [docs/rules/04-api-and-webhook.md](../../../../docs/rules/04-api-and-webhook.md)
- [docs/rules/05-testing.md](../../../../docs/rules/05-testing.md)
- [docs/rules/06-development.md](../../../../docs/rules/06-development.md)
- [apps/api/internal/activity/AGENTS.md](../../../../apps/api/internal/activity/AGENTS.md)
- [2026-06-17-decentralized-rulebook-design.md](./2026-06-17-decentralized-rulebook-design.md)
