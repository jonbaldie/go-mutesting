package arithmetic

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorArithmeticBitwise(t *testing.T) {
	test.Mutator(
		t,
		MutatorArithmeticBitwise,
		"../../testdata/arithmetic/bitwise.go",
		6,
	)
}
