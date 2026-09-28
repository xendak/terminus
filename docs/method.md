# Plans: plan, execute, verify, hand over

How a project runs work that is too big for one session. A multi-session task is
not tracked in a person's or an agent's head; it is tracked in a folder of
markdown files under version control. Those files are the memory of the work. A
session that starts cold can pick up exactly where the last one stopped, without
re-deriving anything — and without taking the last session's word for it.

This document is the rulebook, and only the rulebook. It never holds project
state — no milestones, no verified facts, no session logs; those live in
`plans/`. It is project-agnostic: a master copy lives outside any project, and
each project carries a snapshot as `docs/method.md`. The `plans/<task>/` folders
it describes live inside whichever project is doing the work; this document
doesn't.

## When to use this

Use this for work likely to outlast a single session or context window: a new
subsystem, a large algorithmic or protocol effort, a feature that touches many
files and will be resumed after a gap. Small tasks — a fix, a bounded refactor —
do not need the apparatus. Do them directly: change, verify, commit.

The threshold is not the size of the change but its memory. If the next session
would need to remember how the work is decomposed, why a decision was made, and
where exactly the work stopped, it goes in a plan folder.

## The root principles

**Context is disposable; the repository is the memory.**

A session's context is lost on compaction, at session end, and whenever a
different agent picks up the work. Nothing a later session needs may live only
in context. The goal, the verified facts, the decomposition, the current
position, the baseline state — all of it lives in `plans/<task>/`, committed to
git.

**A session cannot be trusted to accurately report itself.**

"Done," "tests passed," "stage verified" — a session's own narration of its own
work is a claim, not a fact, exactly as a person's account of a task they just
finished should be spot-checked before it's relied on. This isn't distrust of
any particular agent; it's a property of self-report generally. The only things
a later session (or a human) may treat as settled are what git itself shows: a
commit that exists, a diff that matches what the card promised, and a verify
command that _this_ session actually ran and saw pass — not one a previous
session said passed. `handover.md` and `progress.md` are hypotheses about state
until the baseline command confirms them fresh, every session, no exceptions for
a confident-sounding handover.

From these two principles the rest follows:

- **Verify is woven in, not batched.** Every work item names the command that
  proves it, and the check runs before the next item starts. A mistake is caught
  near the work that made it — and near the session that made it, before that
  session's own confidence about it gets baked into a handover.
- **The working tree is the intent signal.** Each completed stage commits, plan
  files included. A clean tree means the work stopped on purpose; a dirty tree
  means it stopped mid-stage. The open checklist plus the last commit mark
  exactly where.
- **One task per session, sized to finish in one sitting.** A task is a
  self-contained plan → do → verify → stop loop, so no session has to hold more
  than one task in its head, and no session's self-report has to be trusted
  across more than one task's worth of claims.
- **Facts are recorded once.** Discovery is the expensive part of a session. It
  is written down in the notes with a "must not rediscover" framing, so the next
  session performs a read instead of a re-derivation — but a _fact_ (a protocol
  quirk, a confirmed API shape) and a _claim of completion_ are different
  things, and only the latter needs re-verifying every session.
- **The plan is a living document.** When understanding improves, the plan is
  re-scoped, and the re-scoping is itself recorded as a session — a no-code
  session that re-cuts the remaining work into session-sized cards is a normal
  and useful kind.

## Git

Everything above assumes the `plans/` folder and the code it plans live in one
real, local git repository — not a folder of files that happens to be backed up
somehow. If the project isn't a git repo yet, `git init` is the first act of the
first session, before any plan file is written.

This is deliberately local-first: a remote isn't required for the discipline to
work. Commits exist to give the _next local session_ (or a human) something to
check against, not to publish anything.

### Commit shape

- One commit per completed stage — a task-card's Verify going green, or a
  sub-item in a `step_<n>_progress.md` closing. Never squash several stages into
  one commit (it destroys the "where exactly did it stop" granularity), and
  never split one verified stage across several commits a later session can't
  tell apart.
- Message format: `<task>: <stage> (plan T<n>)`, so `git log` reads as the
  project history of the work without needing to open any plan file.
- The end-of-session commit always includes the plan files — `handover.md`, the
  `progress.md` append, any `step_<n>_progress.md` checkbox — alongside the code
  change. Plan-file drift from the commit it describes is exactly the staleness
  this whole method exists to prevent.

### Commits are append-only, like the log

