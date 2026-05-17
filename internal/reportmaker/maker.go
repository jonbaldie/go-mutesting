package reportmaker

import (
	_ "embed" // for embedding report template
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonbaldie/go-mutesting/v2/internal/baseline"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
)

// mutatorDescriptions maps mutator names to plain-English explanations.
var mutatorDescriptions = map[string]string{
	"arithmetic/base":              "Swaps an arithmetic operator (+, -, *, /) for a different one",
	"branch/if":                    "Removes an if-block body so the condition becomes a no-op",
	"branch/else":                  "Removes an else-block body",
	"concurrency/goroutine_remove": "Converts a goroutine launch to a regular blocking call, removing concurrency",
	"conditional/boundaryNegation": "Negates a boundary condition (< becomes >=, > becomes <=, etc.)",
	"conditional/negation":         "Negates a boolean condition (true becomes false and vice versa)",
	"expression/remove":            "Removes an expression statement entirely, dropping its side effect",
	"loop/break":                   "Removes a break statement, potentially causing an infinite loop",
	"loop/condition":               "Changes the loop's termination condition",
	"loop/range_break":             "Removes a break statement inside a range loop",
	"numbers/incrementer":          "Increments a numeric literal by 1",
	"numbers/decrementer":          "Decrements a numeric literal by 1",
	"select/case_remove":           "Removes a case from a select statement, reducing channel handling paths",
	"select/default_remove":        "Removes the default case from a select statement",
	"statement/remove":             "Removes a statement entirely, dropping its side effect or return value",
}

// killHints maps mutator names to heuristic advice for writing a killing test.
var killHints = map[string]string{
	"arithmetic/base":              "Write a test with specific numeric inputs and assert the exact output — boundary values expose operator swaps best",
	"branch/if":                    "Write a test that enters this branch and asserts the output or side effect it produces",
	"branch/else":                  "Write a test where the else path is taken and assert its expected result",
	"concurrency/goroutine_remove": "Write a test that asserts concurrent behaviour — e.g. a channel receive, a timing constraint, or a race-detector hit",
	"conditional/boundaryNegation": "Write tests at the exact boundary value — one that satisfies the condition and one that doesn't — and assert different outcomes",
	"conditional/negation":         "Write tests that exercise both the true and false branches and assert different outcomes for each",
	"expression/remove":            "Write a test that asserts the side effect or state change this expression produces",
	"loop/break":                   "Write a test that asserts the loop terminates at the right iteration",
	"loop/condition":               "Write a test with a known input and assert the exact number of loop iterations or the final state",
	"loop/range_break":             "Write a test that asserts the loop stops at the correct element",
	"numbers/incrementer":          "Write a test that asserts the exact numeric value — off-by-one mutations are killed by precise equality assertions",
	"numbers/decrementer":          "Write a test that asserts the exact numeric value",
	"select/case_remove":           "Write a test that sends on the removed channel case and asserts the expected receive or resulting action",
	"select/default_remove":        "Write a test where no channel is ready (the default path) and assert its behaviour",
	"statement/remove":             "Write a test that asserts the side effect or state change this statement produces",
}

//go:embed templates/report.html.gotpl
var reportTmpl string

var funcMap = template.FuncMap{
	"splitDiff": func(diff string) []string {
		return strings.Split(diff, "\n")
	},
	"hasPrefix": strings.HasPrefix,
}

// MakeHTMLReport is a function for creating an HTML report based on a stripped-down version of the models.Report model (not all fields are used)
func MakeHTMLReport(report models.Report) error {
	// MSI in percent
	report.Stats.Msi = math.Round(report.Stats.Msi*10_000) / 100
	groupedMutants := groupEscapedMutants(report.Escaped)

	t, err := template.New(models.ReportHTMLFileName).Funcs(funcMap).Parse(reportTmpl)
	if err != nil {
		return fmt.Errorf("Error while parse template: %w ", err)
	}

	file, err := createOrTruncateReportFile(models.ReportHTMLFileName)
	if err != nil {
		return fmt.Errorf("Error while open/create .html report file from template: %w ", err)
	}
	defer closeReportFile(file, models.ReportHTMLFileName)

	data := struct {
		Stats          models.Stats
		GroupedMutants map[string][]models.Mutant
	}{
		Stats:          report.Stats,
		GroupedMutants: groupedMutants,
	}

	err = t.Execute(file, data)
	if err != nil {
		return fmt.Errorf("Error while execute template for .html report: %w ", err)
	}

	return nil
}

// MakeJSONReport is a function for creating json report, which is based on models.Report
func MakeJSONReport(report models.Report) error {
	jsonContent, err := json.Marshal(report)
	if err != nil {
		return err
	}

	file, err := createOrTruncateReportFile(models.ReportFileName)
	if err != nil {
		return fmt.Errorf("Error while open/create .json report file from template: %w ", err)
	}
	defer closeReportFile(file, models.ReportFileName)

	if file == nil {
		return errors.New("cannot create file for .json report")
	}

	_, err = file.WriteString(string(jsonContent))
	if err != nil {
		return err
	}

	return nil
}

