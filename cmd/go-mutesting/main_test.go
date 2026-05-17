package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestMainSimple(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec-timeout", "1"},
		returnOk,
		"mutation score",
	)
}

func TestMainRecursive(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec-timeout", "1", "./..."},
		returnOk,
		"mutation score",
	)
}

func TestMainFromOtherDirectory(t *testing.T) {
	testMain(
		t,
		"../..",
		[]string{"--debug", "--exec-timeout", "1", "github.com/jonbaldie/go-mutesting/v2/example"},
		returnOk,
		"mutation score",
	)
}

func TestMainMatch(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec", "../scripts/exec/test-mutated-package.sh", "--exec-timeout", "1", "--match", "baz", "./..."},
		returnOk,
		"mutation score",
	)
}

func TestMainSkipWithoutTest(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec-timeout", "1", "--config", "../testdata/configs/configSkipWithoutTest.yml.test"},
		returnOk,
		"mutation score",
	)
}

func TestMainMinMsiPass(t *testing.T) {
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec-timeout", "1", "--min-msi", "1"},
		returnOk,
		"mutation score",
	)
}

func TestMainMinMsiFail(t *testing.T) {
	// 101 exceeds the maximum possible MSI (100%) so the gate always fires.
	testMain(
		t,
		"../../example",
		[]string{"--debug", "--exec-timeout", "1", "--min-msi", "101"},
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
		[]string{"--debug", "--exec-timeout", "1", "--min-covered-msi", "90"},
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
		[]string{"--debug", "--exec-timeout", "1", "--config", "../testdata/configs/configForJson.yml.test"},
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
	// Totals must be internally consistent.
	assert.Equal(t, s.TotalMutantsCount, s.KilledCount+s.EscapedCount+s.ErrorCount+s.SkippedCount+s.NotCoveredCount)
	// Something must have been mutated and tested.
	assert.Greater(t, s.KilledCount+s.EscapedCount, int64(0))
	// MSI must be a valid ratio.
	assert.GreaterOrEqual(t, s.Msi, 0.0)
	assert.LessOrEqual(t, s.Msi, 1.0)
	// Collection lengths must match the stat fields.
	assert.Equal(t, int(s.EscapedCount), len(mutationReport.Escaped))
	assert.Equal(t, int(s.KilledCount), len(mutationReport.Killed))
	assert.Nil(t, mutationReport.Timeouted)
	assert.Nil(t, mutationReport.Errored)

	for i := 0; i < len(mutationReport.Escaped); i++ {
		assert.Contains(t, mutationReport.Escaped[i].ProcessOutput, "FAIL")
	}
	for i := 0; i < len(mutationReport.Killed); i++ {
		assert.Contains(t, mutationReport.Killed[i].ProcessOutput, "PASS")
	}
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