`git log` on this project plays the same role `progress.md` does: history that
doesn't get rewritten. Don't amend or rebase a commit that anything points at —
a hash pasted into a `progress.md` entry, or the commit the handover's
provenance check anchors to. If a mistake surfaces, fix it forward in a new
commit, the same way a wrong `progress.md` entry gets corrected by a later
entry, never edited away. A handover whose provenance commit no longer exists in
`git log` is a broken resume, not a tidied-up history.

### Verifying a claim of "committed"

A commit "exists" only when `git log` shows it — a session asserting it
committed, without `git status`/`git log` confirming a clean tree at that
commit, is exactly the unreliable-self-report case the root principles call out.
And a handover cannot self-certify with a hash: **a file cannot contain the hash
of the commit that contains it**, since the hash covers the file's content.
Provenance is therefore derived from git, never declared in prose:

- A handover is current iff the commit that last touched it is HEAD:
  `git log -1 --format=%H -- plans/<task>/handover.md` equals
  `git log -1 --format=%H`. Equal → nothing has committed since the handover was
  written. Different → the repository moved since the handover was written —
  read `git log <that commit>..HEAD` before trusting anything else in the
  handover.
- The end-of-session ritual therefore ends with `git status` clean and
  `git log -1` showing the ritual's own commit — nothing to backfill, no amend.
  (Optionally tag a card's closing commit `plans/<task>/T<n>` so a specific
  completed stage can be checked out or diffed later — a convenience, not a
  requirement.)

## The artifacts

One scheduler file per project, one folder per task: `plans/README.md` and
`plans/<task>/`.

| File                                                      | Role                                                                                                                                                                                        | Updated                                                                                                   |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `README.md` (in `plans/`, one per project — not per task) | the scheduler: ordered task index, one line per task with status — `[ ]` todo, `[~]` active, `[x]` done — plus the project's opening prompt at the top                                                                      | rewritten freely — it is current state, like the handover, never history                                  |
| `plan.md`                                                 | the goal, work items as checkboxes, session-sized task cards, open risks, how to resume                                                                                                     | re-scoped as understanding grows; items crossed off as they land in git                                   |
| `handover.md`                                             | the resume card: current state, the next task's plan/do/verify, the baseline command, the facts that task needs — fresh by provenance (git-checked), not by a hash it declares about itself | **rewritten** (never appended) at the end of every session, in the same commit as the `progress.md` entry |
| `progress.md`                                             | append-only session log, newest entry on top: what landed, what was discovered, what is next, and the actual verify output (not just "passed")                                              | appended at the end of every session                                                                      |
| `step_<n>_progress.md`                                    | a sub-item checklist for one work item that outlasts a single step; every item carries a `*verify:*` line naming the command that proves it                                                 | crossed off as sub-items land; created from the task card when the task starts                            |
| `notes*.md`                                               | the verified fact set: protocol behavior, API shapes, format quirks, file locations, environment recipes. One file, or one file per topic when a topic grows                                | appended and corrected in place as facts are discovered                                                   |
| `references.md`                                           | external pointers: RFCs, docs, reference implementations, fetched recipes                                                                                                                   | mostly once, at research time                                                                             |
| other files                                               | artifacts produced while planning — expected outputs, captured requests/responses, reference captures — named for what they are                                                             | during research                                                                                           |

Two files carry the state, with different disciplines. `progress.md` is
append-only history: why things are the way they are, and — per the git section
above — what a claimed "green" verify actually showed. `handover.md` is the
current position: what to do next, fresh by provenance — the commit that last
touched it is HEAD. Mixing the two — appending state updates to the log, or
rewriting history — breaks the resume.

## Plan

### Research first

The plan opens with research, before any code. Read the sources the task depends
on and record the verified facts in the notes, so the work items are written
against what the task actually is, not what it is assumed to be.

Weigh sources in this order when they disagree: the reference implementations
and the real artifacts (the behavior of live servers, captured requests and
responses, the loaded module) win over documentation; official docs win over
specs read in isolation; specs win over assumptions. A fact the research cannot
settle becomes an open item in the plan, not a guess in the code.

Research also produces the verification target before the code exists: the
reference outputs the finished work must reproduce. The exact artifacts depend
on the task — captured expected responses, golden request/response pairs,
templates — all captured in the recon session, all used later as golden tests.

### Work items

The plan's body is work items as checkboxes, in order of dependency: recon,
scaffold, the core of the feature, the verification, wrap-up. Each item says
what lands and what proves it. Items are crossed off as they land **in git** —
not when the code is written, not when it works locally, and not because a
session says it landed.

### Task cards

