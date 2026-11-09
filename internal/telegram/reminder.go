package telegram

import (
	"context"
	"log"
	"time"
)

// wib is Indonesia Western Time. A fixed zone needs no tzdata in the container,
// and WIB has no daylight saving.
var wib = time.FixedZone("WIB", 7*60*60)

const monthEndChecklist = `📝 *Month-end checklist*

1. Budgeting: fill the Sisa balances (BRI, Jago, Cash) and Utang
2. Move a positive Sisa to investment, or repay Backup
3. Pocket tracker: reset BRI, Cash and GoPay Used (J4, J5, J8) to 0
4. Pocket tracker: set next month's Opening Balances

Use /pockets to see the current balances.`

// nextMonthEnd returns the next reminder time: 20:00 WIB on the last day of the month.
func nextMonthEnd(now time.Time) time.Time {
	now = now.In(wib)
	for m := now.Month(); ; m++ {
		at := time.Date(now.Year(), m+1, 0, 20, 0, 0, 0, wib) // day 0 of next month = last day of m
		if at.After(now) {
			return at
		}
	}
}

// runMonthEndReminder sends the month-end checklist to chatID until ctx is cancelled.
func (tc *Controller) runMonthEndReminder(ctx context.Context, chatID int64) {
	for {
		at := nextMonthEnd(time.Now())
		log.Printf("[reminder] next month-end reminder at %s", at.Format(time.RFC3339))
		timer := time.NewTimer(time.Until(at))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			tc.replyMarkdown(chatID, monthEndChecklist)
		}
	}
}
