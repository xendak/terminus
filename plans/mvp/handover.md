# Handover — mvp

## State

T9 landed (session 11): all client screens exist — dashboard (three
cuts, Chart.js on aggregate series), history + corrections, params,
audit (ListAudit built per operations.md), CSV export (BOM + RFC
4180). loginRedirect complete. Acceptance criteria 2–4 demonstrated on
a live server. All guards hold. Cluster up, test DB migrated.

## Next

**T10. Part-1 specification document — final review and render**
(`plans/mvp/plan.md`). The deliverable exists (session 2); this card
reconciles it with the FINISHED implementation.

- Step 0: cards T1–T9 crossed off in git (check plan.md); the devshell
  provides `plantuml` (`nix develop -c plantuml -tsvg …`).
- Plan (read): `docs/especificacao.md`, `docs/spec/use-cases.md`,
  `tp.md` sections 4, 6, 10, `plans/mvp/notes.md` (naming policy +
  PlantUML facts).
- Do: walk every UC description and diagram label against the real
  operations, screens, and tables; fix drift on both sides in one
  commit. Identifiers are verbatim English — never translate one.
  Re-render and commit the SVGs. PDF only if the professor asks
  (docs/deliverables/); otherwise the markdown is the document.
- Verify: traceability walk — every RF01–RF12 and RNF01–RNF06 in the
  matrix or a documented non-UC decision; every UC names its
  operations; fresh render exits 0, empty error scan, SVG set matches
  the .puml set. Human review sign-off (user + at least one teammate)
  recorded in progress.
- Stop-when: W10 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...'
nix develop -c plantuml -tsvg docs/especificacao/diagrams/*.puml   # the render step
```

## Facts this task needs

- **Implementation drift to reconcile (discovered across T2–T9):**
  - migration 0002: `audit_log.entity_id` is now TEXT (the
    especificação ER was corrected in the same commit — verify it
    reads `text entity_id`).
  - The golden seed has FIVE demo users (admin, manager, drivers A/B/C
    — RN05 forces distinct drivers), not "three users one per role";
    the document must say what the seed does.
  - The driver_profile km_per_l override (driver B = 12.50) and the
    departure-point placeholder addresses are seed facts
    (notes.md T2).
  - Sessions are HMAC-SHA256 cookies (st_session, 12h), stateless;
    role checks in the service layer; ListAudit is admin-only.
  - Screens shipped exactly as screens.md's state tables; the
    dashboard series are SQL aggregates, Chart.js renders only.
- **PlantUML facts (notes.md, verified):** no `robustness` directive;
  boundary/control/entity render the icons; output name follows
  `@startuml <name>`, not the filename; derived attributes (/attr)
  work. Robustness syntax errors die at line 2 if you invent the
  directive.
- The especificação is pt-BR prose with VERBATIM English identifiers;
  actors carry the real role value ("Motorista (role: driver)"); UC
  titles are Portuguese mapping to English operations.
- Traceability targets: RF01–RF12, RNF01–RNF06 (tp.md §6), RN01–RN07
  (§4), UCs from use-cases.md; every one must appear in the matrix or
  as a documented decision.
- Sign-off: the user AND at least one teammate must review before the
  commit — schedule it; record it verbatim in progress.

## Open risks (subset relevant to T10)

- The SVG set must match the .puml set after re-render — diff the file
  names and mtimes; stale SVGs lie.
- A PDF export is OUT of scope unless the professor asks (open
  question 1 in notes.md, resolved: repo is the hand-in).

## Out of scope

No code changes beyond drift fixes the document demands (any real
spec-vs-code bug found → fix BOTH in this commit, like migration 0002
was). No T11 work (name/campaign/demo — next card).
