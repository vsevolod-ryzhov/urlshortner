package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzersList(t *testing.T) {
	analyzers := getAnalyzers()

	assert.NotEmpty(t, analyzers, "Analyzers list should not be empty")

	analyzerNames := make(map[string]bool)
	for _, a := range analyzers {
		analyzerNames[a.Name] = true
	}

	t.Logf("Analyzers found: %v", analyzerNames)

	saCount, qfCount, stCount, sCount := 0, 0, 0, 0
	for name := range analyzerNames {
		switch {
		case len(name) >= 2 && name[:2] == "SA":
			saCount++
		case len(name) >= 2 && name[:2] == "QF":
			qfCount++
			t.Logf("Found QF analyzer: %s", name)
		case len(name) >= 2 && name[:2] == "ST":
			stCount++
			t.Logf("Found ST analyzer: %s", name)
		case len(name) >= 1 && name[:1] == "S" && (len(name) < 2 || name[:2] != "SA"):
			sCount++
			t.Logf("Found S analyzer: %s", name)
		}
	}

	t.Logf("SA count: %d, QF count: %d, ST count: %d, S count: %d",
		saCount, qfCount, stCount, sCount)

	expectedAnalyzers := []string{
		"printf", "shadow", "structtag", "httpresponse", "copylocks", "exitcheck",
	}
	for _, name := range expectedAnalyzers {
		assert.Contains(t, analyzerNames, name, "Analyzer %s should be present", name)
	}

	assert.Greater(t, saCount, 10, "Should have many SA analyzers, got %d", saCount)

	assert.Contains(t, analyzerNames, "QF1007", "QF1007 should be present")

	assert.Contains(t, analyzerNames, "ST1023", "ST1023 should be present")

	assert.Contains(t, analyzerNames, "S1039", "S1039 should be present")
}

func TestBuildTagsFlag(t *testing.T) {
	analyzers := getAnalyzers()

	for _, a := range analyzers {
		if a.Flags.Lookup("buildtags") != nil {
			assert.NotPanics(t, func() {
				err := a.Flags.Set("buildtags", "")
				assert.NoError(t, err)
			})
		}
	}
}

func TestNoDuplicateAnalyzers(t *testing.T) {
	analyzers := getAnalyzers()

	seen := make(map[string]bool)
	for _, a := range analyzers {
		assert.False(t, seen[a.Name], "Analyzer %s appears twice", a.Name)
		seen[a.Name] = true
	}
}

func TestStaticcheckCount(t *testing.T) {
	analyzers := getAnalyzers()

	counts := make(map[string]int)
	for _, a := range analyzers {
		name := a.Name
		if len(name) >= 2 {
			prefix := name[:2]
			counts[prefix]++
		} else if len(name) >= 1 {
			prefix := name[:1]
			counts[prefix]++
		}
	}

	t.Logf("Analyzer counts by prefix: %v", counts)

	assert.GreaterOrEqual(t, counts["SA"], 10, "Should have many SA analyzers")
}
