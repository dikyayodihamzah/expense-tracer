package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// ParseAddCommand parses "/add <description> <nominal> <category>" args.
// args is the text after "/add ".
func ParseAddCommand(args string) (*model.Expense, error) {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		return nil, fmt.Errorf("usage: /add <description> <nominal> <category>")
	}

	// Last part is category, second-to-last is nominal, rest is description.
	categoryStr := parts[len(parts)-1]
	nominalStr := parts[len(parts)-2]
	description := strings.Join(parts[:len(parts)-2], " ")

	nominal, err := strconv.ParseInt(nominalStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("nominal must be a number, got %q", nominalStr)
	}

	return &model.Expense{
		Description: description,
		Nominal:     nominal,
		Category:    model.MatchCategory(categoryStr),
	}, nil
}

// FillExpenseMeta sets Date (if empty) and Month from the given time.
func FillExpenseMeta(e *model.Expense, now time.Time) error {
	if e.Date == "" {
		e.Date = now.Format("02/01/2006")
	}
	month, err := MonthFromDate(e.Date)
	if err != nil {
		return err
	}
	e.Month = month
	return nil
}

// MonthFromDate returns the full month name from a DD/MM/YYYY date string.
func MonthFromDate(date string) (string, error) {
	t, err := time.Parse("02/01/2006", date)
	if err != nil {
		return "", fmt.Errorf("invalid date %q (expected DD/MM/YYYY): %w", date, err)
	}
	return t.Format("January"), nil
}

// FormatConfirmation formats a parsed expense for user confirmation.
func FormatConfirmation(e *model.Expense) string {
	return fmt.Sprintf(
		"📋 *Expense Details*\n\n📅 Date: %s\n🗂 Category: %s\n📝 Description: %s\n💰 Nominal: Rp %s\n\nSave this expense?",
		e.Date,
		string(e.Category),
		e.Description,
		formatNominal(e.Nominal),
	)
}

// FormatToday formats today's expenses as a message.
func FormatToday(rows [][]interface{}, today string) string {
	var lines []string
	var total int64

	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		if row[1] != today {
			continue
		}
		desc, _ := row[4].(string)
		nomStr, _ := row[5].(string)
		nom, _ := strconv.ParseInt(nomStr, 10, 64)
		total += nom
		lines = append(lines, fmt.Sprintf("• %s — Rp %s", desc, formatNominal(nom)))
	}

	if len(lines) == 0 {
		return "No expenses recorded today."
	}
	return fmt.Sprintf("*Today (%s)*\n\n%s\n\n*Total: Rp %s*", today, strings.Join(lines, "\n"), formatNominal(total))
}

// FormatSummary formats a monthly category breakdown.
func FormatSummary(rows [][]interface{}, month string) string {
	totals := make(map[string]int64)
	var grandTotal int64

	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		if row[2] != month {
			continue
		}
		cat, _ := row[3].(string)
		nomStr, _ := row[5].(string)
		nom, _ := strconv.ParseInt(nomStr, 10, 64)
		totals[cat] += nom
		grandTotal += nom
	}

	if len(totals) == 0 {
		return fmt.Sprintf("No expenses found for %s.", month)
	}

	var lines []string
	for _, c := range model.AllCategories() {
		if v, ok := totals[string(c)]; ok {
			lines = append(lines, fmt.Sprintf("• %s: Rp %s", c, formatNominal(v)))
		}
	}
	return fmt.Sprintf("*%s Summary*\n\n%s\n\n*Total: Rp %s*", month, strings.Join(lines, "\n"), formatNominal(grandTotal))
}

// FormatBudget formats the monthly budget progress.
func FormatBudget(rows [][]interface{}) string {
	if len(rows) < 2 {
		return "No budget data available."
	}

	var lastDay, dailyExp, cumExp, cumBudget string
	for _, row := range rows[1:] { // skip header
		if len(row) < 4 {
			continue
		}
		d, _ := row[0].(string)
		if d != "" {
			lastDay = d
			dailyExp, _ = row[1].(string)
			cumExp, _ = row[2].(string)
			cumBudget, _ = row[3].(string)
		}
	}

	return fmt.Sprintf(
		"*Budget Progress (Day %s)*\n\n📆 Today's Expense: %s\n📊 Cumulative Expense: %s\n🎯 Cumulative Budget: %s",
		lastDay, dailyExp, cumExp, cumBudget,
	)
}

func formatNominal(n int64) string {
	if n < 0 {
		return "-" + formatNominal(-n)
	}
	s := strconv.FormatInt(n, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
