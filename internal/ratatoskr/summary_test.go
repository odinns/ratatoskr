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
	if len(projection.RunningTotalBytes) != 1 || projection.RunningTotalBytes[0] != 200 {
		t.Fatalf("projection should include running totals: %#v", projection.RunningTotalBytes)
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

func TestProjectArtifactGroupsRankProjectsByTotalBytes(t *testing.T) {
	small := candidate("/repo-a/dist", 100, RiskSafe, CategoryBuilds, true)
	small.ProjectRoot = "/repo-a"
	small.ProjectType = "node"
	small.ArtifactPath = "dist"
	largeA := candidate("/repo-b/node_modules", 300, RiskCautious, CategoryDependencies, false)
	largeA.ProjectRoot = "/repo-b"
	largeA.ProjectType = "node"
	largeA.ArtifactPath = "node_modules"
	largeB := candidate("/repo-b/.next", 200, RiskSafe, CategoryBuilds, true)
	largeB.ProjectRoot = "/repo-b"
	largeB.ProjectType = "node"
	largeB.ArtifactPath = ".next"
	manual := candidate("/repo-b/model.gguf", 1000, RiskCautious, CategoryLocalAIModels, false)
	manual.ProjectRoot = "/repo-b"
	summary := Summary{Result: ScanResult{Candidates: []Candidate{small, largeA, largeB, manual}}}

	groups := summary.ProjectArtifactGroups(0)

	if len(groups) != 2 || groups[0].ProjectRoot != "/repo-b" || groups[0].TotalBytes != 500 {
		t.Fatalf("project artifact groups not ranked by bytes: %#v", groups)
	}
	if len(groups[0].Candidates) != 2 || groups[0].Candidates[0].Path != "/repo-b/node_modules" {
		t.Fatalf("project artifact groups should contain rebuildable project artifacts sorted by size: %#v", groups[0])
	}
}

func TestReportQualityHintsFlagDuplicatesErrorsAndFallbackDominance(t *testing.T) {
	fallback := candidate("/cache/Mystery", 1000, RiskCautious, CategoryCaches, false)
	fallback.Rule = "explicit-application-cache"
	named := candidate("/cache/Homebrew", 100, RiskCautious, CategoryCaches, false)
	named.Rule = "homebrew-cache"
	summary := Summary{Result: ScanResult{
		Candidates: []Candidate{fallback, named},
		Totals:     calculateTotals([]Candidate{fallback, named}),
		Skipped: []SkippedPath{
			{Path: "/alias", Reason: "duplicate of /real"},
		},
		Errors: []ScanError{{Path: "/bad", Error: "permission denied"}},
	}}

	hints := summary.ReportQualityHints()

	for _, kind := range []string{"duplicate-paths", "scan-errors", "broad-cache-fallbacks"} {
		if !hasHintKind(hints, kind) {
			t.Fatalf("missing %s hint: %#v", kind, hints)
		}
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

func hasHintKind(hints []ReportQualityHint, kind string) bool {
	for _, hint := range hints {
		if hint.Kind == kind {
			return true
		}
	}
	return false
}
