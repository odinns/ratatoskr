package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odinns/ratatoskr/internal/ratatoskr"
)

func TestScanJSONAndReportJSONAreValid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	writeFile(t, filepath.Join(root, "dist/app.js"), "dist")

	for _, command := range []string{"scan", "report"} {
		var out bytes.Buffer
		cmd := NewRootCommandWithOutput(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{command, "--path", root, "--format", "json"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s failed: %v\n%s", command, err, out.String())
		}

		var result ratatoskr.ScanResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatalf("%s emitted invalid JSON: %v\n%s", command, err, out.String())
		}
		if result.SchemaVersion != ratatoskr.SchemaVersion || len(result.Candidates) == 0 || len(result.ActiveRules) == 0 {
			t.Fatalf("%s emitted incomplete result: %#v", command, result)
		}
	}
}

func TestRulesListsInspectableFields(t *testing.T) {
	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"rules"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, fragment := range []string{"laravel-storage-logs", "source:", "category:", "risk:", "matches:", "exclusions:", "reason:", "consequence:", "cleanable by default:"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("rules output missing %q:\n%s", fragment, text)
		}
	}
}

func TestSummaryReadsReportFile(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example/node_modules","size_bytes":200,"apparent_size_bytes":200,"category":"dependencies","risk":"cautious","rule":"node-modules","reason":"x","consequence":"y","cleanable_by_default":false},
    {"path":"/tmp/example/storage/logs/app.log","size_bytes":100,"apparent_size_bytes":100,"category":"logs","risk":"safe","rule":"laravel-storage-logs","reason":"x","consequence":"y","cleanable_by_default":true}
  ],
  "skipped": [{"path":"/tmp/example/.git","reason":"repository metadata is protected"}],
  "errors": [],
  "active_rules": []
}`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--limit", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	text := out.String()
	for _, fragment := range []string{"Ratatoskr scan summary", "Candidates: 2", "Cautious rebuildable waste", "node_modules", "Reports contain local file paths"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("summary output missing %q:\n%s", fragment, text)
		}
	}
}

func TestSummaryKeepsDangerousCandidatesOutOfActionableBuckets(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example/storage/logs/app.log","size_bytes":100,"apparent_size_bytes":100,"category":"logs","risk":"safe","rule":"laravel-storage-logs","reason":"x","consequence":"y","cleanable_by_default":true},
    {"path":"/tmp/example/target","size_bytes":200,"apparent_size_bytes":200,"category":"builds","risk":"cautious","rule":"rust-target","reason":"x","consequence":"y","cleanable_by_default":false},
    {"path":"/tmp/example/big.bin","size_bytes":300,"apparent_size_bytes":300,"category":"unknown-large-files","risk":"dangerous","rule":"unknown-large-file","reason":"x","consequence":"y","cleanable_by_default":false}
  ],
  "skipped": [{"path":"/tmp/example/.git","reason":"repository metadata is protected"}],
  "errors": [{"path":"/tmp/example/nope","error":"permission denied"}],
  "active_rules": []
}`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--limit", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	text := out.String()
	for _, fragment := range []string{"Safe generated waste", "Cautious rebuildable waste", "Manual review", "Report-only / do-not-touch", "Skipped: 1", "Errors: 1"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("summary output missing %q:\n%s", fragment, text)
		}
	}
	reportOnlySection := text[strings.Index(text, "Report-only / do-not-touch"):]
	if strings.Contains(reportOnlySection, "safe to clean") || strings.Contains(reportOnlySection, "ready to remove") {
		t.Fatalf("dangerous section should not use actionable cleanup wording:\n%s", reportOnlySection)
	}
}

