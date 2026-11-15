package engine

import (
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
	"testing"
)

type collectorProbeListener struct {
	events  []models.MutationEvent
	summary *models.Report
}

func (l *collectorProbeListener) OnMutantCompleted(e models.MutationEvent) {
	l.events = append(l.events, e)
}
func (l *collectorProbeListener) OnProgress(models.Stats)    {}
func (l *collectorProbeListener) OnSummary(r *models.Report) { l.summary = r }

func TestVerifyCollector(t *testing.T) {
	l := &collectorProbeListener{}
	c := &completedResults{listener: l}
	cases := []struct {
		outcome models.MutantOutcome
		id      string
	}{
		{models.OutcomeKilled, "k"}, {models.OutcomeEscaped, "e"},
		{models.OutcomeSkipped, "s"}, {models.OutcomeErrored, "x"},
		{models.OutcomeNotCovered, "n"},
	}
	for _, item := range cases {
		c.record(item.outcome, models.Mutant{ID: item.id})
	}
	if len(l.events) != len(cases) {
		t.Fatalf("events: got %d", len(l.events))
	}
	for i, item := range cases {
		e := l.events[i]
		if e.Mutant.ID != item.id || e.Outcome != item.outcome {
			t.Fatalf("event identity: %+v", e)
		}
		total := e.Stats.KilledCount + e.Stats.EscapedCount + e.Stats.SkippedCount + e.Stats.ErrorCount + e.Stats.NotCoveredCount
		if total != int64(i+1) {
			t.Fatalf("event stats: got %d, want %d", total, i+1)
		}
		counts := []int64{e.Stats.KilledCount, e.Stats.EscapedCount, e.Stats.SkippedCount, e.Stats.ErrorCount, e.Stats.NotCoveredCount}
		for j, count := range counts {
			var want int64
			if j <= i {
				want = 1
			}
			if count != want {
				t.Fatalf("event stats: bucket %d got %d, want %d", j, count, want)
			}
		}
	}
	report := &models.Report{}
	c.summarize(report)
	if l.summary != report {
		t.Fatal("summary not published")
	}
	groups := [][]models.Mutant{report.Killed, report.Escaped, report.Skipped, report.Errored, report.NotCovered}
	counts := []int64{report.Stats.KilledCount, report.Stats.EscapedCount, report.Stats.SkippedCount, report.Stats.ErrorCount, report.Stats.NotCoveredCount}
	for i, item := range cases {
		if len(groups[i]) != 1 || groups[i][0].ID != item.id {
			t.Fatalf("report bucket %s: %+v", item.id, groups[i])
		}
		if counts[i] != int64(len(groups[i])) {
			t.Fatalf("report count %s: %d", item.id, counts[i])
		}
	}
	report.Escaped[0].ID = "edited"
	again := &models.Report{}
	c.fill(again)
	if again.Escaped[0].ID != "e" {
		t.Fatalf("snapshot alias: %s", again.Escaped[0].ID)
	}
}
