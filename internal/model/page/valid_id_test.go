package page_test

import (
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestValidID(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"0123456789abcdef": true,
		"0123456789abcde":  false,
		"invalid/id/value": false,
	}
	for id, want := range tests {
		if got := modelpage.ValidID(id); got != want {
			t.Errorf("ValidID(%q) = %v, want %v", id, got, want)
		}
	}
}
