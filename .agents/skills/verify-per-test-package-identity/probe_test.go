package coverage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const probeModule = "example.test/verify"

type identityToolchain struct {
	owners map[string]string
}

func (identityToolchain) Context() context.Context      { return context.Background() }
func (identityToolchain) ListTests(string) *exec.Cmd    { panic("unexpected ListTests") }
func (identityToolchain) ListPackages(string) *exec.Cmd { panic("unexpected ListPackages") }

func (identityToolchain) CompileTestBinary(pkg, _, output string) *exec.Cmd {
	return exec.Command("sh", "-c", `printf '%s' "$1" > "$2"`, "compile", pkg, output)
}

func (tc identityToolchain) RunTest(binary, name, profile string) *exec.Cmd {
	const run = `owner=$(cat "$1") || exit 1
hit=0
if [ "$owner" = "$2" ]; then hit=1; fi
printf 'mode: set\nexample.test/verify/target.go:4.1,4.9 1 %s\n' "$hit" > "$3"`
	return exec.Command("sh", "-c", run, "run", binary, tc.owners[name], profile)
}

func profilePair(t *testing.T, dir, first, second string) (*PerTestProfile, error) {
	t.Helper()
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatalf("probe setup: create pair directory: %v", err)
	}
	owners := map[string]string{"TestFirst": first, "TestSecond": second}
	pkgs := []TestPackage{
		{ImportPath: first, Tests: []string{"TestFirst"}},
		{ImportPath: second, Tests: []string{"TestSecond"}},
	}
	return BuildPerTestProfileForPackages(identityToolchain{owners}, probeModule, pkgs, probeModule, dir, 1)
}

func checkToolchain(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "harness")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("probe setup: create harness directory: %v", err)
	}
	first, second := probeModule+"/plain/one", probeModule+"/plain/two"
	tc := identityToolchain{map[string]string{"TestFirst": first, "TestSecond": second}}
	binary := filepath.Join(dir, "binary")
	if out, err := tc.CompileTestBinary(first, probeModule, binary).CombinedOutput(); err != nil {
		t.Fatalf("probe setup: compile control: %v: %s", err, out)
	}
	for _, test := range []struct{ name, hit string }{{"TestFirst", "1"}, {"TestSecond", "0"}} {
		profile := filepath.Join(dir, test.name)
		if out, err := tc.RunTest(binary, test.name, profile).CombinedOutput(); err != nil {
			t.Fatalf("probe setup: run control: %v: %s", err, out)
		}
		data, err := os.ReadFile(profile)
		if err != nil || !strings.Contains(string(data), "1 "+test.hit+"\n") {
			t.Fatalf("probe setup: wrong control profile %q: %v", data, err)
		}
	}
}

func TestGeneratedPackageIdentity(t *testing.T) {
	root := t.TempDir()
	checkToolchain(t, root)
	control, err := profilePair(t, filepath.Join(root, "control"), probeModule+"/plain/one", probeModule+"/plain/two")
	if err != nil || control == nil {
		t.Fatalf("probe setup: disjoint control did not produce a profile: %v", err)
	}
	if got := control.CoveringTestsRelative("target.go", 4); !slices.Contains(got, "TestFirst") || !slices.Contains(got, "TestSecond") {
		t.Errorf("incomplete profiles; first: disjoint packages returned covering tests %v; both tests cover the line", got)
	}

	// All ordered pairs of distinct two-segment package paths drawn from this
	// finite alphabet. Every segment starts with a letter and is a legal Go
	// directory name. The underscore variants exercise separator ambiguity.
	segments := []string{"a", "b", "a_a", "a_b"}
	var paths []string
	for _, first := range segments {
		for _, second := range segments {
			paths = append(paths, probeModule+"/"+first+"/"+second)
		}
	}
	checked, incomplete, fallback := 0, 0, 0
	var firstFailure string
	for _, first := range paths {
		for _, second := range paths {
			if first == second {
				continue
			}
			checked++
			dir := filepath.Join(root, fmt.Sprintf("pair-%03d", checked))
			profile, err := profilePair(t, dir, first, second)
			if err != nil || profile == nil {
				fallback++
			} else {
				got := profile.CoveringTestsRelative("target.go", 4)
				if !slices.Contains(got, "TestFirst") || !slices.Contains(got, "TestSecond") {
					incomplete++
					if firstFailure == "" {
						firstFailure = fmt.Sprintf("packages %q and %q returned covering tests %v; both tests cover the line", first, second, got)
					}
				}
			}
			if err := os.RemoveAll(dir); err != nil {
				t.Fatalf("probe setup: remove pair directory: %v", err)
			}
		}
	}
	t.Logf("checked %d ordered pairs; %d requested full-suite fallback", checked, fallback)
	if incomplete > 0 {
		t.Fatalf("%d incomplete profiles; first: %s", incomplete, firstFailure)
	}
}
