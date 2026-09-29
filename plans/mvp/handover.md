# Handover — mvp

## State

**The MVP task is complete** except one item that cannot happen inside an
agent session: the **human review sign-off of `docs/especificacao.md`**
(W10) by the user and at least one teammate is **PENDING**. It has not been
faked anywhere.

T11 landed (session 13): Terminus branding finished (legacy htmx
title/wordmark, prose and comments; identifiers untouched), campaign
material in `docs/campanha.md` with screenshots in `docs/campanha/`
(dashboard, driver tracker at 375 px, closed route, `/sobre`), README final
(pt-BR summary, stack, run paths, demo logins, tests), `tp.md` §10
acceptance criteria 1–4 demonstrated with literal output (progress.md,
session 13), full parity green (Go tests, `pnpm lint`, `pnpm build`,
Playwright 15/15). Tagged `plans/mvp/T11` locally; nothing pushed.

Session 14 (post-T11 audit against `tp.md`) closed the remaining gaps and
reconciled the especificação: RNF06 (driver document masked for managers,
`AnonymizeDriver`/`AnonymizeManager`, `UpdateManager`), sessions
re-validated per request (deactivated/anonymized → 401 on the next
request), `UpdateLocation` audited (`update_location`) with `route_stop`
location snapshots taken at add time (migration 0003), and dashboard
drill-down to pre-filtered History (`by_driver.driver_user_id`). The
especificação now has §8.1 (tp.md §8 mapping), §10.1 (LGPD), §10.2
(deliverables map, incl. "entrada de pedidos"), §10.3 (acceptance
criteria) and §10.4 (decisions D1–D7, D1 = single company, no team).

## Next

No task card is left. Remaining, for humans:

1. The user and a teammate review `docs/especificacao.md` (and, ideally,
   `docs/campanha.md`). Record the sign-off verbatim (who, date, what they
   said) in a new `progress.md` entry; any doc change they ask for follows
   the naming policy in `notes.md` (never translate an identifier; an
   edited `.puml` re-renders every `.svg` in the same commit).
2. Push `main` and the tag when the user says so.
3. On demo day, rerun `scripts/dev-seed.sh` (the demo data is relative to
   CURRENT_DATE).

## Baseline commands

```
git status                                            # clean
# toolchain: nix develop, or Ubuntu PATH=$PATH:/usr/local/go/bin:/usr/lib/postgresql/18/bin
eval "$(scripts/db-up.sh)" && scripts/testdb.sh && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...
cd frontend && pnpm lint && pnpm build && pnpm test:e2e   # both servers up; then scripts/dev-seed.sh
java -jar ~/.local/share/plantuml/plantuml.jar -tsvg docs/especificacao/diagrams/*.puml   # only if a .puml changes
```

## Facts a follow-up needs

- Demo logins (dev seed): admin@, manager@, driver-a@, driver-b@,
  driver-c@stoptime.dev, password `stoptime-dev`.
- Golden: A 75 / B 41 / C 45, day 161, route A 15.625%, day/period 11.181%.
- Acceptance 4 demo on the dev seed: route of driver C on the day before
  the seed date, 56.3 km, cost 34.29 → 39.41 at fuel 7.00; journey 14.375%
  → 11.5% at 10 h. Revert both after demonstrating.
- Shared machine: never kill generic `next`/`go` processes; restart only
  by exact PID; `-p 1` and a fresh `testdb.sh` before trusting test output.
- The e2e suite writes into the dev DB — reseed afterwards.
- Migrations are now 0001–0003; a dev DB created before session 14 needs
  `scripts/migrate.sh` (0003 backfills existing stops' snapshots).
- Route detail, history and CSV show the stop's snapshot address
  (`address_snapshot`), not the location's current one — by design (D7).
- Remaining session tradeoff (documented, accepted): logout cannot revoke
  a copied cookie of a still-active user before its 12 h expiry.

## Open risks

- The sign-off could ask for doc changes; see the naming policy.
- `origin/main` is behind local `main` (not pushed by instruction).

## Out of scope

Route optimization, payroll/ERP, telemetry, native apps (tp.md 3.2);
Docker; ORMs; Node outside `frontend/`. No new features.
