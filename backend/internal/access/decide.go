package access

// Money is a department's money setting. It is data, stored on the department row.
type Money string

const (
	MoneyYes   Money = "yes"   // may open pages that show money
	MoneyNo    Money = "no"    // may not, unless one named person is given a money grant
	MoneyNever Money = "never" // may not, and no grant can change it (doctors, Front Desk)
)

// Action is what the person wants to do on a page.
type Action string

const (
	ActionRead  Action = "read"  // open the page
	ActionEnter Action = "enter" // submit to the page, never read back
)

// Reason codes for a refusal. They go into the audit log, so they hold no personal data.
const (
	ReasonInactive       = "inactive"
	ReasonMoneyForbidden = "money_forbidden"
	ReasonNoMoneyAccess  = "no_money_access"
	ReasonNoPageAccess   = "no_page_access"
)

// Facts is everything the decision needs about one request. The caller loads it
// fresh from the database every time, so nothing here is ever cached.
type Facts struct {
	Active            bool  // the person is an active member of staff
	Money             Money // the money setting of the person's department
	ShowsMoney        bool  // the page shows a money figure
	DepartmentHasPage bool  // the permission table gives the department this page and action
	PageGrant         bool  // the person holds an active grant for this page and action
	MoneyGrant        bool  // the person holds an active "may see money" grant
}

// Decision is the answer: allowed, or refused with a reason code.
type Decision struct {
	Allowed bool
	Reason  string
}

// Decide is the one place where access is decided. Three checks must all pass:
// the person is active, the money rule allows the page, and the person has the page.
func Decide(f Facts) Decision {
	// 1. A person who has left is refused, whatever else is true.
	if !f.Active {
		return Decision{Reason: ReasonInactive}
	}

	// 2. Money. "never" is tested before any grant is looked at, so no grant overrides it.
	if f.ShowsMoney {
		if f.Money == MoneyNever {
			return Decision{Reason: ReasonMoneyForbidden}
		}
		if f.Money != MoneyYes && !f.MoneyGrant {
			return Decision{Reason: ReasonNoMoneyAccess}
		}
	}

	// 3. The page itself: from the permission table, or from a grant to this person.
	if !f.DepartmentHasPage && !f.PageGrant {
		return Decision{Reason: ReasonNoPageAccess}
	}

	return Decision{Allowed: true}
}
