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

func topCandidates(candidates []Candidate, limit int, risk Risk) []Candidate {
	filtered := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if risk == "" || candidate.Risk == risk {
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
