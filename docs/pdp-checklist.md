# UU PDP checklist

Indonesia's Personal Data Protection law (Law No. 27 of 2022) applies to this system: customer
names and phones are personal data, and health details, photos and a customer's payments are
*specific* personal data, which needs stronger protection. Run this list before any change is
called done. A "no" or "not sure" on a **must** item blocks the change.

## What data is touched
- [ ] Named the data this change reads, writes or exposes, and whether any is specific data (health, photos, personal financial).
- [ ] The feature uses no more data than it needs.

## Who can reach it (must)
- [ ] Access is decided on the server, on every request, by the one access decision function.
- [ ] Default is deny; access comes only from the rules data or a recorded grant.
- [ ] Rules M1–M5 in `AGENTS.md` still hold, each with a passing test.
- [ ] Branch staff still see only their own branch.
- [ ] A revoked grant or deactivated person is refused on the very next request.

## Where it can leak (must)
- [ ] No personal data in URLs (paths or query strings).
- [ ] No personal data, passwords, session tokens or secrets in logs, audit rows or error messages.
- [ ] No real personal data in source, seed data, tests or fixtures.
- [ ] No secrets in git.
- [ ] Responses carry only the fields the role needs (no money fields for roles that may not see money).

## Leaving the system
- [ ] Any third party receiving data (AI model, messaging vendor, monitoring) is named.
- [ ] Nothing personal leaves Indonesia unless it is de-identified first or the transfer has a documented basis.
- [ ] The AI assistant only gets what the signed-in person may see, through the same access decision.
- [ ] Exports of customer data are approved and logged.

## Accountability
- [ ] Refused access is audited (who, which page, when). Grants record who, when, why, and are never deleted.
- [ ] New stored data has a retention and deletion story.
- [ ] `AGENTS.md` and `README.md` updated if a rule or behaviour changed.
