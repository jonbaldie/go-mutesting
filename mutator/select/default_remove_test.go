package selectmutator

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorSelectDefaultRemove(t *testing.T) {
	test.Mutator(
		t,
		MutatorSelectDefaultRemove,
		"../../testdata/select/default_remove.go",
		1,
	)
}
