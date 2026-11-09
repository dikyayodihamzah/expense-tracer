package expense

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/pkg/utils"
)

// FormatLeft says how much of the month's living budget is left and what that
// allows per day. rows are the Monthly Expense daily rows read with unformatted
// values: [day, daily expense, cumulative expense, cumulative budget].
func FormatLeft(rows [][]any, now time.Time) string {
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	var spent, planned, budget int64
	var found bool
	for _, row := range rows {
		switch int(amount(cell(row, 0))) {
		case now.Day():
			spent, planned, found = amount(cell(row, 2)), amount(cell(row, 3)), true
		case daysInMonth:
			budget = amount(cell(row, 3))
		}
	}
	if !found || budget == 0 {
		return "No budget data found for today."
	}

	left := budget - spent
	daysLeft := int64(daysInMonth - now.Day() + 1)
	pace := fmt.Sprintf("✅ Rp %s under plan so far", utils.FormatNominal(planned-spent))
	if spent > planned {
		pace = fmt.Sprintf("⚠️ Rp %s over plan so far", utils.FormatNominal(spent-planned))
	}
	return fmt.Sprintf("*Left to spend (Day %d)*\n\n💰 Rp %s left of Rp %s\n📆 Rp %s per day for the next %d days (incl. today)\n%s",
		now.Day(), utils.FormatNominal(left), utils.FormatNominal(budget),
		utils.FormatNominal(left/daysLeft), daysLeft, pace)
}

// FormatRecent lists the last n transaction rows, newest first.
func FormatRecent(rows [][]any, n int) string {
	if len(rows) > n {
		rows = rows[len(rows)-n:]
	}
	var lines []string
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		date, _ := cell(row, 1).(string)
		category, _ := cell(row, 3).(string)
		desc, _ := cell(row, 4).(string)
		nomStr, _ := cell(row, 5).(string)
		nom, _ := strconv.ParseInt(nomStr, 10, 64)
		if len(date) >= 5 {
			date = date[:5] // DD/MM
		}
		lines = append(lines, fmt.Sprintf("• %s · %s · %s · Rp %s", date, category, desc, utils.FormatNominal(nom)))
	}
	if len(lines) == 0 {
		return "No expenses recorded yet."
	}
	return fmt.Sprintf("*Last %d expenses*\n\n%s", len(lines), strings.Join(lines, "\n"))
}
