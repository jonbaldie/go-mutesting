package parser

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestClearDoesNotAcceptAnInFlightLoad(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, data := range map[string]string{"go.mod": "module example.com/cachetest\n\ngo 1.20\n", "p.go": "package p\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	bin := t.TempDir()
	started, finish := filepath.Join(bin, "started"), filepath.Join(bin, "finish")
	script := "#!/bin/sh\nif [ \"$1\" = list ]; then\n  : > \"$CACHE_TEST_STARTED\"\n  while [ ! -e \"$CACHE_TEST_FINISH\" ]; do sleep 0.01; done\nfi\nexec \"$CACHE_TEST_GO\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CACHE_TEST_GO", goBinary)
	t.Setenv("CACHE_TEST_STARTED", started)
	t.Setenv("CACHE_TEST_FINISH", finish)
	ClearPackageCache()
	done := make(chan struct{})
	go func() { loadPkgForDir(dir); close(done) }()
	t.Cleanup(func() { _ = os.WriteFile(finish, nil, 0600); <-done; ClearPackageCache() })
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("load did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	ClearPackageCache()
	if err := os.WriteFile(finish, nil, 0600); err != nil {
		t.Fatal(err)
	}
	<-done
	_, ok, _ := cachedPackages.packageFor(dir)
	if ok {
		t.Fatal("load begun before clear repopulated the cache")
	}
}
