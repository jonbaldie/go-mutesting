package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
)

func TestVerifyCLIEarlyExitPrecedence(t *testing.T) {
	old := os.Stdout
	defer func() { os.Stdout = old }()
	for bits := 0; bits < 16; bits++ {
		opts := models.NewOptions()
		opts.General.Version = bits&1 != 0
		opts.General.Help = bits&2 != 0
		opts.Mutator.ListMutators = bits&4 != 0
		completion := bits&8 != 0
		if completion {
			t.Setenv("GO_FLAGS_COMPLETION", "1")
		} else {
			t.Setenv("GO_FLAGS_COMPLETION", "")
		}
		p := flags.NewNamedParser("go-mutesting", flags.None)
		if _, err := p.AddGroup("go-mutesting", "go-mutesting arguments", opts); err != nil {
			t.Fatal(err)
		}
		output, err := os.CreateTemp(t.TempDir(), "output")
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = output
		handled, code := handleEarlyExitFlags(opts, p, []string{"target"})
		if err := output.Close(); err != nil {
			t.Fatal(err)
		}
		os.Stdout = old
		data, err := os.ReadFile(output.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		expected := "none"
		switch {
		case opts.General.Version:
			expected = "version"
		case opts.General.Help && !completion:
			expected = "help"
		case opts.Mutator.ListMutators:
			expected = "list"
		}
		actual := "none"
		if handled {
			if code != returnOk {
				t.Fatalf("precedence bits=%d: exit %d", bits, code)
			}
			switch {
			case strings.HasPrefix(text, "go-mutesting "):
				actual = "version"
			case strings.HasPrefix(text, "Usage:"):
				actual = "help"
			case strings.Contains(text, "arithmetic"):
				actual = "list"
			default:
				t.Fatalf("precedence bits=%d: unexpected output %q", bits, text)
			}
		}
		if actual != expected {
			t.Fatalf("precedence bits=%d: got %s want %s", bits, actual, expected)
		}
	}
}

func TestVerifyCLIHandledBeforeErrors(t *testing.T) {
	oldOut, oldErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	for _, args := range [][]string{
		{"--version", "--not-a-real-flag"},
		{"--help", "--not-a-real-flag"},
		{"--help", "--config", "/path/that/does/not/exist"},
	} {
		file, err := os.CreateTemp(t.TempDir(), "output")
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout, os.Stderr = file, file
		handled, code := checkArguments(args, models.NewOptions())
		os.Stdout, os.Stderr = oldOut, oldErr
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if !handled || code != returnOk {
			t.Fatalf("early exit was not handled before errors for %v: %v %d", args, handled, code)
		}
	}
}

func TestVerifyCLIDispatch(t *testing.T) {
	t.Setenv("GO_FLAGS_COMPLETION", "")
	dir := t.TempDir()
	source := "main.go"
	baseline := filepath.Join(dir, "bad.json")
	config := filepath.Join(dir, "bad.yaml")
	for path, text := range map[string]string{baseline: "not json", config: "unknown_field: true\n"} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
		code int
	}{
		{"config before target", []string{"--config=" + config}, "Could not parse config", returnError},
		{"target before baseline", []string{"--baseline=" + baseline, filepath.Join(dir, "missing.go")}, "Could not find any suitable", returnError},
		{"baseline before info", []string{"--baseline=" + baseline, "--list-files", source}, "Cannot load baseline", returnError},
		{"list files", []string{"--list-files", source}, source, returnOk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "out")
			if err != nil {
				t.Fatal(err)
			}
			oldOut, oldErr := os.Stdout, os.Stderr
			os.Stdout, os.Stderr = f, f
			code := mainCmd(tc.args)
			os.Stdout, os.Stderr = oldOut, oldErr
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			out, err := os.ReadFile(f.Name())
			if err != nil {
				t.Fatal(err)
			}
			if code != tc.code || !strings.Contains(string(out), tc.want) {
				t.Fatalf("dispatch %s: got %d %q, want %d containing %q", tc.name, code, out, tc.code, tc.want)
			}
		})
	}
}
