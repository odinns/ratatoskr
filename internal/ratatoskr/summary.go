package ratatoskr

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Summary struct {
	Result ScanResult
}

func ReadSummary(r io.Reader) (Summary, error) {
	var result ScanResult
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&result); err != nil {
		return Summary{}, fmt.Errorf("could not read scan report JSON: %w", err)
	}
	result.Totals = calculateTotals(result.Candidates)
	return Summary{Result: result}, nil
}

func (s Summary) TopCandidates(limit int) []Candidate {
	return topCandidates(s.Result.Candidates, limit, "")
}

func (s Summary) TopCandidatesByRisk(risk Risk, limit int) []Candidate {
	return topCandidates(s.Result.Candidates, limit, risk)
}

func (s Summary) CautiousRebuildableCandidates(limit int) []Candidate {
	return topCandidatesByPredicate(s.Result.Candidates, limit, func(candidate Candidate) bool {
		return candidate.Risk == RiskCautious && isRebuildableCategory(candidate.Category)
	})
}

func (s Summary) ManualReviewCandidates(limit int) []Candidate {
	return topCandidatesByPredicate(s.Result.Candidates, limit, func(candidate Candidate) bool {
		return candidate.Risk == RiskCautious && !isRebuildableCategory(candidate.Category)
	})
}

func (s Summary) ProjectArtifactGroups(limit int) []ProjectArtifactGroup {
	groupsByRoot := map[string]*ProjectArtifactGroup{}
	for _, candidate := range s.Result.Candidates {
		if candidate.ProjectRoot == "" {
			continue
		}
		if !isRebuildableCategory(candidate.Category) {
			continue
		}
		group, ok := groupsByRoot[candidate.ProjectRoot]
		if !ok {
			group = &ProjectArtifactGroup{
				ProjectRoot: candidate.ProjectRoot,
				ProjectType: candidate.ProjectType,
				Candidates:  []Candidate{},
			}
			groupsByRoot[candidate.ProjectRoot] = group
		}
		if group.ProjectType == "" && candidate.ProjectType != "" {
			group.ProjectType = candidate.ProjectType
		}
		group.TotalBytes += candidate.SizeBytes
		group.Candidates = append(group.Candidates, candidate)
	}
	groups := make([]ProjectArtifactGroup, 0, len(groupsByRoot))
	for _, group := range groupsByRoot {
		sort.Slice(group.Candidates, func(i, j int) bool {
			if group.Candidates[i].SizeBytes == group.Candidates[j].SizeBytes {
				return group.Candidates[i].Path < group.Candidates[j].Path
			}
			return group.Candidates[i].SizeBytes > group.Candidates[j].SizeBytes
		})
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].TotalBytes == groups[j].TotalBytes {
			return groups[i].ProjectRoot < groups[j].ProjectRoot
		}
		return groups[i].TotalBytes > groups[j].TotalBytes
	})
	if limit > 0 && len(groups) > limit {
		return groups[:limit]
	}
	return groups
}

func (s Summary) ReportQualityHints() []ReportQualityHint {
	hints := []ReportQualityHint{}
	duplicateSkips := 0
	for _, skipped := range s.Result.Skipped {
		if strings.Contains(strings.ToLower(skipped.Reason), "duplicate") {
			duplicateSkips++
		}
	}
	if duplicateSkips > 0 {
		hints = append(hints, ReportQualityHint{
			Kind:    "duplicate-paths",
			Message: "Some paths looked like duplicates and were skipped. Totals may be lower than a plain directory listing suggests.",
			Count:   duplicateSkips,
		})
	}
	if len(s.Result.Skipped) >= 10 {
		hints = append(hints, ReportQualityHint{
			Kind:    "many-skipped-paths",
			Message: "Many paths were skipped. Treat the report as incomplete until the skips are understood.",
			Count:   len(s.Result.Skipped),
		})
	}
	if len(s.Result.Errors) > 0 {
		hints = append(hints, ReportQualityHint{
			Kind:    "scan-errors",
			Message: "Scan errors were recorded. Fix those before trusting the totals too hard.",
			Count:   len(s.Result.Errors),
		})
	}
	var fallbackBytes int64
	var fallbackCount int
	for _, candidate := range s.Result.Candidates {
		if candidate.Rule == "explicit-application-cache" || candidate.Rule == "explicit-package-cache" {
			fallbackBytes += candidate.SizeBytes
			fallbackCount++
		}
	}
	if fallbackCount > 0 && fallbackBytes*2 >= s.Result.Totals.TotalBytes {
		hints = append(hints, ReportQualityHint{
			Kind:    "broad-cache-fallbacks",
			Message: "Broad cache fallback rules dominate this report. Inspect those owners before treating the space as easy relief.",
			Count:   fallbackCount,
			Bytes:   fallbackBytes,
		})
	}
	return hints
}