func TestSummaryBucketsComeFromRiskNotMetadata(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example/expensive-safe","size_bytes":100,"apparent_size_bytes":100,"category":"logs","risk":"safe","rule":"laravel-storage-logs","reason":"x","consequence":"y","cleanable_by_default":true,"rebuild_cost":"extreme","reclaim_durability":"one minute","preferred_cleanup":"inspect first"},
    {"path":"/tmp/example/cheap-cautious","size_bytes":200,"apparent_size_bytes":200,"category":"builds","risk":"cautious","rule":"rust-target","reason":"x","consequence":"y","cleanable_by_default":false,"rebuild_cost":"none","reclaim_durability":"forever","preferred_cleanup":"use owner tool"},
    {"path":"/tmp/example/model.gguf","size_bytes":300,"apparent_size_bytes":300,"category":"local-ai-models","risk":"cautious","rule":"lm-studio-model-file","reason":"x","consequence":"y","cleanable_by_default":false,"rebuild_cost":"none","reclaim_durability":"forever","preferred_cleanup":"use owner tool"}
  ],
  "skipped": [],
  "errors": [],
  "active_rules": []
}`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--limit", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	text := out.String()
	safeStart := strings.LastIndex(text, "\nSafe generated waste")
	cautiousStart := strings.LastIndex(text, "\nCautious rebuildable waste")
	manualStart := strings.LastIndex(text, "\nManual review")
	reportOnlyStart := strings.LastIndex(text, "\nReport-only / do-not-touch")
	if safeStart < 0 || cautiousStart < 0 || manualStart < 0 || reportOnlyStart < 0 {
		t.Fatalf("summary missing expected sections:\n%s", text)
	}
	safeSection := text[safeStart:cautiousStart]
	cautiousSection := text[cautiousStart:manualStart]
	manualSection := text[manualStart:reportOnlyStart]
	if !strings.Contains(safeSection, "expensive-safe") || strings.Contains(safeSection, "cheap-cautious") {
		t.Fatalf("safe section should only use risk, not metadata:\n%s", text)
	}
	if !strings.Contains(cautiousSection, "cheap-cautious") || strings.Contains(cautiousSection, "expensive-safe") {
		t.Fatalf("cautious section should only use risk, not metadata:\n%s", text)
	}
	if !strings.Contains(manualSection, "model.gguf") || strings.Contains(manualSection, "cheap-cautious") {
		t.Fatalf("manual section should contain cautious non-rebuildable candidates:\n%s", text)
	}
}

func TestTargetSpaceProjectionRecommendsSafeThenCautiousOnly(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example/safe-a","size_bytes":100,"apparent_size_bytes":100,"category":"logs","risk":"safe","rule":"laravel-storage-logs","reason":"x","consequence":"safe consequence","cleanable_by_default":true,"rebuild_cost":"none","reclaim_durability":"short","preferred_cleanup":"review logs"},
    {"path":"/tmp/example/cautious-a","size_bytes":80,"apparent_size_bytes":80,"category":"builds","risk":"cautious","rule":"rust-target","reason":"x","consequence":"cautious consequence","cleanable_by_default":false,"rebuild_cost":"medium","reclaim_durability":"medium","preferred_cleanup":"use cargo"},
    {"path":"/tmp/example/dangerous-a","size_bytes":1000,"apparent_size_bytes":1000,"category":"unknown-large-files","risk":"dangerous","rule":"unknown-large-file","reason":"x","consequence":"y","cleanable_by_default":false}
  ],
  "skipped": [],
  "errors": [],
  "active_rules": []
}`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--target-bytes", "150"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	text := out.String()
	if !strings.Contains(text, "Target-space projection") || !strings.Contains(text, "safe-a") || !strings.Contains(text, "cautious-a") {
		t.Fatalf("target projection missing expected candidates:\n%s", text)
	}
	for _, fragment := range []string{"Safe selected: 100 B", "Cautious selected: 80 B", "total", "rebuild:", "durability:", "consequence:", "inspect:"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("target projection missing %q:\n%s", fragment, text)
		}
	}
	projection := text[strings.Index(text, "Target-space projection"):]
	if strings.Contains(projection, "dangerous-a") || strings.Contains(projection, "dangerous") || strings.Contains(projection, "ready to remove") || strings.Contains(projection, "deletion list") || strings.Contains(projection, "ratatoskr clean") {
		t.Fatalf("target projection should not include dangerous/actionable cleanup wording:\n%s", text)
	}
}

func TestSummaryShowsProjectGroupsAndQualityHints(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example-a/dist","size_bytes":100,"apparent_size_bytes":100,"category":"builds","risk":"safe","rule":"node-build-output","reason":"x","consequence":"y","cleanable_by_default":true,"project_root":"/tmp/example-a","project_type":"node","marker_file":"package.json","artifact_path":"dist","rebuild_cost":"low","reclaim_durability":"temporary"},
    {"path":"/tmp/example-b/node_modules","size_bytes":300,"apparent_size_bytes":300,"category":"dependencies","risk":"cautious","rule":"node-modules","reason":"x","consequence":"y","cleanable_by_default":false,"project_root":"/tmp/example-b","project_type":"node","marker_file":"package.json","artifact_path":"node_modules","rebuild_cost":"medium","reclaim_durability":"until next install"},
    {"path":"/tmp/cache/Mystery","size_bytes":500,"apparent_size_bytes":500,"category":"caches","risk":"cautious","rule":"explicit-application-cache","reason":"x","consequence":"y","cleanable_by_default":false}
  ],
  "skipped": [{"path":"/tmp/alias","reason":"duplicate of /tmp/real"}],
  "errors": [{"path":"/tmp/nope","error":"permission denied"}],
  "active_rules": []
}`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--limit", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	text := out.String()
	for _, fragment := range []string{"Project artifacts by project", "/tmp/example-b", "node_modules", "durability:", "Report quality hints", "duplicate-paths", "scan-errors", "broad-cache-fallbacks"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("summary output missing %q:\n%s", fragment, text)
		}
	}
}

func TestSummaryJSONIncludesProjectGroupsAndQualityHints(t *testing.T) {
	reportPath := writeSummaryReport(t)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--target", "150B", "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}

	var payload struct {
		ProjectArtifactGroups []ratatoskr.ProjectArtifactGroup `json:"project_artifact_groups"`
		ReportQualityHints    []ratatoskr.ReportQualityHint    `json:"report_quality_hints"`
		TargetProjection      *ratatoskr.TargetProjection      `json:"target_projection"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("summary emitted invalid JSON: %v\n%s", err, out.String())
	}
	if payload.ProjectArtifactGroups == nil || payload.ReportQualityHints == nil {
		t.Fatalf("summary JSON should include summary-specific arrays: %#v", payload)
	}
	if payload.TargetProjection == nil || len(payload.TargetProjection.RunningTotalBytes) == 0 {
		t.Fatalf("summary JSON target projection should include running totals: %#v", payload.TargetProjection)
	}
}

