package expense_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/expense"
	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

func TestParseAddCommand_Account(t *testing.T) {
	e, err := expense.ParseAddCommand("ojek 8000 Transportation @gopay 17/10/2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Account != model.AccountGoPay || e.Category != model.CategoryTransportation || e.Nominal != 8000 || e.Date != "17/10/2026" {
		t.Errorf("got %+v", e)
	}

	e, err = expense.ParseAddCommand("nasi 15000 Eating Out")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Account != model.AccountJagoMain {
		t.Errorf("default account = %q, want Jago Main", e.Account)
	}

	if _, err := expense.ParseAddCommand("nasi 15000 Eating Out @dana"); err == nil || !strings.Contains(err.Error(), "unknown account") {
		t.Errorf("want unknown account error, got %v", err)
	}
}

func TestFormatConfirmations_ShowsAccount(t *testing.T) {
	one := []*model.Expense{{Date: "14/05/2026", Category: model.CategoryEatingOut, Description: "lunch", Nominal: 35000, Account: model.AccountGoPay}}
	many := []*model.Expense{
		{Date: "14/05/2026", Category: model.CategoryGroceries, Description: "milk", Nominal: 25000, Account: model.AccountBRI},
		{Date: "14/05/2026", Category: model.CategoryShopping, Description: "soap", Nominal: 20000, Account: model.AccountBRI},
	}
	if out := expense.FormatConfirmations(one); !strings.Contains(out, "Account: GoPay") {
		t.Errorf("single: %s", out)
	}
	if out := expense.FormatConfirmations(many); !strings.Contains(out, "Account: BRI") {
		t.Errorf("multi: %s", out)
	}
}

func TestPocketActions(t *testing.T) {
	es := []*model.Expense{
		// Daily living paid from a tracked account: goes to that account's tracker cell.
		{Account: model.AccountGoPay, Category: model.CategoryTransportation, Nominal: 8000},
		{Account: model.AccountGoPay, Category: model.CategoryGroceries, Nominal: 21200},
		{Account: model.AccountCash, Category: model.CategoryEatingOut, Nominal: 10000},
		// Daily living from Jago Main: the tracker derives it, nothing to do.
		{Account: model.AccountJagoMain, Category: model.CategoryEatingOut, Nominal: 16000},
		// A bill paid from GoPay: not tracked, Bills must pay GoPay back.
		{Account: model.AccountGoPay, Category: model.CategoryRecurringBills, Description: "Gojek PLUS", Nominal: 7000},
		// A date paid from Jago Main: Leisure must pay Jago Main back.
		{Account: model.AccountJagoMain, Category: model.CategoryDate, Description: "Kali Lereng", Nominal: 62000},
		// Paid from its own pocket: nothing to do.
		{Account: model.AccountEducation, Category: model.CategoryEducation, Nominal: 8000},
		// Daily living paid from a dedicated pocket: Jago Main must pay it back.
		{Account: model.AccountLeisure, Category: model.CategoryEatingOut, Description: "lunch", Nominal: 15000},
		// Investment has no pocket.
		{Account: model.AccountGoPay, Category: model.CategoryInvestment, Nominal: 50000},
	}

	adds, reminders := expense.PocketActions(es)

	wantAdds := map[model.Account]int64{model.AccountGoPay: 29200, model.AccountCash: 10000}
	if !reflect.DeepEqual(adds, wantAdds) {
		t.Errorf("adds = %v, want %v", adds, wantAdds)
	}
	want := []string{
		"move Rp 7.000 from Bills to GoPay (Gojek PLUS)",
		"move Rp 62.000 from Leisure to Jago Main (Kali Lereng)",
		"move Rp 15.000 from Jago Main to Leisure (lunch)",
	}
	if len(reminders) != len(want) {
		t.Fatalf("reminders = %q, want %q", reminders, want)
	}
	for i, w := range want {
		if !strings.Contains(reminders[i], w) {
			t.Errorf("reminder %d = %q, want it to contain %q", i, reminders[i], w)
		}
	}
}
