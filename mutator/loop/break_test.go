package loop

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorLoopBreak(t *testing.T) {
	test.Mutator(
		t,
		MutatorLoopBreak,
		"../../testdata/loop/break.go",
		2,
	)
}
