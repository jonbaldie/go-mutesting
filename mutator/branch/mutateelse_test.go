package branch

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/mutator"
	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorElse(t *testing.T) {
	test.Mutator(
		t,
		MutatorElse,
		"../../testdata/branch/mutateelse.go",
		1,
	)
}

func TestMutatorElseRegistered(t *testing.T) {
	if _, err := mutator.New("branch/else"); err != nil {
		t.Fatalf("mutator not registered: %v", err)
	}
}
