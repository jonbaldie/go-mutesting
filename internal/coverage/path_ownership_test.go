package coverage

import (
	"reflect"
	"testing"
)

func TestPathResolutionCacheOwnedByProfile(t *testing.T) {
	const path = "/repo/pkg/a.go"
	first := &Profile{coveredLines: map[string]map[int]bool{"pkg/a.go": {1: true}}}
	second := &Profile{coveredLines: map[string]map[int]bool{"pkg/a.go": {2: true}}}
	if !first.IsCovered(path, 1) || !second.IsCovered(path, 2) || second.IsCovered(path, 1) {
		t.Fatal("profile cache leaked across owners")
	}
	first.coveredLines["pkg/a.go"] = map[int]bool{3: true}
	if !first.IsCovered(path, 1) || first.IsCovered(path, 3) {
		t.Fatal("cached profile hit changed")
	}

	one := &PerTestProfile{data: map[string]map[int][]string{"pkg/a.go": {1: {"one"}}}}
	two := &PerTestProfile{data: map[string]map[int][]string{"pkg/a.go": {1: {"two"}}}}
	if !reflect.DeepEqual(one.CoveringTests(path, 1), []string{"one"}) || !reflect.DeepEqual(two.CoveringTests(path, 1), []string{"two"}) {
		t.Fatal("per-test cache leaked across owners")
	}
	one.data["pkg/a.go"] = map[int][]string{1: {"changed"}}
	if !reflect.DeepEqual(one.CoveringTests(path, 1), []string{"one"}) {
		t.Fatal("cached per-test hit changed")
	}
}
