package ratatoskr

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
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

func (s Summary) TargetProjection(targetBytes int64) TargetProjection {
	projection := TargetProjection{
		TargetBytes:           targetBytes,
		RecommendedCandidates: []Candidate{},
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
