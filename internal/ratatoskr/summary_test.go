package ratatoskr

import "testing"

func TestTargetProjectionUsesSafeCandidatesFirst(t *testing.T) {
	summary := Summary{Result: ScanResult{Candidates: []Candidate{
		candidate("small-safe", 100, RiskSafe, CategoryLogs, true),
		candidate("large-safe", 200, RiskSafe, CategoryLogs, true),
		candidate("unsafe-safe", 300, RiskSafe, CategoryLogs, false),
		candidate("cautious", 500, RiskCautious, CategoryBuilds, false),
	}}}

	projection := summary.TargetProjection(150)

	if !projection.TargetMet || projection.ProjectedBytes != 200 || projection.SafeSelectedBytes != 200 || projection.CautiousSelectedBytes != 0 {
		t.Fatalf("safe-only projection has wrong totals: %#v", projection)
	}
	if len(projection.RecommendedCandidates) != 1 || projection.RecommendedCandidates[0].Path != "large-safe" {
		t.Fatalf("safe-only projection should pick largest safe candidate first: %#v", projection.RecommendedCandidates)
	}
}

func TestTargetProjectionUsesCautiousWhenNoSafeCandidateQualifies(t *testing.T) {
	summary := Summary{Result: ScanResult{Candidates: []Candidate{
		candidate("unsafe-safe", 300, RiskSafe, CategoryLogs, false),
		candidate("cautious", 200, RiskCautious, CategoryBuilds, false),
	}}}

	projection := summary.TargetProjection(150)

	if !projection.TargetMet || projection.SafeSelectedBytes != 0 || projection.CautiousSelectedBytes != 200 {
		t.Fatalf("cautious-only projection has wrong totals: %#v", projection)
	}
	if len(projection.RecommendedCandidates) != 1 || projection.RecommendedCandidates[0].Path != "cautious" {
		t.Fatalf("projection should fall back to cautious candidates: %#v", projection.RecommendedCandidates)
	}
}

func TestTargetProjectionUsesCautiousAfterSafe(t *testing.T) {
	summary := Summary{Result: ScanResult{Candidates: []Candidate{
		candidate("safe", 100, RiskSafe, CategoryLogs, true),
		candidate("large-cautious", 80, RiskCautious, CategoryBuilds, false),
		candidate("small-cautious", 20, RiskCautious, CategoryBuilds, false),
	}}}

	projection := summary.TargetProjection(150)

	if !projection.TargetMet || projection.ProjectedBytes != 180 || projection.SafeSelectedBytes != 100 || projection.CautiousSelectedBytes != 80 {
		t.Fatalf("safe-plus-cautious projection has wrong totals: %#v", projection)
	}
	if len(projection.RecommendedCandidates) != 2 || projection.RecommendedCandidates[0].Path != "safe" || projection.RecommendedCandidates[1].Path != "large-cautious" {
		t.Fatalf("projection selected wrong candidates: %#v", projection.RecommendedCandidates)
	}
}

func TestTargetProjectionReportsRemainingAndExcludesReportOnly(t *testing.T) {
	summary := Summary{Result: ScanResult{Candidates: []Candidate{
		candidate("safe", 100, RiskSafe, CategoryLogs, true),
		candidate("cautious", 80, RiskCautious, CategoryBuilds, false),
		candidate("dangerous", 1000, RiskDangerous, CategoryUnknownLargeFiles, false),
		candidate("report-only", 300, RiskCautious, CategoryReportOnly, false),
	}}}

	projection := summary.TargetProjection(500)

	if projection.TargetMet || projection.ProjectedBytes != 180 || projection.RemainingBytes != 320 {
		t.Fatalf("unmet projection has wrong totals: %#v", projection)
	}
	if projection.ExcludedReportOnlyCount != 2 || projection.ExcludedReportOnlyBytes != 1300 {
		t.Fatalf("projection should count excluded report-only candidates: %#v", projection)
	}
	for _, selected := range projection.RecommendedCandidates {
		if selected.Risk == RiskDangerous || selected.Category == CategoryReportOnly {
			t.Fatalf("projection selected report-only candidate: %#v", selected)
		}
	}
}

func TestTargetProjectionIgnoresDisplayMetadataForSelection(t *testing.T) {
	expensiveSafe := candidate("expensive-safe", 100, RiskSafe, CategoryLogs, true)
	expensiveSafe.RebuildCost = "extreme"
	expensiveSafe.ReclaimDurability = "one minute"
	cheapCautious := candidate("cheap-cautious", 200, RiskCautious, CategoryBuilds, false)
	cheapCautious.RebuildCost = "none"
	cheapCautious.ReclaimDurability = "forever"
	summary := Summary{Result: ScanResult{Candidates: []Candidate{expensiveSafe, cheapCautious}}}

	projection := summary.TargetProjection(100)

	if len(projection.RecommendedCandidates) != 1 || projection.RecommendedCandidates[0].Path != "expensive-safe" {
		t.Fatalf("metadata should not move cautious candidate ahead of safe candidate: %#v", projection.RecommendedCandidates)
	}
}

func candidate(path string, size int64, risk Risk, category Category, cleanable bool) Candidate {
	return Candidate{
		Path:               path,
		SizeBytes:          size,
		ApparentSizeBytes:  size,
		Category:           category,
		Risk:               risk,
		Rule:               "test-rule",
		Reason:             "test",
		Consequence:        "test",
		CleanableByDefault: cleanable,
	}
}
