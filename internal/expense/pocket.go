package expense

import (
	"fmt"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/pkg/utils"
)

// trackedAccounts are the daily-living accounts whose "Used" cell in the pocket
// tracker is typed in; Jago Main's is derived from them by the sheet.
var trackedAccounts = map[model.Account]bool{
	model.AccountGoPay: true,
	model.AccountBRI:   true,
	model.AccountCash:  true,
}

// PocketActions decides what saving es means for the pocket tracker: the amount to
// add to each tracked account's "Used" cell, and the transfers needed when an
// expense was paid from an account other than the pocket whose budget covers it.
// Only daily living goes into a tracker cell; anything else (a bill paid with
// GoPay, a date paid from Jago Main) becomes a "move money back" reminder instead.
func PocketActions(es []*model.Expense) (map[model.Account]int64, []string) {
	adds := map[model.Account]int64{}
	var reminders []string
	for _, e := range es {
		paidFrom := e.Account
		if paidFrom == "" {
			paidFrom = model.AccountJagoMain
		}
		pocket := e.Category.Pocket()
		switch {
		case pocket == "" || pocket == paidFrom:
		case pocket == model.AccountJagoMain && trackedAccounts[paidFrom]:
			adds[paidFrom] += e.Nominal
		default:
			reminders = append(reminders, fmt.Sprintf("↪ move Rp %s from %s to %s (%s)",
				utils.FormatNominal(e.Nominal), pocket, paidFrom, e.Description))
		}
	}
	return adds, reminders
}
