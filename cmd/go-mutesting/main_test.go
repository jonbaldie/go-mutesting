package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/jonbaldie/go-mutesting/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestMainSimple(t *testing.T) {
	// Basic smoke test: tool runs and produces a summary.
	testMain(t, "../../example", []string{"--exec-timeout", "1"}, returnOk, "mutation score")
}

func TestMainRecursive(t *testing.T) {
	// ./... includes sub/, so the recursive run must mention the sub package.
	testMain(t, "../../example", []string{"--exec-timeout", "1", "./..."}, returnOk, "sub/")
}

func TestMainFromOtherDirectory(t *testing.T) {
	// Package-path resolution from the module root must work.
	testMain(t, "../..", []string{"--exec-timeout", "1", "github.com/jonbaldie/go-mutesting/example"}, returnOk, "mutation score")
}

func TestMainMatch(t *testing.T) {
	// --match baz restricts mutations to the baz function; the run must complete.
	testMain(t, "../../example", []string{"--exec", "../scripts/exec/test-mutated-package.sh", "--exec-timeout", "1", "--match", "baz", "./..."}, returnOk, "mutation score")
}

func TestMainSkipWithoutTest(t *testing.T) {
	// skip_without_test skips files that have no corresponding test file.
	testMain(t, "../../example", []string{"--exec-timeout", "1", "--config", "../testdata/configs/configSkipWithoutTest.yml.test"}, returnOk, "mutation score")
}

func TestMainMinMsiPass(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--min-msi", "1"},
		returnOk,
		"mutation score",
	)
}

func TestMainMinMsiFail(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--min-msi", "100"},
		returnMsiThresholdNotMet,
		"MSI",
	)
}

func TestMainMinCoveredMsiNoProfile(t *testing.T) {
	// Without coverage analysis NotCoveredCount==0 → covered MSI is 0.
	// A --min-covered-msi>0 gate should trigger.
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--min-covered-msi", "90"},
		returnMsiThresholdNotMet,
		"Covered MSI",
	)
}

func TestMainJSONReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-mutesting-main-test-")
	assert.NoError(t, err)

	reportFileName := "reportTestMainJSONReport.json"
	jsonFile := tmpDir + "/" + reportFileName
	if _, err := os.Stat(jsonFile); err == nil {
		err = os.Remove(jsonFile)
		assert.NoError(t, err)
	}

	models.ReportFileName = jsonFile

	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--config", "../testdata/configs/configForJson.yml.test"},
		returnOk,
		"mutation score",
	)

	info, err := os.Stat(jsonFile)
	assert.NoError(t, err)
	assert.NotNil(t, info)

	defer func() {
		err = os.Remove(jsonFile)
		if err != nil {
			fmt.Println("Error while deleting temp file")
		}
	}()

	jsonData, err := os.ReadFile(jsonFile)
	assert.NoError(t, err)

	var mutationReport models.Report
	err = json.Unmarshal(jsonData, &mutationReport)
	assert.NoError(t, err)

	s := mutationReport.Stats
	// All outcome counts must sum to the total.
	assert.Equal(t, s.TotalMutantsCount,
		s.KilledCount+s.EscapedCount+s.ErrorCount+s.SkippedCount+s.NotCoveredCount)
	// At least one mutation must have run.
	assert.Greater(t, s.TotalMutantsCount, int64(0))
	// MSI must be in [0, 1].
	assert.GreaterOrEqual(t, s.Msi, 0.0)
	assert.LessOrEqual(t, s.Msi, 1.0)
	// Slice lengths must match the counters.
	assert.Equal(t, int(s.KilledCount), len(mutationReport.Killed))
	assert.Equal(t, int(s.EscapedCount), len(mutationReport.Escaped))
	// ProcessOutput labels must match the outcome.
	for _, m := range mutationReport.Killed {
		assert.Contains(t, m.ProcessOutput, "PASS")
	}
	for _, m := range mutationReport.Escaped {
		assert.Contains(t, m.ProcessOutput, "FAIL")
	}
}

func TestMainQuiet(t *testing.T) {
	out, exitCode := captureMain(t, "../../example", []string{"--exec-timeout", "1", "--quiet"})
	assert.Equal(t, returnOk, exitCode)
	assert.Contains(t, out, "mutation score")
	assert.NotContains(t, out, "PASS")
}

func TestMainFailOnEscaped(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--fail-on-escaped"},
		returnMsiThresholdNotMet,
		"mutant(s) escaped",
	)
}

func TestMainCoverage(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "10", "--coverage"},
		returnOk,
		"covered-code mutation score",
	)
}

func TestMainGitDiffLines(t *testing.T) {
	// --git-diff-lines + --ignore-msi-with-no-mutations must exit 0 and produce
	// a mutation score summary regardless of how many mutations the diff filters in.
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--git-diff-lines", "--git-diff-base", "HEAD", "--ignore-msi-with-no-mutations"},
		returnOk,
		"mutation score",
	)
}

func TestMainLoggerGithub(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--exec-timeout", "1", "--logger-github"},
		returnOk,
		"::warning",
	)
}

// captureMain runs mainCmd and returns (combined stdout+stderr output, exit code).
func captureMain(t *testing.T, root string, args []string) (string, int) {
	t.Helper()
	saveStderr := os.Stderr
	saveStdout := os.Stdout
	saveCwd, err := os.Getwd()
	assert.Nil(t, err)

	r, w, err := os.Pipe()
	assert.Nil(t, err)

	os.Stderr = w
	os.Stdout = w
	assert.Nil(t, os.Chdir(root))

	bufChannel := make(chan string)
	go func() {
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, r)
		assert.Nil(t, err)
		assert.Nil(t, r.Close())
		bufChannel <- buf.String()
	}()

	exitCode := mainCmd(args)

	assert.Nil(t, w.Close())
	os.Stderr = saveStderr
	os.Stdout = saveStdout
	assert.Nil(t, os.Chdir(saveCwd))

	return <-bufChannel, exitCode
}

func testMain(t *testing.T, root string, exec []string, expectedExitCode int, contains string) {
	saveStderr := os.Stderr
	saveStdout := os.Stdout
	saveCwd, err := os.Getwd()
	assert.Nil(t, err)

	r, w, err := os.Pipe()
	assert.Nil(t, err)

	os.Stderr = w
	os.Stdout = w
	assert.Nil(t, os.Chdir(root))

	bufChannel := make(chan string)

	go func() {
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, r)
		assert.Nil(t, err)
		assert.Nil(t, r.Close())

		bufChannel <- buf.String()
	}()

	exitCode := mainCmd(exec)

	assert.Nil(t, w.Close())

	os.Stderr = saveStderr
	os.Stdout = saveStdout
	assert.Nil(t, os.Chdir(saveCwd))

	out := <-bufChannel

	assert.Equal(t, expectedExitCode, exitCode)
	assert.Contains(t, out, contains)
}
