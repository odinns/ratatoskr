package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odinn/ratatoskr/internal/ratatoskr"
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
	for _, fragment := range []string{"Ratatoskr scan summary", "Candidates: 2", "Top cautious", "node_modules", "Reports contain local file paths"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("summary output missing %q:\n%s", fragment, text)
		}
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
