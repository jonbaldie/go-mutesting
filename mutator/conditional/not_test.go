package conditional

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorConditionalNot(t *testing.T) {
	test.Mutator(
		t,
		MutatorConditionalNot,
		"../../testdata/conditional/not.go",
		2,
	)
}

func TestMutatorConditionalNotForStmt(t *testing.T) {
	test.Mutator(
		t,
		MutatorConditionalNot,
		"../../testdata/conditional/not_for.go",
		1,
	)
}
