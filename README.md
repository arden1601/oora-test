# Abra staff portal

**Abra Skin Clinic** — *Your natural shimmer.*

A mini staff portal for Abra, an invented Jakarta skin clinic: who can sign in, and what each
person can see and do. Built for the OORA backend engineer test; written answers are in
`docs/answers.md`. Go + PostgreSQL, with an optional React admin screen.

> **Status: scaffold only.** The service starts and answers `GET /healthz`. Sign-in, pages,
> grants and audit are not built yet. Sections marked _to be written_ are filled in as each part
> of the test is done.

## Run it

Needs Docker.

```
docker compose up
```

Backend: http://localhost:8080 (`GET /healthz` returns `ok`).

If port 8080 or 5433 is already in use on your machine, copy `.env.example` to `.env` and change
`BACKEND_HOST_PORT` / `DB_HOST_PORT`.

## Test it

```
make test
```

## Layout

| Path | What |
|---|---|
| `backend/` | Go service, standard library `net/http`, PostgreSQL |
| `frontend/` | React + Vite + TypeScript, optional admin screen |
| `docs/answers.md` | the written answers (warm-up, Question 1, Question 3) |
| `docs/test-brief/` | the test as received |
| `docs/pdp-checklist.md` | the UU PDP gate every change passes |
| `AGENTS.md` | the instruction file the AI coding tools worked from |

## Access rules and why

_To be written (Question 2, item 3)._

## Sign in as

_To be written: the seeded fake staff and the demo password._

## Assumptions

_To be written._

## What I would build next

_To be written._

## Optional frontend

_To be written: built, or described._

## How I would run this for real (plan only)

### Where the server lives
_To be written._

### Customer data
_To be written._

### Company data
_To be written._

### Running it
_To be written._

### Getting there
_To be written._
