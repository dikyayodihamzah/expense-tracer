package model

import "strings"

// Account is where an expense was paid from: a bank account, e-wallet, cash,
// or one of the dedicated Jago pockets.
type Account string

const (
	AccountJagoMain    Account = "Jago Main"
	AccountGoPay       Account = "GoPay"
	AccountBRI         Account = "BRI"
	AccountCash        Account = "Cash"
	AccountLeisure     Account = "Leisure"
	AccountBills       Account = "Bills"
	AccountEducation   Account = "Education"
	AccountOrthodental Account = "Orthodental"
)

var allAccounts = []Account{
	AccountJagoMain,
	AccountGoPay,
	AccountBRI,
	AccountCash,
	AccountLeisure,
	AccountBills,
	AccountEducation,
	AccountOrthodental,
}

// AllAccounts returns a copy of the fixed account list.
func AllAccounts() []Account {
	result := make([]Account, len(allAccounts))
	copy(result, allAccounts)
	return result
}

// MatchAccount matches s to an Account ignoring case and spaces ("jago" means
// Jago Main). It returns Jago Main and false when s matches nothing.
func MatchAccount(s string) (Account, bool) {
	key := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	if key == "jago" {
		return AccountJagoMain, true
	}
	for _, a := range allAccounts {
		if key == strings.ToLower(strings.ReplaceAll(string(a), " ", "")) {
			return a, true
		}
	}
	return AccountJagoMain, false
}

// Pocket returns the account whose budget pays for the category: Jago Main for
// daily living, a dedicated pocket for the rest, or "" when none (Investment).
func (c Category) Pocket() Account {
	switch c {
	case CategoryDate:
		return AccountLeisure
	case CategoryHousing, CategoryRecurringBills:
		return AccountBills
	case CategoryEducation:
		return AccountEducation
	case CategoryOrthodental:
		return AccountOrthodental
	case CategoryInvestment:
		return ""
	}
	return AccountJagoMain
}
