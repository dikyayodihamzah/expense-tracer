package model

import "strings"

type Category string

const (
	CategoryFood                 Category = "Food"
	CategoryPersonalCare         Category = "Personal Care"
	CategoryTransportation       Category = "Transportation"
	CategoryShopping             Category = "Shopping"
	CategoryEntertainmentLeisure Category = "Entertainment and Leisure"
	CategoryDonation             Category = "Donation"
	CategoryOrthodental          Category = "Orthodental"
	CategoryInvestment           Category = "Investment"
	CategoryOthers               Category = "Others"
)

var allCategories = []Category{
	CategoryFood,
	CategoryPersonalCare,
	CategoryTransportation,
	CategoryShopping,
	CategoryEntertainmentLeisure,
	CategoryDonation,
	CategoryOrthodental,
	CategoryInvestment,
	CategoryOthers,
}

type Expense struct {
	Date        string   // DD/MM/YYYY
	Month       string   // January, February, ...
	Category    Category
	Description string
	Nominal     int64
}

// MatchCategory matches a string to a known Category, case-insensitive.
// Returns CategoryOthers if no match found.
func MatchCategory(s string) Category {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, c := range allCategories {
		if strings.ToLower(string(c)) == lower {
			return c
		}
	}
	return CategoryOthers
}

// AllCategories returns a copy of the fixed category list.
func AllCategories() []Category {
	result := make([]Category, len(allCategories))
	copy(result, allCategories)
	return result
}
