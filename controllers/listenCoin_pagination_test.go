package controllers

import "testing"

func TestFundingRatePagination(t *testing.T) {
	for _, tc := range []struct {
		name, rawPage, rawLimit string
		total                   int64
		page, limit, offset     int
	}{
		{"default", "", "", 120, 1, 50, 0},
		{"second page", "2", "50", 120, 2, 50, 50},
		{"last page", "3", "50", 120, 3, 50, 100},
		{"clamp out of bounds", "999", "50", 120, 3, 50, 100},
		{"single page", "2", "50", 3, 1, 50, 0},
		{"empty", "3", "50", 0, 1, 50, 0},
		{"bad inputs", "-1", "invalid", 100, 1, 50, 0},
		{"limit cap", "1", "9999", 1000, 1, 500, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, limit, offset := fundingRatePagination(tc.rawPage, tc.rawLimit, tc.total)
			if page != tc.page || limit != tc.limit || offset != tc.offset {
				t.Fatalf("got (%d,%d,%d), want (%d,%d,%d)", page, limit, offset, tc.page, tc.limit, tc.offset)
			}
		})
	}
}
