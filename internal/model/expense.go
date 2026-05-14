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
