package conditional

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorBoolLiteral(t *testing.T) {
	test.Mutator(
		t,
		MutatorBoolLiteral,
		"../../testdata/conditional/bool_literal.go",
		2,
	)
}
