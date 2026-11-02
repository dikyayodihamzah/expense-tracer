package model

import "strings"

type Category string

const (
	CategoryGroceries      Category = "Groceries"
	CategoryEatingOut      Category = "Eating Out"
	CategoryTransportation Category = "Transportation"
	CategoryShopping       Category = "Shopping"
	CategoryPersonalCare   Category = "Personal Care"
	CategoryDate           Category = "Date"
	CategoryHobbiesLeisure Category = "Hobbies & Leisure"
	CategoryGiftsDonation  Category = "Gifts & Donation"
	CategoryOthers         Category = "Others"
	CategoryHousing        Category = "Housing"
	CategoryRecurringBills Category = "Recurring Bills"
	CategoryEducation      Category = "Education"
	CategoryOrthodental    Category = "Orthodental"
	CategoryInvestment     Category = "Investment"
)

var allCategories = []Category{
	CategoryGroceries,
	CategoryEatingOut,
	CategoryTransportation,
	CategoryShopping,
	CategoryPersonalCare,
	CategoryDate,
	CategoryHobbiesLeisure,
	CategoryGiftsDonation,
	CategoryOthers,
	CategoryHousing,
	CategoryRecurringBills,
	CategoryEducation,
	CategoryOrthodental,
	CategoryInvestment,
}

type Expense struct {
	Date        string // DD/MM/YYYY
	Month       string // January, February, ...
	Category    Category
	Description string
	Nominal     int64
}

// MatchCategory matches a string to a known Category, case-sensitive.
// Returns CategoryOthers if no match found.
func MatchCategory(s string) Category {
	if category, ok := AllCategoriesMap()[strings.ToLower(strings.TrimSpace(s))]; ok {
		return category
	}
	return CategoryOthers
}

// AllCategories returns a copy of the fixed category list.
func AllCategories() []Category {
	result := make([]Category, len(allCategories))
	copy(result, allCategories)
	return result
}

func AllCategoriesMap() map[string]Category {
	result := make(map[string]Category)
	for _, c := range allCategories {
		result[strings.ToLower(string(c))] = c
	}
	return result
}
