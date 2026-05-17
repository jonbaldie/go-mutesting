package selectmutator

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorSelectCaseRemove(t *testing.T) {
	test.Mutator(
		t,
		MutatorSelectCaseRemove,
		"../../testdata/select/case_remove.go",
		2,
	)
}
