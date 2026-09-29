# Plans

The scheduler for this repo's multi-session work. Method: `docs/method.md`
(the rulebook). Session protocol: `AGENTS.md` (read it at session start).

Opening prompt, one copy-paste:

**"Follow the AGENTS.md session protocol — resume the active plans/ task."**

## Task index

| Task | Status | Summary |
| --- | --- | --- |
| `mvp/` | [~] | Build the Terminus MVP (working title StopTime) per `docs/spec/`: 11 session cards. T1–T10 landed (devshell, db scripts, schema + golden seed, pure domain, audit-in-tx writes, reads/aggregation, auth + role matrix, HTTP shell, builder + tracker, dashboard/history/params/audit/export; T10 especificação reconciled + diagrams re-rendered — human sign-off pending). Next.js client in `frontend/` and the /sobre campaign page are being delivered in parallel. Next: T11 branding/campaign/demo/final acceptance (+ collect the T10 sign-off). |

Status legend: `[ ]` todo, `[~]` active, `[x]` done.

## Reading order for a cold session

1. `AGENTS.md` (session protocol)
2. This file (which task is active)
3. `mvp/handover.md` (resume card, provenance-checked per the protocol)
4. The card in `mvp/plan.md` the handover names

Do not read the whole spec set cold. Each card's **Plan (read)** names the
exact spec files that session needs.
