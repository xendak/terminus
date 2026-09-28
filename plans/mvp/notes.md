# Notes — mvp

Verified facts. A fact is recorded here the moment it is discovered, so no
session re-derives it. Claims of completion are not facts; they are re-verified
every session per `docs/method.md`.

## Environment (verified 2026-09-28, bootstrap session)

- OS: NixOS. `nix` 2.34.8 with flakes enabled (`nix-command flakes` in both
  `/etc/nix/nix.conf` and `~/.config/nix/nix.conf`).
- **No system `psql`, no `pg_isready`, no PostgreSQL service, no Docker.**
  Only a `postgresql-14.7-lib` exists in the store (some other package's
  dependency). Postgres comes from the devshell: `nix eval --raw
  nixpkgs#postgresql.version` → 18.6.
- Go 1.26.7 (system). Node v24.19.0 (unused by this project). Python 3.14.7
  (unused). Zig 0.16.0 (unused). git 2.55.0.
- Nothing listens on 5432 at bootstrap.

## Decisions (with the user, bootstrap session)

- Stack: **Go backend** (stdlib `net/http` + `html/template`, pgx v5, x/crypto
  bcrypt), **SSR + htmx** frontend, **Chart.js vendored** for charts, plain SQL
  migrations applied by psql, PostgreSQL from the Nix devshell, repo-local
  cluster in `.pg/`. No Docker, no npm, no ORM, no build pipeline.
- The user's stated motive: the backend must survive a later frontend switch
  (htmx → React). Their five caveats are normative layering rules in
  `docs/spec/architecture.md`: (1) handlers only parse, call a service, and
  format; (2) services take and return plain structs so `DashboardByMonth` can
  feed HTML or JSON; (3) queries and aggregations live in SQL, so charts receive
  aggregate data only; (4) the API contract is written per operation
  (create route, register arrival/departure, get dashboard by period), not per
  endpoint; (5) screens are defined by use case and states (route builder:
  empty, adding points, reordering, error), not by page fragments.
- Language: engineering artifacts in English. UI English by default, one
  labels map per screen for a possible translation layer later (explicitly not
  a priority until the MVP runs). User-entered data (names, addresses, notes)
  accepted in Portuguese. Golden seed keeps the Portuguese addresses of
  `tp.md` section 5 verbatim.
- Working title **StopTime** for module/DB/internal naming. The graded product
  name is chosen in T11; branding must be one rename commit away.

## Golden fixture (from `tp.md` section 5 — do not re-derive)

- Route A: stops at Rua Peru 55 (15 min), Rua X 5 (10 min), Av. João César
  (50 min) → total **75 minutes**.
- Route B: 10 + 5 + 26 → **41 minutes**. Route C: 5 + 10 + 30 → **45 minutes**.
- Stop 1 of every route contributes **0** (RN01), even if seeded with
  timestamps.
- Journey percent route A at 8h standard: 75/480 = **15.625%** (RN04).
- All three routes on one date → day/month/period totals all read **161**.
- With `min_stop_minutes = 6`: route B's 5-minute stop drops out → B totals
  **36 minutes**.

## tp.md reading notes

- `tp.md` is the professor's brief and is **immutable** in this repo. It is
  the requirements ground truth; our spec set translates and sharpens it.
  Conflicts: `tp.md` wins, spec updated, recorded here.
- Graded extras (up to 2 points): best product name, best publicity campaign.
  Both are deliberately last-mile cards (T11), after function.
- The brief's "Gerente" entity includes "equipe sob responsabilidade"
  (team under responsibility). MVP simplification: single company, all
  managers coordinate all drivers. Recorded as accepted simplification in
  `data-model.md`; extension point if the professor pushes.

## Open questions

1. **Professor's expected format for the part-1 document** (Projeto
   Preliminar): diagram tool, document format (PDF?), language (pt-BR?).
   Ask by email (Laudares@pucminas.br). T10 depends on this; if unanswered,
   Mermaid-rendered Markdown + PDF export is the default assumption.
2. Team composition (dupla ou trio): who the teammates are and whether they
   will work through this plan folder too. The README tells them to.
3. Whether the professor wants the 8h journeyday percent interpreted per route
   or per driver-day. `tp.md` RN04 says "indicadores de tempo parado" based on
   the 8h day; we compute per route (a driver has one route per day, RN05, so
   the two readings coincide in practice). Recorded in `business-rules.md`.
