package coverage

import (
	"reflect"
	"testing"
)

func TestPathResolutionCacheOwnedByProfile(t *testing.T) {
	const path = "/repo/pkg/a.go"
	first := &Profile{coveredRanges: map[string][]lineRange{"pkg/a.go": {{start: 1, end: 1}}}}
	second := &Profile{coveredRanges: map[string][]lineRange{"pkg/a.go": {{start: 2, end: 2}}}}
	if !first.IsCovered(path, 1) || !second.IsCovered(path, 2) || second.IsCovered(path, 1) {
		t.Fatal("profile cache leaked across owners")
	}
	first.coveredRanges["pkg/a.go"] = []lineRange{{start: 3, end: 3}}
	if !first.IsCovered(path, 1) || first.IsCovered(path, 3) {
		t.Fatal("cached profile hit changed")
	}

	one := &PerTestProfile{data: map[string]map[string][]lineRange{"pkg/a.go": {"one": {{start: 1, end: 1}}}}}
	two := &PerTestProfile{data: map[string]map[string][]lineRange{"pkg/a.go": {"two": {{start: 1, end: 1}}}}}
	if !reflect.DeepEqual(one.CoveringTests(path, 1), []string{"one"}) || !reflect.DeepEqual(two.CoveringTests(path, 1), []string{"two"}) {
		t.Fatal("per-test cache leaked across owners")
	}
	one.data["pkg/a.go"] = map[string][]lineRange{"changed": {{start: 1, end: 1}}}
	if !reflect.DeepEqual(one.CoveringTests(path, 1), []string{"one"}) {
		t.Fatal("cached per-test hit changed")
	}
}
