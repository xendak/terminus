# Plans

The scheduler for this repo's multi-session work. Method: `docs/method.md`
(the rulebook). Session protocol: `AGENTS.md` (read it at session start).

Opening prompt, one copy-paste:

**"Follow the AGENTS.md session protocol — resume the active plans/ task."**

## Task index

| Task | Status | Summary |
| --- | --- | --- |
| `mvp/` | [x] | Build the Terminus MVP (working title StopTime) per `docs/spec/`: 11 session cards, all landed (T1–T11: devshell, db scripts, schema + golden seed, domain, audited writes, reads/aggregation, auth + roles, HTTP shell, builder + tracker, dashboard/history/params/audit/export, especificação + diagrams, branding/campaign/final acceptance; Next.js client in `frontend/`). Done **except the human review sign-off of `docs/especificacao.md`** (user + teammate), still pending outside any agent session — see `mvp/handover.md`. |

Status legend: `[ ]` todo, `[~]` active, `[x]` done.

## Reading order for a cold session

1. `AGENTS.md` (session protocol)
2. This file (which task is active)
3. `mvp/handover.md` (resume card, provenance-checked per the protocol)
4. The card in `mvp/plan.md` the handover names

Do not read the whole spec set cold. Each card's **Plan (read)** names the
exact spec files that session needs.
