package arithmetic

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/mutator"
	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorArithmeticBitwise(t *testing.T) {
	test.Mutator(
		t,
		MutatorArithmeticBitwise,
		"../../testdata/arithmetic/bitwise.go",
		6,
	)
}

func TestMutatorArithmeticBitwiseRegistered(t *testing.T) {
	if _, err := mutator.New("arithmetic/bitwise"); err != nil {
		t.Fatalf("mutator not registered: %v", err)
	}
}