func (s Summary) TargetProjection(targetBytes int64) TargetProjection {
	projection := TargetProjection{
		TargetBytes:           targetBytes,
		RecommendedCandidates: []Candidate{},
		RunningTotalBytes:     []int64{},
		Note:                  "Read-only planning projection. Inspect these candidates first; report-only paths are excluded.",
	}
	if targetBytes <= 0 {
		return projection
	}
	for _, candidate := range s.Result.Candidates {
		if candidate.Risk == RiskDangerous || candidate.Category == CategoryReportOnly {
			projection.ExcludedReportOnlyBytes += candidate.SizeBytes
			projection.ExcludedReportOnlyCount++
		}
	}

	safe := topCandidatesByPredicate(s.Result.Candidates, 0, func(candidate Candidate) bool {
		return candidate.Risk == RiskSafe && candidate.CleanableByDefault && candidate.Category != CategoryReportOnly
	})
	for _, candidate := range safe {
		projection.SafeAvailableBytes += candidate.SizeBytes
		projection.addCandidate(candidate)
		if projection.ProjectedBytes >= targetBytes {
			projection.finish()
			return projection
		}
	}

	cautious := topCandidatesByPredicate(s.Result.Candidates, 0, func(candidate Candidate) bool {
		return candidate.Risk == RiskCautious && candidate.Category != CategoryReportOnly
	})
	for _, candidate := range cautious {
		projection.addCandidate(candidate)
		if projection.ProjectedBytes >= targetBytes {
			break
		}
	}

	projection.finish()
	return projection
}

func (p *TargetProjection) addCandidate(candidate Candidate) {
	p.RecommendedCandidates = append(p.RecommendedCandidates, candidate)
	p.ProjectedBytes += candidate.SizeBytes
	p.RunningTotalBytes = append(p.RunningTotalBytes, p.ProjectedBytes)
	if candidate.Risk == RiskSafe {
		p.SafeSelectedBytes += candidate.SizeBytes
		return
	}
	if candidate.Risk == RiskCautious {
		p.CautiousSelectedBytes += candidate.SizeBytes
	}
}

func (p *TargetProjection) finish() {
	p.TargetMet = p.ProjectedBytes >= p.TargetBytes
	if p.TargetMet {
		p.RemainingBytes = 0
		return
	}
	p.RemainingBytes = p.TargetBytes - p.ProjectedBytes
}

func topCandidates(candidates []Candidate, limit int, risk Risk) []Candidate {
	return topCandidatesByPredicate(candidates, limit, func(candidate Candidate) bool {
		return risk == "" || candidate.Risk == risk
	})
}

func topCandidatesByPredicate(candidates []Candidate, limit int, keep func(Candidate) bool) []Candidate {
	filtered := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if keep(candidate) {
			filtered = append(filtered, candidate)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SizeBytes == filtered[j].SizeBytes {
			return filtered[i].Path < filtered[j].Path
		}
		return filtered[i].SizeBytes > filtered[j].SizeBytes
	})
	if limit > 0 && len(filtered) > limit {
		return filtered[:limit]
	}
	return filtered
}

func isRebuildableCategory(category Category) bool {
	switch category {
	case CategoryBuilds, CategoryDependencies, CategoryFrameworkRuntime, CategoryPackageManager:
		return true
	default:
		return false
	}
}
