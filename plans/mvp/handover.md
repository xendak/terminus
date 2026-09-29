# Handover — mvp

## State

T5 landed (session 7): the read path is complete — `store/reads.go` (one SQL
aggregation query per read service, parameters pivoted per query, percent
rounded once to 3, cost once to 2, NULL cost without distance) and
`app/reads.go` (GetRoute, ListRoutes with current-month defaults,
GetDashboardByDay/Month/Period, query-level driver scoping). Perf: 36
months of synthetic data via a Go test helper; 12-month dashboards answer
in ~5–7 ms; EXPLAIN ANALYZE shows `route_route_date_idx` on selective
windows. TestMain applies the golden seed after truncation. Cluster up;
test DB migrated.

## Next

**T6. Auth + roles — the matrix is enforced in the service layer**
(`plans/mvp/plan.md`).

- Step 0: baseline green (below); role matrix in `operations.md` read
  (it is — this conversation holds the full file; re-check on disk).
- Plan (read): `docs/spec/operations.md` (error model + role matrix),
  `docs/spec/architecture.md` (security section).
- Do: Login/Logout services (bcrypt verify, HMAC session cookie per
  `architecture.md`), a session type carried through a context, role
  checks in services (not only middleware), driver scoping enforced in
  the queries. A table-driven test walking the whole matrix: every
  operation × every role → allowed/denied as the spec says.
- Verify: `nix develop -c bash -c 'cd backend && go test -count=1
  ./internal/...'` green; the matrix test output shows every cell.
- Stop-when: W6 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'scripts/testdb.sh'            # fresh migrated test DB
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 ./internal/...'
```

`-count=1` matters: go's cache cannot see the DB rebuild.

## Facts this task needs

- `app.Actor{UserID, Role}` already threads through every audited or
  provenance-taking service (T4); enforcement was deliberately deferred to
  this card. Services currently trust the actor — T6 adds the matrix
  checks at the service boundary and forces driver scoping from the
  session role.
- Read services take an optional `DriverUserID *uuid.UUID` filter — that
  is the seam where T6 forces `session.UserID` for drivers.
- bcrypt is already a dependency (x/crypto v0.57.0, used by CreateDriver);
  password hashes are cost 10. The golden seed's demo password is
  `stoptime-dev` (notes.md "T2 session") — usable for login tests.
- Session cookie: HMAC-signed per architecture.md's security section
  (read it); `SESSION_KEY` env documented in `.env.example` (dev default
  in code until this card replaces it).
- Error model additions needed: `ErrUnauthenticated`, `ErrForbidden`
  (operations.md error table) — define in app, same sentinel pattern.
- Errors already aliased: ErrDriverDateConflict, ErrDuplicateEmail,
  ErrNotFound (store), ErrDepartureBeforeArrival (domain).
- Integration tests connect to `TEST_DATABASE_URL`; TestMain truncates
  then applies `db/seed/golden.sql` — auth tests create their own users
  via services; golden demo users exist too.
- Test order: reads_test.go before service_test.go (alphabetical); a new
  auth/matrix test file sorts FIRST if named e.g. `auth_test.go` — keep
  its expectations independent of later suites' data (distinct emails/
  dates), or name it `zmatrix_test.go` to run last.

## Open risks (subset relevant to T6)

- The matrix test must show EVERY cell in its output (the card demands
  it) — design the table so a t.Logf run prints operation × role →
  allowed/denied compactly.
- "Role checks in services (not only middleware)": T7's handlers will
  construct Actor from the session; the service-level check is the
  authority. Do not duplicate the matrix in two places — one table in
  the service layer, referenced by middleware.

## Out of scope

No HTTP (T7 brings handlers/cookies wiring — the session SERVICE this
card builds must not import net/http; the cookie encoding is a pure
function). No UI. No new tables — sessions are HMAC cookies; if a
`session` table becomes required, that is a spec change first
(data-model.md documents the extension point).
