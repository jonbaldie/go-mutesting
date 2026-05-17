package expression

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorLogical(t *testing.T) {
	test.Mutator(
		t,
		MutatorLogical,
		"../../testdata/expression/logical.go",
		2,
	)
}