Remaining work is written as task cards, one per session. Each card is a
self-contained loop:

```
### T2. <name> — <the deliverable in one line>

- **Step 0 (identify):** the existence and sanity checks the card depends
  on, before any Do — the Plan files exist and contain what the card
  assumes; ground-truth sources, fixtures, tools, and external paths exist
  where the card claims; names remembered "from memory" are confirmed. A
  failed check updates the card (and the notes), never the code.
- **Plan (read):** the exact files the task needs — a reference file, a
  sibling implementation, a contract. Only these.
- **Do:** what gets written, concretely.
- **Verify:** the commands that must be green, and what green means.
- **Stop-when:** the condition for ending the session (verify actually
  re-run and green in this session, committed, `git log -1` confirmed,
  handover rewritten).
```

Sizing rule: a card must be finishable in one sitting **and one context window**
— see "The context budget and the early stop." If a card will not, split it —
splitting is free, a session that ends mid-card with verify red is not. Cards
that depend on a prior card say so; cards that must not depend on a later card
say so deliberately — isolating one card on purpose, so a failure in a later
task can never be shaped like a failure in it, is a feature, not an accident.

A card also pre-writes the failure handling for its verify: the bisection order
when it is red, and the escape hatch — the sanctioned path for a known failure
class, so the session follows the plan instead of improvising. Where a card's
verify compares against a captured reference, name the sanctioned response
explicitly: if the mismatch is shaped like a difference in the capture itself,
re-capture the reference; do not suspect the implementation.

### Open risks

The plan ends with the risks that are known and accepted: the expected wrinkles,
the budget limits, the things deliberately left out of scope. A risk is a
prediction with a check attached — which pin or command will catch it, and what
the sanctioned response is.

## Execute

### Session start

A resuming session does six things, in order:

0. **Identify.** Confirm the session's inputs exist before reading them: the
   task folder and handover the scheduler names, the environment paths the
   current card touches (reference repos, wikis, tools, data dirs), the
   toolchain version. Checked with `ls` / `test -f`, not assumed. A missing
   input is a plan update, never a guess to code around.
1. `git status`. A clean tree means the last session committed a completed task;
   a dirty tree means it stopped mid-task — open the checklist the resume card
   names first.
2. Read `plans/README.md` — the active task is the topmost `[~]`, else the first
   `[ ]` — then that task's `handover.md`. It is the resume card: state, the
   next task's plan/do/verify, the baseline command, the facts that task needs.
3. Check the handover's provenance:
   `git log -1 --format=%H --
   plans/<task>/handover.md` versus
   `git log -1 --format=%H`. Equal means the handover is current. Different
   means the repository moved since the handover was written — read
   `git log <that commit>..HEAD` before trusting anything else in the handover.
4. Run the baseline command, in this session, regardless of what the handover or
   `progress.md` claims about the last session's verify. This is not a
   formality: a prior session's "tests passed" is a claim, not a fact, until
   this session sees it pass itself. If it's red, that is this session's first
   task: fix, commit, then continue.
5. Start at the first unchecked item of the next task. Read only the files its
   Plan line names. The notes hold the full fact set — open the section a card
   points at, not the whole file.

A project's `AGENTS.md` typically encodes this ritual as its session protocol;
the two must not disagree — when they do, whichever the agent read last wins,
which is how the method silently breaks.

`plans/README.md` carries the exact opening prompt for this ritual, so "continue
the work" costs one copy-paste.

### Working the card

Work only the task the card names. Do not reach for the next task's items, do
not fix adjacent things (record them instead — see "Out of scope" below).

A task that spans several commits keeps a checklist file,
`step_<n>_progress.md`, created from its card when the task starts. Its
sub-items are checkboxes, each with a `*verify:*` line naming the command that
proves it. The resuming session of a multi-commit task starts at the first
unchecked sub-item; `progress.md` names which checklist is open.

### The context budget and the early stop

A session's context window is a consumable — and the session is the worst
possible judge of how much of it is left. Token counts live in the harness, not
in the model; a session's self-estimate of its own fullness is precisely the
unreliable self-report the root principles refuse to trust. So the method does
not ask the session to watch a gauge it cannot see. It makes stopping so cheap
that precision doesn't matter:

- **A card is sized to one context window, not just one sitting.** The dominant
  budget cost is reading, not writing: a card whose Plan (read) names large
  references is a recon card and should do little else. Split read-heavy work
  from write-heavy work — the notes exist so later cards re-read nothing.