// MakeSummaryJSONReport writes a compact stats-only JSON to go-mutesting-summary.json.
// Useful for badge generation and CI dashboards that don't need per-mutant detail.
func MakeSummaryJSONReport(stats models.Stats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	file, err := createOrTruncateReportFile(models.ReportSummaryJSONFileName)
	if err != nil {
		return fmt.Errorf("Error while open/create summary JSON report file: %w", err)
	}
	defer closeReportFile(file, models.ReportSummaryJSONFileName)

	if file == nil {
		return errors.New("cannot create file for summary JSON report")
	}

	_, err = file.WriteString(string(data))
	return err
}

func createOrTruncateReportFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func closeReportFile(file *os.File, filename string) {
	if err := file.Close(); err != nil {
		fmt.Printf("Error while closing %s: %v\n", filename, err)
	}
}

// AgenticMutant describes one escaped mutant for LLM consumption.
type AgenticMutant struct {
	ID           string   `json:"id"`
	File         string   `json:"file"`
	Line         int64    `json:"line"`
	Mutator      string   `json:"mutator"`
	Description  string   `json:"description,omitempty"`
	KillHint     string   `json:"kill_hint,omitempty"`
	Diff         string   `json:"diff"`
	ContextLines []string `json:"context_lines,omitempty"`
	TestFiles    []string `json:"test_files,omitempty"`
}

type agenticReport struct {
	GeneratedAt  string          `json:"generated_at"`
	Msi          float64         `json:"msi"`
	EscapedCount int             `json:"escaped_count"`
	Mutants      []AgenticMutant `json:"mutants"`
}

// MakeAgenticJSONReport writes go-mutesting-agentic.json with enriched escaped-mutant
// data designed for LLM consumption: stable IDs, context lines, test file paths,
// mutator descriptions, and heuristic test-writing hints.
func MakeAgenticJSONReport(report models.Report, moduleRoot string) error {
	msi := math.Round(report.Stats.Msi*10_000) / 100
	mutants := make([]AgenticMutant, 0, len(report.Escaped))
	for _, m := range report.Escaped {
		relFile := toRelPath(m.Mutator.OriginalFilePath, moduleRoot)
		id := baseline.MutantID(relFile, m.Mutator.MutatorName, m.Diff)
		mutants = append(mutants, AgenticMutant{
			ID:           id,
			File:         relFile,
			Line:         m.Mutator.OriginalStartLine,
			Mutator:      m.Mutator.MutatorName,
			Description:  mutatorDescriptions[m.Mutator.MutatorName],
			KillHint:     killHints[m.Mutator.MutatorName],
			Diff:         m.Diff,
			ContextLines: extractContextLines(m.Mutator.OriginalSourceCode, int(m.Mutator.OriginalStartLine), 3),
			TestFiles:    findTestFiles(filepath.Dir(m.Mutator.OriginalFilePath), moduleRoot),
		})
	}

	doc := agenticReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Msi:          msi,
		EscapedCount: len(report.Escaped),
		Mutants:      mutants,
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	file, err := createOrTruncateReportFile(models.ReportAgenticJSONFileName)
	if err != nil {
		return fmt.Errorf("Error while open/create agentic JSON report file: %w", err)
	}
	defer closeReportFile(file, models.ReportAgenticJSONFileName)

	if file == nil {
		return errors.New("cannot create file for agentic JSON report")
	}

	_, err = file.WriteString(string(data))
	return err
}

// extractContextLines returns up to radius lines before and after line (1-based)
// from the given source string.
func extractContextLines(source string, line, radius int) []string {
	if source == "" || line <= 0 {
		return nil
	}
	lines := strings.Split(source, "\n")
	start := max(line-radius-1, 0)
	end := min(line+radius-1, len(lines)-1)
	return lines[start : end+1]
}

func toRelPath(absOrRel, moduleRoot string) string {
	rel, err := filepath.Rel(moduleRoot, absOrRel)
	if err != nil {
		return filepath.ToSlash(absOrRel)
	}
	return filepath.ToSlash(rel)
}

// findTestFiles returns relative paths to *_test.go files in dir.
func findTestFiles(dir, moduleRoot string) []string {
	matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	result := make([]string, 0, len(matches))
	for _, f := range matches {
		if rel, err := filepath.Rel(moduleRoot, f); err == nil {
			result = append(result, filepath.ToSlash(rel))
		} else {
			result = append(result, filepath.ToSlash(f))
		}
	}
	return result
}

func groupEscapedMutants(escaped []models.Mutant) map[string][]models.Mutant {
	if len(escaped) == 0 {
		return make(map[string][]models.Mutant)
	}

	mutantCount := make(map[string]int)
	for _, mutant := range escaped {
		filePath := mutant.Mutator.OriginalFilePath
		mutantCount[filePath]++
	}

	groupedMutants := make(map[string][]models.Mutant, len(mutantCount))
	for filePath, count := range mutantCount {
		groupedMutants[filePath] = make([]models.Mutant, 0, count)
	}

	for _, mutant := range escaped {
		filePath := mutant.Mutator.OriginalFilePath
		groupedMutants[filePath] = append(groupedMutants[filePath], mutant)
	}

	return groupedMutants
}
