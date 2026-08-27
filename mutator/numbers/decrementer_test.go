package numbers

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/mutator"
	"github.com/jonbaldie/go-mutesting/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutatorNumbersDecrementer(t *testing.T) {
	test.Mutator(
		t,
		MutatorNumbersDecrementer,
		"../../testdata/numbers/decrementer.go",
		3,
	)
}

func TestMutatorNumbersDecrementerRegistered(t *testing.T) {
	if _, err := mutator.New("numbers/decrementer"); err != nil {
		t.Fatalf("mutator not registered: %v", err)
	}
}

func TestMutatorNumbersDecrementerParenthesizesNegativeValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		kind     token.Token
		original string
		mutated  string
	}{
		{name: "integer", kind: token.INT, original: "0", mutated: "(-1)"},
		{name: "positive integer", kind: token.INT, original: "1", mutated: "0"},
		{name: "float", kind: token.FLOAT, original: "0.5", mutated: "(-0.5)"},
		{name: "positive float", kind: token.FLOAT, original: "1.0", mutated: "0"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			node := &ast.BasicLit{Kind: testCase.kind, Value: testCase.original}
			mutations := MutatorNumbersDecrementer(nil, nil, node)
			require.Len(t, mutations, 1)

			mutations[0].Change()
			assert.Equal(t, testCase.mutated, node.Value)

			mutations[0].Reset()
			assert.Equal(t, testCase.original, node.Value)
		})
	}
}
