package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunMutantIDScopesOtherMutants covers #276: mutant selection is applied
// before coverage classification, and dry runs use the same scope.
func TestRunMutantIDScopesOtherMutants(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), `module example.com/scopebug

go 1.26.6
`)
	writeFixtureFile(t, filepath.Join(root, "go-mutesting.yml"), `enable_mutators:
  - arithmetic/base
`)
	writeFixtureFile(t, filepath.Join(root, "calc.go"), `package scopebug

func Add(a, b int) int { return a + b }

func Sum(a, b int) int { return a - b }
`)
	writeFixtureFile(t, filepath.Join(root, "calc_test.go"), `package scopebug

import "testing"

func TestAdd(t *testing.T) { Add(1, 2) }
`)

	agenticPath := filepath.Join(root, "go-mutesting-agentic.json")
	summaryPath := filepath.Join(root, "go-mutesting-summary.json")
	previousAgentic := models.ReportAgenticJSONFileName
	previousSummary := models.ReportSummaryJSONFileName
	models.ReportAgenticJSONFileName = agenticPath
	models.ReportSummaryJSONFileName = summaryPath
	t.Cleanup(func() {
		models.ReportAgenticJSONFileName = previousAgentic
		models.ReportSummaryJSONFileName = previousSummary
	})

	common := []string{"--workers", "1", "--exec-timeout", "30", "--config", "go-mutesting.yml", "."}
	testMain(t, root, append([]string{"--logger-agentic-json", "--logger-summary-json"}, common...), returnOk, "mutation score")
	allStats := readScopeSummary(t, summaryPath)
	addID, sumID := scopeBugIDs(t, agenticPath)
	require.NotEqual(t, addID, sumID)

	covered := testMain(t, root, append([]string{"--coverage", "--run-mutant-id", addID, "--logger-summary-json"}, common...), returnOk, "ESCAPED")
	assert.NotContains(t, covered, "NOT COVERED")
	assert.NotContains(t, covered, "mutation score")
	assert.NotContains(t, covered, "calc.go:5")
	coveredStats := readScopeSummary(t, summaryPath)
	assertScopeSummaryConsistent(t, coveredStats)
	assert.Equal(t, int64(0), coveredStats.NotCoveredCount)
	assert.Less(t, coveredStats.TotalMutantsCount, allStats.TotalMutantsCount)

	dry := testMain(t, root, append([]string{"--dry-run", "--run-mutant-id", addID}, common...), returnOk, "mutation(s) would be generated")
	dryCount, err := parseScopeDryRunCount(dry)
	require.NoError(t, err)
	assert.Equal(t, coveredStats.TotalMutantsCount, dryCount)

	uncovered := testMain(t, root, append([]string{"--coverage", "--run-mutant-id", sumID, "--logger-summary-json"}, common...), returnOk, "NOT COVERED")
	assert.NotContains(t, uncovered, "No mutant with ID")
	assert.NotContains(t, uncovered, "ESCAPED")
	assert.NotContains(t, uncovered, "KILLED")
	assert.NotContains(t, uncovered, "calc.go:3")
	assert.NotContains(t, uncovered, "mutation score")
	uncoveredStats := readScopeSummary(t, summaryPath)
	assertScopeSummaryConsistent(t, uncoveredStats)
	assert.Greater(t, uncoveredStats.NotCoveredCount, int64(0))
	assert.Less(t, uncoveredStats.TotalMutantsCount, allStats.TotalMutantsCount)
}

func TestDryRunGitDiffCountMatchesRealRun(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "go.mod"), `module example.com/scopediff

go 1.26.6
`)
	writeFixtureFile(t, filepath.Join(root, "go-mutesting.yml"), `enable_mutators:
  - arithmetic/base
`)
	writeFixtureFile(t, filepath.Join(root, "calc.go"), `package scopediff

func Add(a, b int) int { return a + b }

func Sum(a, b int) int { return a - b }
`)
	writeFixtureFile(t, filepath.Join(root, "calc_test.go"), `package scopediff

import "testing"

func TestAdd(t *testing.T) { Add(1, 2) }
`)
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "go-mutesting@example.com")
	runGit(t, root, "config", "user.name", "go-mutesting test")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "base")
	writeFixtureFile(t, filepath.Join(root, "calc.go"), `package scopediff

func Add(a, b int) int { return a + b } // changed

func Sum(a, b int) int { return a - b }
`)

	summaryPath := filepath.Join(root, "go-mutesting-summary.json")
	previousSummary := models.ReportSummaryJSONFileName
	models.ReportSummaryJSONFileName = summaryPath
	t.Cleanup(func() { models.ReportSummaryJSONFileName = previousSummary })

	common := []string{"--workers", "1", "--exec-timeout", "30", "--git-diff-lines", "--git-diff-base", "HEAD", "--config", "go-mutesting.yml", "."}
	dry := testMain(t, root, append([]string{"--dry-run"}, common...), returnOk, "mutation(s) would be generated")
	real := testMain(t, root, append([]string{"--logger-summary-json"}, common...), returnOk, "mutation score")
	assert.NotContains(t, real, "calc.go:5")

	dryCount, err := parseScopeDryRunCount(dry)
	require.NoError(t, err)
	stats := readScopeSummary(t, summaryPath)
	assertScopeSummaryConsistent(t, stats)
	assert.Equal(t, dryCount, stats.TotalMutantsCount)
}

func scopeBugIDs(t *testing.T, path string) (addID, sumID string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var report struct {
		Mutants []struct {
			ID   string `json:"id"`
			Line int64  `json:"line"`
		} `json:"mutants"`
	}
	require.NoError(t, json.Unmarshal(data, &report))
	for _, mutant := range report.Mutants {
		switch mutant.Line {
		case 3:
			addID = mutant.ID
		case 5:
			sumID = mutant.ID
		}
	}
	require.NotEmpty(t, addID)
	require.NotEmpty(t, sumID)
	return addID, sumID
}

func readScopeSummary(t *testing.T, path string) models.Stats {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var stats models.Stats
	require.NoError(t, json.Unmarshal(data, &stats))
	return stats
}

func assertScopeSummaryConsistent(t *testing.T, stats models.Stats) {
	t.Helper()
	assert.Equal(t, stats.TotalMutantsCount, stats.KilledCount+stats.EscapedCount+stats.ErrorCount+stats.SkippedCount+stats.NotCoveredCount)
}

func parseScopeDryRunCount(out string) (int64, error) {
	start := strings.LastIndex(out, "Total: ")
	if start < 0 {
		return 0, fmt.Errorf("dry-run total missing from output %q", out)
	}
	var count int64
	_, err := fmt.Sscanf(out[start:], "Total: %d", &count)
	return count, err
}
