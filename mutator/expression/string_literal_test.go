package expression

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorStringLiteral(t *testing.T) {
	test.Mutator(
		t,
		MutatorStringLiteral,
		"../../testdata/expression/string_literal.go",
		2,
	)
}
