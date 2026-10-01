package paging

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name               string
		page, size         int32
		wantLimit, wantOff int
	}{
		{"defaults", 0, 0, DefaultPageSize, 0},
		{"second page", 2, 10, 10, 10},
		{"negative page", -3, 5, 5, 0},
		{"size capped", 1, 1000, MaxPageSize, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, off := Normalize(tt.page, tt.size)
			if limit != tt.wantLimit || off != tt.wantOff {
				t.Fatalf("Normalize(%d, %d) = (%d, %d), want (%d, %d)",
					tt.page, tt.size, limit, off, tt.wantLimit, tt.wantOff)
			}
		})
	}
}
