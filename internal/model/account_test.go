package model_test

import (
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

func TestMatchAccount(t *testing.T) {
	tests := []struct {
		in     string
		want   model.Account
		wantOK bool
	}{
		{"GoPay", model.AccountGoPay, true},
		{"gopay", model.AccountGoPay, true},
		{"Jago Main", model.AccountJagoMain, true},
		{"jagomain", model.AccountJagoMain, true},
		{"jago", model.AccountJagoMain, true},
		{" bri ", model.AccountBRI, true},
		{"leisure", model.AccountLeisure, true},
		{"", model.AccountJagoMain, false},
		{"dana", model.AccountJagoMain, false},
	}
	for _, tt := range tests {
		got, ok := model.MatchAccount(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("MatchAccount(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestCategoryPocket(t *testing.T) {
	tests := map[model.Category]model.Account{
		model.CategoryEatingOut:      model.AccountJagoMain,
		model.CategoryTransportation: model.AccountJagoMain,
		model.CategoryDate:           model.AccountLeisure,
		model.CategoryHousing:        model.AccountBills,
		model.CategoryRecurringBills: model.AccountBills,
		model.CategoryEducation:      model.AccountEducation,
		model.CategoryOrthodental:    model.AccountOrthodental,
		model.CategoryInvestment:     "",
	}
	for c, want := range tests {
		if got := c.Pocket(); got != want {
			t.Errorf("%s.Pocket() = %q, want %q", c, got, want)
		}
	}
}
