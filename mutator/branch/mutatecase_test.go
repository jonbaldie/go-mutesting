package branch

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorCase(t *testing.T) {
	test.Mutator(
		t,
		MutatorCase,
		"../../testdata/branch/mutatecase.go",
		3,
	)
}
