package statement

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/test"
)

func TestMutatorDeferRemove(t *testing.T) {
	test.Mutator(
		t,
		MutatorDeferRemove,
		"../../testdata/statement/defer_remove.go",
		2,
	)
}

func TestMutatorDeferRemoveSelect(t *testing.T) {
	test.Mutator(
		t,
		MutatorDeferRemove,
		"../../testdata/statement/defer_remove_select.go",
		1,
	)
}
