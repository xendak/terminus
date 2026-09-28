# Handover — <task>

Rewritten at every session end, never appended. Freshness is provenance: the
commit that last touched this file must be HEAD (check with
`git log -1 --format=%H -- <task>/handover.md` vs `git log -1 --format=%H`).

## State

What is true now: what exists, what is green, what is open.

## Next

The next card: number, name, and its plan/do/verify, tightened from
`plan.md`. If mid-card: the open checklist file, and the first unchecked
sub-item. If mid-red: the debug state (literal failing output, bisection
position, hypotheses ruled out, leading hypothesis, the single next check,
anything tried and reverted).

## Baseline commands

What "healthy" means for the next session, actually runnable now. Run these
before new work; a red baseline is the session's first task.

## Facts this task needs

The subset of `notes.md` the next session will use, lifted here.

## Open risks

Only the ones relevant to the next card, with their checks and sanctioned
responses (full list lives in `plan.md`).

## Out of scope

What the next session must not chase, with the evidence it is pre-existing or
another task's fault.
