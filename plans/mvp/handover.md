# Handover — mvp

## State

T6 landed (session 8): sessions (HMAC-SHA256 cookie `st_session`, 12h,
pure encode/decode), Login/Logout, and the full role matrix enforced in
every service (Actor threading complete; own-route checks; scoped reads
force the driver filter). The matrix test prints all 28 operations × 4
roles, every cell a real call. Cluster up; test DB migrated. The Go
module's only consumer is `cmd/server` with `/healthz` — no HTTP layer
exists yet.

## Next

**T7. HTTP shell + adapters + directories — pages exist, guardrails
hold** (`plans/mvp/plan.md`).

- Step 0: baseline green (below); `backend/internal/httpapi` does not
  exist; vendoring htmx/chart.js/CSS needs one-time network access (or
  bring the files).
- Plan (read): `docs/spec/architecture.md` (layering rules + dependency
  budget), `docs/spec/screens.md` (Login + Directories),
  `docs/spec/operations.md` (transports — full file already read this
  conversation; re-check on disk).
- Do: `internal/httpapi` — router (net/http method patterns),
  middleware (session load, role gate), template engine (html/template
  layouts + partials), sentinel→HTTP error mapping, flash messages.
  Vendor htmx, chart.js, and one classless CSS into
  `backend/web/static/`. Screens: Login, Directories (drivers,
  managers, locations) per the state tables in screens.md, plus the
  JSON `/api/*` mirrors for their operations.
- Verify: build/vet/test green; a scripted curl walkthrough (login as
  admin → create driver → list drivers) greps expected markers;
  `grep -rn "https://" backend/web/templates/` and
  `grep -rn "SELECT" backend/internal/httpapi/` both empty.
- Stop-when: W7 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'scripts/testdb.sh'            # fresh migrated test DB
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 ./internal/...'
```

## Facts this task needs

- **Session wiring:** middleware decodes with
  `app.DecodeSession(key, cookieValue, time.Now())`, builds the Actor
  via `app.ActorFromSession`, and may stash the session with
  `app.WithSession` for handlers. `app.Services.SessionKey` signs new
  cookies (`svc.Login` returns the cookie VALUE — set it with name
  `app.SessionCookieName`, HttpOnly, SameSite=Lax, Secure when TLS).
  `cmd/server/main.go` must read SESSION_KEY (a fixed dev default when
  unset — the `.env.example` documents it) and construct
  `app.New(store, key)`.
- **Error mapping (operations.md table):** ErrBadInput 400,
  ErrValidation 422 (FieldError carries field+reason for inline form
  errors), ErrUnauthenticated 401, ErrForbidden 403, ErrNotFound 404,
  ErrDriverDateConflict 409, ErrRouteClosed 409,
  ErrDepartureBeforeArrival 422, ErrDuplicateEmail 409.
- **Role gate:** services enforce the matrix — middleware may pre-check
  for UX (hide links, redirect to login), but the service is the
  authority. Handlers: parse → service → format; no SQL in httpapi, no
  business rules in handlers.
- Transports per operation are listed in operations.md (htmx form
  paths + `/api/*` JSON mirrors).
- Assets are LOCAL (vendored); templates contain no external URLs.
- Handlers construct `app.Actor` from the session — never from request
  input.
- Integration tests for httpapi run against `stoptime_test` via
  `TEST_DATABASE_URL`; the app suite's TestMain truncates and applies
  the golden seed — an httpapi test file sorts between reads/service
  tests alphabetically; keep its data on its own dates or expect
  golden rows in list views.

## Open risks (subset relevant to T7)

- Vendored asset versions: pick and RECORD the htmx + Chart.js versions
  in notes.md (the budget names them, not the versions).
- html/template semantics: check method-pattern routing and template
  parsing behavior against the devshell Go 1.26 docs, not memory.
- The curl walkthrough runs against a live server on the test DB —
  start it on a scratch port, drive with the demo admin
  (admin@stoptime.dev / stoptime-dev), kill it in the same session.

## Out of scope

No screens beyond Login + Directories (route builder/tracker = T8,
dashboard/history/params/export = T9). No CSV export. No SPA. No new
services — T7 wires what exists.
