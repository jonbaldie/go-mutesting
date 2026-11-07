package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/coverage"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
)

func TestPerTestFallbackLeavesSuiteUnfiltered(t *testing.T) {
	profile, err := perTestUnavailable(context.Background(), &models.Options{}, "example.test/verify", errors.New("profile unavailable"))
	if err != nil || profile != nil {
		t.Fatalf("fallback property violation: unavailable profile returned (%v, %v)", profile, err)
	}
	for _, p := range []*coverage.PerTestProfile{profile, new(coverage.PerTestProfile)} {
		if filter := perTestRunFilter(p, "/tmp/target.go", 4); filter != "" {
			t.Fatalf("fallback property violation: absent coverage restricted tests to %q", filter)
		}
	}
}
