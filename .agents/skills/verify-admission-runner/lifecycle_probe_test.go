package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/gitdiff"
)

func TestVerifyScopePrecedence(t *testing.T) {
	for bits := 0; bits < 16; bits++ {
		c := scopeCandidate{repoFile: "a.go", line: 1, checksum: "a", id: "wanted"}
		var lines gitdiff.ChangedLines
		if bits&1 != 0 {
			lines = gitdiff.ChangedLines{"a.go": {{Start: 2, End: 2}}}
		}
		if bits&2 != 0 {
			c.id = "other"
		}
		blocked := map[string]struct{}{}
		if bits&4 != 0 {
			blocked["a"] = struct{}{}
		}
		scope := newMutantScope(lines, "wanted", blocked)
		if bits&8 != 0 {
			scope.seen["a"] = struct{}{}
		}
		want := admitReason
		switch {
		case bits&1 != 0:
			want = skipGitDiff
		case bits&2 != 0:
			want = skipMutantID
		case bits&4 != 0:
			want = skipBlacklist
		case bits&8 != 0:
			want = skipDuplicate
		}
		if got := scope.Admit(c); got != want {
			t.Fatalf("scope precedence bits=%d: got %v want %v", bits, got, want)
		}
	}
}

func TestVerifySinkPrecedence(t *testing.T) {
	opts := escapingPackageOptions(t)
	opts.General.DryRun = true
	opts.Exec.NoExec = true
	var stdout, stderr bytes.Buffer
	e := &Engine{Stdout: &stdout, Stderr: &stderr}
	result, err := e.Run(context.Background(), opts, nil)
	if err != nil || result.ExitCode != returnOk || !strings.Contains(stdout.String(), "No files written, no tests run") || result.Report.Stats.TotalMutantsCount == 0 {
		t.Fatalf("sink precedence: expected dry-run count; got result=%+v error=%v stdout=%q stderr=%q", result, err, stdout.String(), stderr.String())
	}
}
