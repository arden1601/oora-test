package access

import "testing"

// The facts below describe one request: who is asking, which page, and what the
// permission table and the grants say. Decide must give the same answer every time.

func TestActivePersonWithPageRowIsAllowed(t *testing.T) {
	got := Decide(Facts{Active: true, Money: MoneyNo, DepartmentHasPage: true})
	if !got.Allowed {
		t.Fatalf("got refused (%s), want allowed", got.Reason)
	}
}

func TestInactivePersonIsRefused(t *testing.T) { // T4
	got := Decide(Facts{Active: false, Money: MoneyYes, DepartmentHasPage: true})
	wantRefused(t, got, ReasonInactive)
}

func TestNoRowAndNoGrantIsRefused(t *testing.T) { // default is deny
	got := Decide(Facts{Active: true, Money: MoneyYes})
	wantRefused(t, got, ReasonNoPageAccess)
}

func TestPageGrantOpensAPageTheDepartmentDoesNotHave(t *testing.T) { // M5
	got := Decide(Facts{Active: true, Money: MoneyNo, PageGrant: true})
	if !got.Allowed {
		t.Fatalf("got refused (%s), want allowed", got.Reason)
	}
}

func TestManagementSeesMoneyPage(t *testing.T) { // M1
	got := Decide(Facts{Active: true, Money: MoneyYes, ShowsMoney: true, DepartmentHasPage: true})
	if !got.Allowed {
		t.Fatalf("got refused (%s), want allowed", got.Reason)
	}
}

func TestDoctorNeverSeesRevenueEvenWithGrants(t *testing.T) { // M2
	got := Decide(Facts{
		Active: true, Money: MoneyNever, ShowsMoney: true,
		PageGrant: true, MoneyGrant: true,
	})
	wantRefused(t, got, ReasonMoneyForbidden)
}

func TestRegularStaffNeverSeeRevenueWithoutMoneyGrant(t *testing.T) { // M3
	// The person was given the Revenue page but not "may see money".
	got := Decide(Facts{Active: true, Money: MoneyNo, ShowsMoney: true, PageGrant: true})
	wantRefused(t, got, ReasonNoMoneyAccess)
}

func TestMoneyGrantAloneDoesNotOpenThePage(t *testing.T) { // M3
	got := Decide(Facts{Active: true, Money: MoneyNo, ShowsMoney: true, MoneyGrant: true})
	wantRefused(t, got, ReasonNoPageAccess)
}

func TestNamedPersonWithBothGrantsSeesRevenue(t *testing.T) { // M3, M5
	got := Decide(Facts{
		Active: true, Money: MoneyNo, ShowsMoney: true,
		PageGrant: true, MoneyGrant: true,
	})
	if !got.Allowed {
		t.Fatalf("got refused (%s), want allowed", got.Reason)
	}
}

func TestMoneySettingDoesNotMatterOnAPageWithoutMoney(t *testing.T) {
	// A doctor opens the medical record: money is "never", the page shows none.
	got := Decide(Facts{Active: true, Money: MoneyNever, DepartmentHasPage: true})
	if !got.Allowed {
		t.Fatalf("got refused (%s), want allowed", got.Reason)
	}
}

func TestUnknownMoneySettingIsRefusedOnAMoneyPage(t *testing.T) {
	// A typo in the data must not open a money page.
	got := Decide(Facts{Active: true, Money: Money("maybe"), ShowsMoney: true, DepartmentHasPage: true})
	wantRefused(t, got, ReasonNoMoneyAccess)
}

func wantRefused(t *testing.T, got Decision, reason string) {
	t.Helper()
	if got.Allowed {
		t.Fatalf("got allowed, want refused with %q", reason)
	}
	if got.Reason != reason {
		t.Fatalf("got reason %q, want %q", got.Reason, reason)
	}
}
