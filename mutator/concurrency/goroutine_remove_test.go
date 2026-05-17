package concurrency

import (
	"testing"

	"github.com/jonbaldie/go-mutesting/test"
)

func TestMutatorGoroutineRemove(t *testing.T) {
	test.Mutator(
		t,
		MutatorGoroutineRemove,
		"../../testdata/concurrency/goroutine_remove.go",
		2,
	)
}
