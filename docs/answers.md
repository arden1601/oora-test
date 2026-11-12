# Written answers

## Warm-up — Name the company

**Abra Skin Clinic** — *Your natural shimmer.*

Jakarta's heat, humidity and haze leave skin dull and tired, and customers there want skin that
looks healthy on its own rather than covered up; "Abra" is short, easy to say in Indonesian, and
comes from the Sanskrit *abhra*, which names both the clear sky and mica, the mineral that shimmers.

## Question 1 — Design the staff access system

**1. Sign-in.** Work email, password and an authenticator code for everyone. An admin creates
the account; no manager approves a sign-in (single sign-on later). Sessions: 15 idle minutes,
12 hours at most. Each person uses their own account, even on the shared front desk. When
someone leaves, an admin marks them inactive and the next request fails.

**2. Permissions.** Each person has one department and one branch; head office sees all branches.

- **Permission table**: department, page, mode. `read` opens it; `enter` submits, never reads.
- **Money**: a page showing any money figure needs the department's money setting to be `yes`.
  A grant turns `no` into `yes` for one person; nothing changes `never`.
- **Grant**: one named person gets one page or "may see money", recorded with who gave it,
  when and why. Removing it marks the row. Both work from the next request.

| Department | Bookings | Patients | Medical record | Sales entry | Revenue | Money |
|---|---|---|---|---|---|---|
| Management, Growth | R | R | R | E | R | yes |
| Finance | | | | | R | yes |
| CRM & Sales | R | | | | | no |
| Clinic Operations | R | R | | | | no |
| Front Desk | R | R | | E | | never |
| Beauty Consultants | | R | | | | no |
| Doctor | | R | R | | | never |
| Marketing, HR, Partnerships | | | | | | no |

`R` read, `E` enter only; only Revenue shows money. Admins (Management, Growth) manage
accounts and grants on one admin screen: one grant gives a marketer the follow-up page, two
give a clinic lead clinic revenue.

**3. Security settings.** Logged by ID: refusals (who, page, when), sign-ins, grants, exports,
staff changes.

- **Cookie**: holds a random token; the database keeps its hash. `HttpOnly`, `Secure`,
  `SameSite=Lax`.
- **Lockout**: five wrong passwords lock the account for 15 minutes; sign-in is rate-limited.
- **Exports**: only for a page the requester may read, then approved by Management in team chat.
- **Database role**: the app cannot change tables or delete audit or grant rows.
- **AI assistants**: no access of their own; each tool runs the same check as the person.

**4. Architecture.** One Go service on PostgreSQL. Tables: `branches`, `departments`,
`staff`, `pages`, `department_pages`, `grants`, `sessions`, `audit_log`, `jobs`. One function
decides every request: the person must be active, pass the money rule and have the page by
table or grant. Refused: 403 and an audit row. Allowed: it returns the branch scope (own, or
all) for every query. The AI assistant and chat app use the person's session, so the same
function decides. A new branch is one row; a standby database comes before the second branch.

**5. Tools and servers.** One Jakarta VPS runs it all in Docker: one service, one folder
per concern, jobs (reminders, exports) inside it. Database changes are numbered SQL files
that keep old code working, so rollback is a redeploy. The API is one OpenAPI file. People
review every change; AI never deploys.

| Part | Choice | Run by | Cost/month | Why |
|---|---|---|---|---|
| Language | Go, standard library | us | $0 | Plain to read |
| Database | PostgreSQL on a Sumopod VPS | us | ~$10 | One box; data stays local |
| Jobs and queues | a `jobs` table in PostgreSQL | us | $0 | Enough at this size |
| Sign-in and 2FA | built into the app | us | $0 | Small, fully understood |
| Secrets | `.env` file on the VPS, owner-only | us | $0 | Nothing secret in git |
| Deploys | GitHub Actions over Tailscale SSH | us | ~$16 | Tests gate every deploy |
| Logs and alerts | Docker logs, outside uptime check | us | $0–10 | Nothing extra to run |

**The one mistake to avoid:** deciding access in more than one place, where checks drift apart.

## Question 3 — AI: how I use it and how I would manage it

**How I use it.** Claude Code, for design drafts, tests, code and docs. It works from
`AGENTS.md` and my project notes. The order is fixed: design, a failing test that names the
rule, then code.

**What I always check myself.** The AI saying "all pass" is not a check. I run the tests and
lint myself (the end-to-end tests are Go tests on a real PostgreSQL) and read the diff. Then
I test by hand as different people: sign in, open a page, enter a sale, give and take away a
grant. The AI also wrote the tests, so I check they can fail: the code was broken on purpose
in eight ways and a named test caught each. Access, money and personal data go through the
PDP checklist.

**What it got wrong here.** Two mistakes in the Question 1 design, both caught by giving a
second AI reader only the brief and the answer and asking it to get round each rule.

- **Money defined too narrowly.** The draft said "money means sales money", so a doctor
  granted a page with prices would have got through. Fix: a money page is any page that shows
  a money figure.
- **Access decided in two places**, a decision and then a separate branch filter. Fix: the
  one decision returns the branch scope, and every query requires it.

That cold read is now part of every design check, and each mistake is logged the same day.

**Shared instruction and skill files.**

- **Organise**: one `AGENTS.md` at the repo root is the source of truth; tool files such as
  `CLAUDE.md` only import it. Shared skills are committed in `.claude/skills/`. Developers
  pick their own tool and settings, which cannot override the rules.
- **Keep current**: a rule changes in the same pull request as the code it describes; every
  caught AI mistake becomes a rule, a test or a checklist item.
- **Review**: like code. Every change is made on its own branch, never on `main`, and reaches
  `main` by pull request. A change to the rules needs the team's agreement.
- **Never in them**: credentials or keys (they stay in the git-ignored `.env`), personal data
  of customers or staff, or any line that lets the AI skip a test or a review.
