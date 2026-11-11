# AGENTS.md — rules for AI coding tools (and the next engineer)

This is the file the AI tools on this project work from. `CLAUDE.md` only imports it.
Read it fully before changing anything.

## What this project is

A mini **staff portal** for **Abra Skin Clinic**, an invented Jakarta skin clinic: who can sign in, and what each person
can see and do. It is a take-home test (see `docs/test-brief/`), so it must stay **small, readable
and tested**. The author has to explain every line live, without AI, and change one part on the
spot. If a piece of code cannot be explained in a sentence, it is too clever for this repo.

The system guards personal and health data, so Indonesia's Personal Data Protection law (UU PDP,
Law 27/2022) applies to every change. Fake data only.

## Layout

```
backend/                 Go service (standard library net/http) + PostgreSQL
  cmd/server/            main: wiring only, no business logic
  internal/access/       access rules as data + the ONE access decision function
  internal/auth/         sign-in, password hashes, server-side sessions
  internal/grants/       named-person exceptions (give / take away, never delete)
  internal/audit/        log of refused access
  internal/pages/        the five pages with fake data
  internal/store/        database access, branch scoping
  internal/assistant/    AI assistant (DeepSeek); acts only as the signed-in person
  internal/httpx/        middleware and HTTP helpers
  migrations/            SQL migrations and seed
frontend/                React + Vite + TypeScript (optional admin screen)
docs/                    test brief, PDP checklist, written answers
```

## Run and test

```
cp .env.example .env     # optional; defaults work for local dev
docker compose up        # the one command: PostgreSQL + backend on :8080
make test                # all backend tests
make fmt lint            # gofmt + go vet
cd frontend && pnpm install && pnpm dev    # admin screen on :5173, proxies /api to :8080
```

## Rules the system must always keep

These are invariants. Each has a test; no change may break one. `M` rules come from the test's
"rules management has decided".

- **M1** Management and Growth see everything, money included.
- **M2** Doctors never see revenue or any money figure, whatever access they are given. This is a hard deny checked *before* grants.
- **M3** Regular staff never see sales revenue. Only Management, Growth, Finance and specifically named people (by grant) may.
- **M4** The Front Desk enters sales but never sees totals. `/sales-entry` never returns totals or other sales.
- **M5** Every exception is a grant to one named person, recorded with who gave it, when and why, and can be taken away at once. Grants are never deleted.
- **T1** Access is checked on the server on **every request**; refused → `403` and one audit row (who, which page, when).
- **T2** Access rules are **data**. Departments are compared in exactly one place: the decision function in `internal/access`. No `if department == ...` anywhere else.
- **T3** Every staff member belongs to a branch. Branch staff see only their own branch; head office sees all. Queries take the branch scope as a required argument.
- **T4** A revoked grant or deactivated person is refused on the very next request. Never cache permissions, sessions or staff status between requests.
- **T5** Passwords are stored only as hashes.

## What an AI must never do here

1. **Never weaken, skip or delete a test to make it pass.** Fix the code, or stop and ask.
2. **Never put personal data in URLs, logs, audit rows, error messages or source code.** No names, phones, emails, medical details, passwords or session tokens. IDs and reason codes only.
3. **Never use real data.** Seed and fixtures are invented; emails use `example.com`.
4. **Never commit secrets.** Keys live in `.env` (git-ignored). `.env.example` has placeholders only.
5. **Never decide access in the frontend**, in a handler, or in an AI tool. Hiding a button is not a control. Call the decision function.
6. **Never give the AI assistant its own access.** Its tools run through the same access decision and branch scope as the signed-in person. Never send personal or health data to the model provider (it is hosted outside Indonesia).
7. **Never add a dependency without a reason** written in the commit message. Standard library first.
8. **Never delete or rewrite grant or audit rows.** They are append-only history.
9. **Never commit, push, or change the database schema outside a migration** unless asked.
10. **Never invent a rule.** If the test brief or these rules do not cover a case, write the assumption down in the README and ask.

## How to work

1. **Design first.** State which rules (M1–T5) the change touches before writing code.
2. **Test first for rules.** Write the failing test that names the rule (`TestDoctorNeverSeesRevenueEvenWithGrant`), then the code.
3. **Small changes.** One module at a time. Prefer a plain loop over an abstraction. Handle every error; no `panic` in request paths.
4. **Run the PDP gate.** Go through `docs/pdp-checklist.md` before calling a change done.
5. **Verify.** `make fmt lint test` must pass. Say plainly what was run and what was not.
6. **Report mistakes.** If you got something wrong and it was caught, say so, so it can be logged.

## Go conventions

- Standard library `net/http` routing (`mux.HandleFunc("GET /bookings", ...)`).
- `context.Context` first on anything that touches the database.
- SQL is written by hand with parameters (`$1`), never built by string concatenation.
- Errors are wrapped with context (`fmt.Errorf("load staff: %w", err)`) and never shown raw to the client.
- Money is an integer number of rupiah.
- Logging uses `log/slog` with IDs and codes only.

## Documentation

Working documentation (module notes, decisions, the full UU PDP summary and checklist, the log of
AI mistakes) is kept in the author's Obsidian vault at
`~/Obsidian/obsidian/Jarvis/oora-test/` — start at `Project OORA Test.md`. When a rule or
behaviour changes, update the module note there, this file, and `README.md` together.