func TestSummaryParsesHumanTargetSizes(t *testing.T) {
	reportPath := writeSummaryReport(t)

	for _, test := range []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "decimal",
			args:     []string{"summary", "--file", reportPath, "--target", "1.5GB"},
			expected: "Target-space projection: 1.4 GiB",
		},
		{
			name:     "binary",
			args:     []string{"summary", "--file", reportPath, "--target", "2GiB"},
			expected: "Target-space projection: 2.0 GiB",
		},
		{
			name:     "target wins",
			args:     []string{"summary", "--file", reportPath, "--target", "150B", "--target-bytes", "999"},
			expected: "Target-space projection: 150 B",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := NewRootCommandWithOutput(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(test.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("summary failed: %v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), test.expected) {
				t.Fatalf("summary output missing %q:\n%s", test.expected, out.String())
			}
		})
	}
}

func TestSummaryRejectsInvalidHumanTargets(t *testing.T) {
	reportPath := writeSummaryReport(t)

	for _, target := range []string{"nonsense", "0GB", "-2GB"} {
		t.Run(target, func(t *testing.T) {
			var out bytes.Buffer
			cmd := NewRootCommandWithOutput(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"summary", "--file", reportPath, "--target", target})
			if err := cmd.Execute(); err == nil {
				t.Fatalf("expected target %q to fail\n%s", target, out.String())
			}
		})
	}
}

func TestSummaryTargetJSONIncludesProjection(t *testing.T) {
	reportPath := writeSummaryReport(t)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--target", "150B", "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}

	var payload struct {
		Totals           ratatoskr.Totals            `json:"totals"`
		TargetProjection *ratatoskr.TargetProjection `json:"target_projection"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("summary emitted invalid JSON: %v\n%s", err, out.String())
	}
	if payload.Totals.CandidateCount != 3 {
		t.Fatalf("summary JSON should include recalculated totals: %#v", payload.Totals)
	}
	if payload.TargetProjection == nil || payload.TargetProjection.TargetBytes != 150 || !payload.TargetProjection.TargetMet {
		t.Fatalf("summary JSON missing target projection: %#v", payload.TargetProjection)
	}
	if payload.TargetProjection.ProjectedBytes != 180 || payload.TargetProjection.ExcludedReportOnlyCount != 1 || payload.TargetProjection.ExcludedReportOnlyBytes != 1000 {
		t.Fatalf("summary JSON projection has wrong totals: %#v", payload.TargetProjection)
	}
}

func TestSummaryJSONOmitsProjectionWithoutTarget(t *testing.T) {
	reportPath := writeSummaryReport(t)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath, "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("summary failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "target_projection") {
		t.Fatalf("summary JSON should omit target projection without target:\n%s", out.String())
	}
}

func TestSummaryRejectsMalformedJSON(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "partial.json")
	writeFile(t, reportPath, `{"schema_version":"ratatoskr.scan.v1","candidates":[`)

	var out bytes.Buffer
	cmd := NewRootCommandWithOutput(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"summary", "--file", reportPath})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected summary to reject malformed JSON")
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSummaryReport(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.json")
	writeFile(t, reportPath, `{
  "schema_version": "ratatoskr.scan.v1",
  "generated_at": "2026-05-10T12:00:00Z",
  "scan_roots": ["/tmp/example"],
  "totals": {},
  "candidates": [
    {"path":"/tmp/example/safe-a","size_bytes":100,"apparent_size_bytes":100,"category":"logs","risk":"safe","rule":"laravel-storage-logs","reason":"x","consequence":"y","cleanable_by_default":true,"rebuild_cost":"none","reclaim_durability":"short"},
    {"path":"/tmp/example/cautious-a","size_bytes":80,"apparent_size_bytes":80,"category":"builds","risk":"cautious","rule":"rust-target","reason":"x","consequence":"y","cleanable_by_default":false,"rebuild_cost":"medium","reclaim_durability":"medium"},
    {"path":"/tmp/example/dangerous-a","size_bytes":1000,"apparent_size_bytes":1000,"category":"unknown-large-files","risk":"dangerous","rule":"unknown-large-file","reason":"x","consequence":"y","cleanable_by_default":false}
  ],
  "skipped": [],
  "errors": [],
  "active_rules": []
}`)
	return reportPath
}
