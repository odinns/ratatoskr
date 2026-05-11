package ratatoskr

import "time"

type Risk string

const (
	RiskSafe      Risk = "safe"
	RiskCautious  Risk = "cautious"
	RiskDangerous Risk = "dangerous"
)

type Category string

const (
	CategoryCaches            Category = "caches"
	CategoryLogs              Category = "logs"
	CategoryBuilds            Category = "builds"
	CategoryDependencies      Category = "dependencies"
	CategoryFrameworkRuntime  Category = "framework-runtime"
	CategoryPackageManager    Category = "package-manager"
	CategoryUnknownLargeFiles Category = "unknown-large-files"
	CategoryReportOnly        Category = "report-only"
	CategoryLocalAIModels     Category = "local-ai-models"
)

type Rule struct {
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	Category           Category `json:"category"`
	Risk               Risk     `json:"risk"`
	PathPatterns       []string `json:"path_patterns"`
	AppliesWhen        []string `json:"applies_when"`
	Exclusions         []string `json:"exclusions"`
	Reason             string   `json:"reason"`
	Consequence        string   `json:"consequence"`
	CleanableByDefault bool     `json:"cleanable_by_default"`
	RebuildCost        string   `json:"rebuild_cost"`
	ReclaimDurability  string   `json:"reclaim_durability"`
	PreferredCleanup   string   `json:"preferred_cleanup"`
}

type Candidate struct {
	Path               string   `json:"path"`
	SizeBytes          int64    `json:"size_bytes"`
	ApparentSizeBytes  int64    `json:"apparent_size_bytes"`
	Category           Category `json:"category"`
	Risk               Risk     `json:"risk"`
	Rule               string   `json:"rule"`
	Reason             string   `json:"reason"`
	Consequence        string   `json:"consequence"`
	CleanableByDefault bool     `json:"cleanable_by_default"`
	RebuildCost        string   `json:"rebuild_cost"`
	ReclaimDurability  string   `json:"reclaim_durability"`
	PreferredCleanup   string   `json:"preferred_cleanup"`
	ProjectRoot        string   `json:"project_root,omitempty"`
	MarkerFile         string   `json:"marker_file,omitempty"`
	ProjectType        string   `json:"project_type,omitempty"`
	ArtifactPath       string   `json:"artifact_path,omitempty"`
}

type SkippedPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type ScanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type Totals struct {
	CandidateCount       int   `json:"candidate_count"`
	TotalBytes           int64 `json:"total_bytes"`
	SafeBytes            int64 `json:"safe_bytes"`
	CautiousBytes        int64 `json:"cautious_bytes"`
	DangerousBytes       int64 `json:"dangerous_bytes"`
	CleanableByDefault   int   `json:"cleanable_by_default"`
	CleanableDefaultSize int64 `json:"cleanable_default_size"`
}

type ScanResult struct {
	SchemaVersion string        `json:"schema_version"`
	GeneratedAt   time.Time     `json:"generated_at"`
	ScanRoots     []string      `json:"scan_roots"`
	Totals        Totals        `json:"totals"`
	Candidates    []Candidate   `json:"candidates"`
	Skipped       []SkippedPath `json:"skipped"`
	Errors        []ScanError   `json:"errors"`
	ActiveRules   []Rule        `json:"active_rules"`
}

type TargetProjection struct {
	TargetBytes             int64       `json:"target_bytes"`
	SafeAvailableBytes      int64       `json:"safe_available_bytes"`
	SafeSelectedBytes       int64       `json:"safe_selected_bytes"`
	CautiousSelectedBytes   int64       `json:"cautious_selected_bytes"`
	ProjectedBytes          int64       `json:"projected_bytes"`
	RemainingBytes          int64       `json:"remaining_bytes"`
	TargetMet               bool        `json:"target_met"`
	RecommendedCandidates   []Candidate `json:"recommended_candidates"`
	ExcludedReportOnlyBytes int64       `json:"excluded_report_only_bytes"`
	ExcludedReportOnlyCount int         `json:"excluded_report_only_count"`
	Note                    string      `json:"note"`
}
