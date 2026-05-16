package models

// ReportFileName File name for json report
var ReportFileName string = "report.json"

// ReportHTMLFileName File name for html report
var ReportHTMLFileName string = "go-mutesting-report.html"

// Report holds the complete mutation testing result.
type Report struct {
	Stats          Stats          `json:"stats"`
	MutatorStats   []MutatorStats `json:"mutatorStats,omitempty"`
	Escaped        []Mutant       `json:"escaped"`
	Timeouted      []Mutant       `json:"timeouted"`
	Killed         []Mutant       `json:"killed"`
	Errored        []Mutant       `json:"errored"`
	NotCovered     []Mutant       `json:"notCovered,omitempty"`
}

// Stats holds aggregate mutation metrics.
type Stats struct {
	TotalMutantsCount    int64   `json:"totalMutantsCount"`
	KilledCount          int64   `json:"killedCount"`
	NotCoveredCount      int64   `json:"notCoveredCount"`
	EscapedCount         int64   `json:"escapedCount"`
	ErrorCount           int64   `json:"errorCount"`
	SkippedCount         int64   `json:"skippedCount"`
	TimeOutCount         int64   `json:"timeOutCount"`
	Msi                  float64 `json:"msi"`
	MutationCodeCoverage int64   `json:"mutationCodeCoverage"`
	CoveredCodeMsi       float64 `json:"coveredCodeMsi"`
	DuplicatedCount      int64   `json:"-"`
}

// MutatorStats holds per-mutator kill/escape counts.
type MutatorStats struct {
	Name    string `json:"name"`
	Killed  int64  `json:"killed"`
	Escaped int64  `json:"escaped"`
	Skipped int64  `json:"skipped"`
	Total   int64  `json:"total"`
}

// Mutant is the result of one mutation attempt.
type Mutant struct {
	Mutator       Mutator `json:"mutator"`
	Diff          string  `json:"diff"`
	ProcessOutput string  `json:"processOutput,omitempty"`
}

// Mutator describes what was mutated.
type Mutator struct {
	MutatorName        string `json:"mutatorName"`
	OriginalSourceCode string `json:"originalSourceCode"`
	MutatedSourceCode  string `json:"mutatedSourceCode"`
	OriginalFilePath   string `json:"originalFilePath"`
	OriginalStartLine  int64  `json:"originalStartLine"`
}

// Calculate computes derived metrics and per-mutator breakdowns.
func (report *Report) Calculate() {
	report.Stats.TotalMutantsCount = report.TotalCount()
	report.Stats.Msi = report.MsiScore()
	report.Stats.CoveredCodeMsi = report.CoveredMsiScore()
	report.MutatorStats = report.computeMutatorStats()
}

// MsiScore returns killed / total (skipped and errors count as killed).
func (report *Report) MsiScore() float64 {
	total := report.TotalCount()
	if total == 0 {
		return 0.0
	}
	return float64(report.Stats.KilledCount+report.Stats.ErrorCount+report.Stats.SkippedCount) / float64(total)
}

// CoveredMsiScore returns killed / (total - notCovered).
// Returns 0 when NotCoveredCount is zero, which indicates that coverage
// analysis was not performed (rather than that all code is covered).
// When you run with coverage enabled, any uncovered mutants will appear
// in NotCoveredCount and this metric becomes meaningful.
func (report *Report) CoveredMsiScore() float64 {
	if report.Stats.NotCoveredCount == 0 {
		return 0.0
	}
	covered := report.TotalCount() - report.Stats.NotCoveredCount
	if covered <= 0 {
		return 0.0
	}
	return float64(report.Stats.KilledCount+report.Stats.ErrorCount+report.Stats.SkippedCount) / float64(covered)
}

// TotalCount returns the count of all mutants that were actually tested plus
// those skipped due to compile errors, but NOT not-covered mutants.
func (report *Report) TotalCount() int64 {
	return report.Stats.KilledCount +
		report.Stats.EscapedCount +
		report.Stats.ErrorCount +
		report.Stats.SkippedCount +
		report.Stats.NotCoveredCount
}

// computeMutatorStats aggregates per-mutator kill/escape/skip counts.
func (report *Report) computeMutatorStats() []MutatorStats {
	counts := make(map[string]*MutatorStats)
	add := func(ms []Mutant, inc func(*MutatorStats)) {
		for _, m := range ms {
			name := m.Mutator.MutatorName
			if _, ok := counts[name]; !ok {
				counts[name] = &MutatorStats{Name: name}
			}
			inc(counts[name])
			counts[name].Total++
		}
	}
	add(report.Killed, func(s *MutatorStats) { s.Killed++ })
	add(report.Escaped, func(s *MutatorStats) { s.Escaped++ })
	add(report.Errored, func(s *MutatorStats) { s.Killed++ }) // errors count as kills
	add(report.NotCovered, func(s *MutatorStats) {})          // not counted in kill rate

	result := make([]MutatorStats, 0, len(counts))
	for _, s := range counts {
		result = append(result, *s)
	}
	return result
}
