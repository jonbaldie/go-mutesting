package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/baseline"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
)

func TestVerificationBaselinePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if f, err := baseline.Load(path); err != nil || f != nil {
		t.Fatalf("missing load: file=%v err=%v", f, err)
	}
	opts := models.NewOptions()
	opts.Baseline.File = path
	report := &models.Report{Escaped: []models.Mutant{{ID: "survivor"}}}
	run := func() int {
		return finalizeResults(&bytes.Buffer{}, &bytes.Buffer{}, opts, report, baseline.Capture(nil), sourceRoots{module: dir}, true)
	}
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	run()
	if b, err := os.ReadFile(path); err != nil || string(b) != "original" {
		t.Fatalf("disabled update changed file: %q %v", b, err)
	}
	opts.Baseline.Update = true
	report.Stats.ErrorCount = 1
	if code := run(); code != returnError {
		t.Fatalf("errors exit: %d", code)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "original" {
		t.Fatalf("errored update changed file: %q %v", b, err)
	}
	report.Stats.ErrorCount = 0
	if code := run(); code != returnOk {
		t.Fatalf("update exit: %d", code)
	}
	f, err := baseline.Load(path)
	if err != nil || f == nil || len(f.Mutants) != 1 || f.Mutants[0].ID != "survivor" {
		t.Fatalf("reload: file=%v err=%v", f, err)
	}
	opts.Baseline.File = dir
	if code := run(); code != returnError {
		t.Fatalf("write failure exit: %d", code)
	}
	if _, err := baseline.Load(dir); err == nil {
		t.Fatal("directory load should fail")
	}
	if err := os.WriteFile(path, []byte("invalid JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := baseline.Load(path); err == nil {
		t.Fatal("invalid baseline should fail")
	}
}
