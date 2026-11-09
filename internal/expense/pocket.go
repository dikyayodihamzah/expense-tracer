package expense

import (
	"fmt"
	"math"
	"strings"

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

// FormatPockets formats the pocket tracker table read with unformatted values.
// Each row is [group, pocket, opening, top-up, used, should remain]: a row with
// only a group name starts a section, and the "Total" row ends the table.
func FormatPockets(rows [][]any) string {
	var lines []string
	for _, row := range rows {
		group, _ := cell(row, 0).(string)
		pocket, _ := cell(row, 1).(string)
		if group == "Total" {
			lines = append(lines, fmt.Sprintf("\n*Total left: Rp %s*", utils.FormatNominal(amount(cell(row, 5)))))
			break
		}
		if pocket == "" {
			if group != "" {
				lines = append(lines, fmt.Sprintf("\n*%s*", group))
			}
			continue
		}
		available := amount(cell(row, 2)) + amount(cell(row, 3))
		used, left := amount(cell(row, 4)), amount(cell(row, 5))
		flag := "•"
		if left < 0 {
			flag = "⚠️"
		}
		pct := ""
		if available > 0 {
			pct = fmt.Sprintf(", %d%%", used*100/available)
		}
		lines = append(lines, fmt.Sprintf("%s %s: Rp %s left (used Rp %s of Rp %s%s)",
			flag, pocket, utils.FormatNominal(left), utils.FormatNominal(used), utils.FormatNominal(available), pct))
	}
	if len(lines) == 0 {
		return "No pocket data found."
	}
	return "*Pockets*\n" + strings.Join(lines, "\n")
}

func cell(row []any, i int) any {
	if i < len(row) {
		return row[i]
	}
	return nil
}

// amount reads an unformatted numeric cell; blanks count as 0.
func amount(v any) int64 {
	f, _ := v.(float64)
	return int64(math.Round(f))
}
