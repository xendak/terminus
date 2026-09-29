# Handover — mvp

## State

T10 landed (session 12): `docs/especificacao.md` reconciled with the
finished implementation (product name Terminus, CurrentUser, RN04
per-route base 11.181%, ER = the 8 real FKs of 0001 + 0002, new section 11
on the architecture), all six PlantUML diagrams re-rendered (PlantUML
1.2026.8), spec set updated for the Next.js frontend decision
(architecture.md, screens.md, product.md, data-model.md, use-cases.md),
AGENTS.md and README aligned. Traceability walk green: every RF01–RF12,
RNF01–RNF06, RN01–RN07 in the matrix; every UC names its operations, all
defined in operations.md.

**Still open from W10: the human review sign-off (user + at least one
teammate) is PENDING** — not faked. T11 collects it.

In parallel, other agents are delivering (not part of T10, not committed
by it): the **Next.js client in `frontend/`** (App Router, TypeScript,
Tailwind, pt-BR UI, rewrite proxy :3210 → Go :8080), including the public
**`/sobre` campaign page**. The **demo seed** already landed
(`db/seed/demo/demo.sql` + `scripts/dev-seed.sh`, commit 993a518).

## Next

**T11. Name, campaign, demo, final acceptance — the finish line**
(`plans/mvp/plan.md`, card updated this session).

- Step 0: baseline green; W1–W10 crossed off in git (W10 carries the
  pending sign-off note); confirm `frontend/` is committed by its owner
  and runs (`cd frontend && pnpm install && pnpm dev`) against the Go
  server before judging any screen.
- Plan (read): `tp.md` (whole file, one last time),
  `docs/spec/product.md` (deliverables map), `frontend/README.md`.
- Do: finish the Terminus branding in the UI (titles, wordmark) — docs
  already say Terminus; never rename identifiers (`stoptime` module/DBs,
  `st_session`, `@stoptime.dev` demo logins). Review the `/sobre`
  campaign page against the graded extra (pitch, three screenshots,
  short post text). Run the demo seed (`scripts/dev-seed.sh`). Final
  acceptance of `tp.md` §10, item by item, through the Next.js client,
  literal output in progress. **Collect the W10 human sign-off** (user +
  one teammate) and record it verbatim. README final. Tag
  `plans/mvp/T11` (only if the team lead asks — this session was told
  not to tag or push).
- Verify: the four acceptance criteria demonstrated in-session with
  literal outputs; sign-off recorded; `git status` clean; `git log -1` is
  the finish commit.

## Baseline commands

```
git status                                            # clean (frontend/ may be untracked until its owner commits it)
# toolchain: nix develop, or Ubuntu PATH=$PATH:/usr/local/go/bin:/usr/lib/postgresql/18/bin
eval "$(scripts/db-up.sh)" && scripts/testdb.sh && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...
java -jar ~/.local/share/plantuml/plantuml.jar -tsvg docs/especificacao/diagrams/*.puml   # only if a .puml changes
```

## Facts this task needs

- Golden: A 75 / B 41 / C 45, day 161, route A 15.625%, day/period
  11.181% (161 / (3 × 480)); min_stop 6 → B 36. Acceptance 4 demo:
  route cost changes with `fuel_price_brl` without code change (T9:
  48.72 → 56.00 at 7.00).
- Demo logins (dev seed): admin@, manager@, driver-a@, driver-b@,
  driver-c@stoptime.dev, password `stoptime-dev`. `dev-seed.sh` is
  relative to CURRENT_DATE — rerun on demo day.
- The Next.js client uses only `/api/*`; `CurrentUser` (`GET
  /api/auth/me`) is its boot check; JSON dates are `YYYY-MM-DD`; error
  body `{"error","field","reason"}`.
- Shared machine: other agents run servers/tests concurrently — never
  kill generic `next`/`go` processes; `-p 1` and a fresh `testdb.sh`
  before trusting test output.
- The httpapi SQL guard grep hits `mux.HandleFunc("DELETE /api/...")` —
  HTTP method, not SQL (false positive).

## Open risks

- Sign-off could surface doc changes: any edit to a `.puml` re-renders
  all SVGs in the same commit (naming policy in notes.md: never translate
  an identifier).
- `frontend/` is uncommitted at T10 time; T11's acceptance depends on it
  landing.

## Out of scope

Route optimization, payroll/ERP, telemetry, native apps (tp.md 3.2);
Docker; ORMs; Node outside `frontend/`. No new operations or schema
changes in T11 unless acceptance finds a real bug (fix spec + code in the
same commit).
