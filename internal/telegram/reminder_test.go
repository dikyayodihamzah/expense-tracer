package telegram

import (
	"testing"
	"time"
)

func TestNextMonthEnd(t *testing.T) {
	tests := []struct {
		now, want time.Time
	}{
		// Before the reminder on the last day: fires the same evening.
		{time.Date(2026, 11, 9, 8, 0, 0, 0, wib), time.Date(2026, 11, 30, 20, 0, 0, 0, wib)},
		{time.Date(2026, 11, 30, 19, 59, 0, 0, wib), time.Date(2026, 11, 30, 20, 0, 0, 0, wib)},
		// At or after it: next month's last day.
		{time.Date(2026, 11, 30, 20, 0, 0, 0, wib), time.Date(2026, 12, 31, 20, 0, 0, 0, wib)},
		{time.Date(2026, 12, 31, 21, 0, 0, 0, wib), time.Date(2027, 1, 31, 20, 0, 0, 0, wib)},
		{time.Date(2027, 2, 1, 0, 0, 0, 0, wib), time.Date(2027, 2, 28, 20, 0, 0, 0, wib)},
	}
	for _, tt := range tests {
		if got := nextMonthEnd(tt.now); !got.Equal(tt.want) {
			t.Errorf("nextMonthEnd(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}
}
