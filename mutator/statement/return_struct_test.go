package statement

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorReturnValueStruct(t *testing.T) {
	test.Mutator(
		t,
		MutatorReturnValue,
		"../../testdata/statement/return_struct.go",
		1,
	)
}
