package domain

import "testing"

func TestProduct_IsArchived(t *testing.T) {
	cases := []struct {
		name   string
		status ProductStatus
		want   bool
	}{
		{"active is not archived", ProductStatusActive, false},
		{"archived", ProductStatusArchived, true},
		{"zero value is not archived", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := (Product{Status: tc.status}).IsArchived(); got != tc.want {
				t.Fatalf("IsArchived() = %v, want %v", got, tc.want)
			}
		})
	}
}
