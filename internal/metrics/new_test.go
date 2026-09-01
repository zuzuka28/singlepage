package metrics_test

import (
	"testing"

	"singlepage/internal/metrics"
)

func TestNewReturnsMetrics(t *testing.T) {
	t.Parallel()

	if metrics.New() == nil {
		t.Fatal("New() returned nil")
	}
}