- **Early stop is a designed ending, not a failure.** Any session may be told to
  end now — typically by a user watching the harness's context meter, the one
  reliable gauge. The end ritual is identical at a card boundary and mid-card:
  cross off what's green, commit it, leave the open checklist exact, and
  rewrite the handover to name the open card, the open checklist, and the first
  unchecked sub-item. A clean early stop leaves the same repository shape as a
  finished card: clean tree, current handover, one obvious next action.
- **A compaction is a defect, and its recovery is the session-start ritual.** If
  the harness compacts mid-session, the summarized context the session continues
  on is a lossy claim with no provenance — treat everything before it as gone.
  Stop, re-orient from disk (`git status` → handover → open checklist →
  baseline), and re-run the current sub-item's `*verify:*` before crossing
  anything off: "in this session" restarted at the compaction. Record the
  compaction in the progress entry — it means the card was mis-sized for a
  window, and the fix is a re-scope, the same as any other red.
- **The session's own stop triggers are drift, not gauge.** A session cannot
  measure its own fullness, but it can observe its work diverging from its card
  — and drift is not a proxy for context pressure, it is the pressure: planned
  work converts context into commits, unplanned work converts context into more
  context. End the session (same ritual as an early stop) when any of these
  holds:
  - **read drift** — needing to understand files beyond the card's Plan (read)
    list: either the card is wrong (a discovery — record and end) or the session
    is lost (end; the next session re-anchors on the named files);
  - **debug spiral** — a red verify three fix-iterations deep without the
    bisection moving;
  - **card premise failure** — a Step 0 check failed or the reference
    contradicts the card: record the discovery and end; a re-scope session
    re-cuts the card. Exception: green, or one mechanical step from green, is
    finished, not stopped — the restart's fixed cost exceeds what it saves.
- **Calibrate from the log.** Every progress entry records how the session ended
  — card finished, early stop (at which sub-item), or compacted. Repeated early
  stops or compactions on one kind of card is the signal to re-cut that kind
  smaller.

### The verification loop

Verification happens at three granularities:

- **Per sub-item** — the `*verify:*` line. The item is crossed off only when its
  command is green, in this session, right now — not because a checklist says a
  prior session already ran it.
- **Per task** — the card's Verify. The task is done only when its verify is
  green **and committed**, and the commit is confirmed via `git log -1`.
- **Per stage close** — the parity check that matches the project's checks: in
  this project, `go build ./...` plus `go vet ./...` plus `go test ./...`
  against a freshly migrated test database, because the domain unit tests alone
  do not cover the store or handlers. Whatever the project's real check is,
  it's the same command every session runs, so "green" means the same thing
  across sessions.

The baseline command from the handover is run before new work in every session,
actually executed, not read about. A green baseline is the precondition that a
later red is caused by this session's change and not by inherited rot or a
previous session's inaccurate self-report.

## Verify: when it is red

A red verify is handled by the card, not by guessing. Three instruments:

- **Bisection order.** The card lists the stages of the pipeline in the order to
  check, cheapest suspect first — for a request-handling change, typically (1)
  the parsed request against the captured input, (2) the routing decision
  against the expected match, (3) the response bytes against the golden output.
  Each stage has a named oracle (the captured request, the expected route, the
  golden response).
- **Escape hatches.** A known failure class gets a pre-approved response. The
  response changes the _target_, not the _code_, unless the bisection says
  otherwise (a mismatch shaped like a capture difference: re-capture the
  reference, do not touch the implementation).
- **Out-of-scope list.** Failures that are pre-existing, in another subsystem,
  or a separate known issue are recorded — in the progress entry and in the
  handover — with the evidence that they are not this task's fault (identical at
  the baseline). The next session must not chase them.

What a red verify teaches is written down. The progress entry records the bug
the check surfaced — what was compared against what, why it was subtle, and the
test that now guards it — plus the actual command output that showed red, not a
paraphrase of it. That entry is worth more than the fix; it is the "must not
rediscover" section.

## Hand over

### Session end

Every session ends with the same ritual, in one commit:

1. Rewrite `handover.md` for the next session. Never append to it. It carries no
   commit hash of its own — it cannot: a file cannot contain the hash of the
   commit that contains it, since the hash covers the file's content. Its
   freshness is provenance, checked against git (see "Verifying a claim of
   'committed'").
2. Append the `progress.md` entry: what landed, what was discovered (must not
   rediscover), what is next, the literal output of the verify command that
   proves the stage is done — not "tests passed," but what the command actually
   printed or its confirmed exit status — and how the session ended: card
   finished, early stop (at which sub-item), or compacted.
3. `git add -A && git commit -m "<task>: <stage> (plan T<n>)"` — the work, both
   plan files, and any step checklist in one commit.
4. `git log -1` and `git status`: the ritual's commit is HEAD and the tree is
   clean — confirmed, not asserted. If either is false (a forgotten file, a
   wrong message), fix forward in a follow-up commit. There is no amend
   exception, because there is nothing to backfill.

### What the card carries

- **State + provenance.** What is true now. The handover does not declare its
  own commit; freshness is checked — the last commit that touched `handover.md`
  must be HEAD. If it isn't, the repository moved since the handover was
  written: no judgment, just comparison.
- **Next.** The next task's plan/do/verify, copied or tightened from the card in
  `plan.md` — including which checklist to create or resume.
- **Baseline commands.** What "healthy" is, per granularity.
- **Facts this task needs.** The subset of the notes the next task will use,
  lifted up so the session does not have to go looking.
- **Debug state (when stopped mid-red).** The handover's Next section carries
  the investigation so the next session re-enters at the last junction: the
  literal failing output, the bisection position and what each stage showed,
  hypotheses tested and ruled out, the leading hypothesis, the single next
  check, and anything tried and reverted (so it is not retried).
- **Open risks** relevant to the next task, and the **out-of-scope list**.

### Why the two-file discipline

`progress.md` answers "why is it this way" — append-only, so the reasons for
decisions survive, and so does the actual evidence a verify passed, not just the
claim. `handover.md` answers "what now" — rewritten, so it is never stale within
itself, and git-checked for provenance so it's never trusted purely on its own
say-so either. The provenance rule (the commit that last touched the handover is
HEAD) plus the clean-tree rule (stages commit) mean that at any moment the
repository alone answers: where is the work, is it healthy, and what happens
next — without asking any session, past or present, to be believed.

## Completing the task

The last card's Stop-when is the finish line: the parity check green in this
session, the plan's items all crossed off, a final `progress.md` entry with real
verify output, and one more commit confirmed via `git log -1`. After that the
folder is a record, not a working state. Leave it: it is the reference for how
this kind of work is done in this repository, and the next similar task copies
its shape.

## Applying this to another project

The invariants — keep all of them:

1. The plan folder is under version control in the same **local git repository**
   as the code it plans. If that repository doesn't exist yet, creating it is
   step zero. `plans/README.md` is the scheduler: one line per task with status,
   so the active task is findable without reading any handover.
2. Every work item names the command that proves it, and that command is re-run
   — not re-read about — by whichever session needs to trust it.
3. One task per session; a task ends with verify green _in that session_,
   committed, `git log -1` confirming the ritual's commit, and the handover
   rewritten — all in one commit.
4. The handover is rewritten (never appended), the log is append-only (never
   amended), and the handover's freshness is checked against git — the commit
   that last touched it must be HEAD — never a hash the file declares about
   itself.
5. Discovered facts go in the notes with a "must not rediscover" framing, as
   soon as discovered. A session's claim that it _finished_ something is not
   this kind of fact — it's re-verified every time, on principle.
6. Research precedes code; reference outputs are captured during research.

The adjustable parts:

- **Threshold.** How much work justifies the apparatus. Keep it high enough that
  small tasks never pay the overhead.
- **Baselines and parity.** The session-start baseline and the stage-close
  parity check should be the cheapest commands that catch real regressions, and
  the parity check should match what the project runs.
- **Granularity of notes.** One `notes.md` until a topic grows, then one file
  per topic.
- **Checklist numbering.** `step_<n>_progress.md` tracks by work-item number; a
  task that continues an earlier item reuses its checklist when it closes that
  stage.
- **Tagging.** Whether completed task-card commits get a `plans/<task>/T<n>` git
  tag is a convenience, not a requirement — useful once a plan folder has enough
  stages that walking commit messages gets tedious.
- **Step 0 depth.** How much existence-checking each session and card does up
  front. Never zero.
- **The opening prompt.** Write it into `plans/README.md` — the exact message a
  new session starts with.

What the technique is not: it is not documentation for humans to read later, and
it is not a system that trusts any session — including the one that just
finished — to accurately report on itself. It is an operating system for
resuming, built on the assumption that the only durable, trusted facts are the
ones git can show: a commit that exists, a provenance check that passes, a
command that was just re-run and seen to pass. If a file would not be read by
the next session, it does not belong in the folder. If a claim would not survive
`git log -1` and a fresh run of the verify command, it does not belong in a
handover.
