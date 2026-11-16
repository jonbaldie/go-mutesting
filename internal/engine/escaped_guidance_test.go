package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jonbaldie/go-mutesting/v2/internal/baseline"
	"github.com/jonbaldie/go-mutesting/v2/internal/models"
)

func TestQualityGatesEscapedGuidance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		file    *baseline.File
		escaped []models.Mutant
		want    int
		message string
	}{
		{
			name:    "without baseline",
			escaped: []models.Mutant{{ID: "first"}, {ID: "second"}},
			want:    returnMsiThresholdNotMet,
			message: "2 mutant(s) escaped. Add tests that fail on them.\n",
		},
		{
			name:    "new escape with baseline",
			file:    &baseline.File{Mutants: []baseline.Entry{{ID: "accepted"}}},
			escaped: []models.Mutant{{ID: "accepted"}, {ID: "new"}},
			want:    returnMsiThresholdNotMet,
			message: "1 new mutant(s) escaped. Add tests that fail on them.\n",
		},
		{
			name:    "accepted survivors",
			file:    &baseline.File{Mutants: []baseline.Entry{{ID: "accepted"}}},
			escaped: []models.Mutant{{ID: "accepted"}},
			want:    returnOk,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := models.NewOptions()
			opts.Score.FailOnEscaped = true
			report := &models.Report{Escaped: tc.escaped}
			report.Stats.EscapedCount = int64(len(tc.escaped))
			report.Calculate()
			var stderr bytes.Buffer
			if got := checkQualityGates(&stderr, opts, report, baseline.Capture(tc.file)); got != tc.want {
				t.Fatalf("exit = %d, want %d; stderr = %q", got, tc.want, stderr.String())
			}
			if got := stderr.String(); got != tc.message {
				t.Errorf("stderr = %q, want %q", got, tc.message)
			}
			if strings.Contains(stderr.String(), "--update-baseline") {
				t.Error("escaped-mutant diagnostic suggests updating the baseline")
			}
		})
	}
}
