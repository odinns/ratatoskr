package ratatoskr

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestScanFindsKnownGeneratedWaste(t *testing.T) {
	root := t.TempDir()
	laravel := filepath.Join(root, "laravel")
	node := filepath.Join(root, "node")
	writeFile(t, filepath.Join(laravel, "artisan"), "")
	writeFile(t, filepath.Join(laravel, "composer.json"), `{"require":{"laravel/framework":"^11.0"}}`)
	writeFile(t, filepath.Join(laravel, "storage/logs/laravel.log"), "log")
	writeFile(t, filepath.Join(laravel, "bootstrap/cache/packages.php"), "cache")
	writeFile(t, filepath.Join(laravel, "storage/framework/views/a.php"), "view")
	writeFile(t, filepath.Join(laravel, "storage/framework/cache/data/a.cache"), "cache")
	writeFile(t, filepath.Join(laravel, "storage/app/user-upload.txt"), "mine")
	writeFile(t, filepath.Join(laravel, "database/database.sqlite"), "db")
	writeFile(t, filepath.Join(laravel, ".env"), "APP_KEY=nope")

	writeFile(t, filepath.Join(node, "package.json"), "{}")
	writeFile(t, filepath.Join(node, ".next/cache.bin"), "next")
	writeFile(t, filepath.Join(node, "dist/app.js"), "dist")
	writeFile(t, filepath.Join(node, "node_modules/pkg/index.js"), "dep")

	result := scanForTest(t, root)

	wantRules := []string{
		"laravel-storage-logs",
		"laravel-bootstrap-cache",
		"laravel-compiled-views",
		"laravel-framework-cache",
		"node-build-output",
		"node-modules",
		"composer-vendor",
	}
	for _, rule := range wantRules[:6] {
		if !hasCandidateRule(result, rule) {
			t.Fatalf("missing candidate for rule %s; got %#v", rule, result.Candidates)
		}
	}
	assertNoCandidateContains(t, result, "storage/app")
	assertNoCandidateContains(t, result, "database.sqlite")
	assertNoCandidateContains(t, result, ".env")

	nodeModules := candidateByRule(result, "node-modules")
	if nodeModules.Risk != RiskCautious || nodeModules.CleanableByDefault {
		t.Fatalf("node_modules risk/default cleanability wrong: %#v", nodeModules)
	}
}

func TestUnknownLargeFilesAreDangerousUnlessPersonalOrProtected(t *testing.T) {
	root := t.TempDir()
	writeLargeFile(t, filepath.Join(root, "big.bin"), unknownLargeFileMin+1)
	writeLargeFile(t, filepath.Join(root, "archive.zip"), unknownLargeFileMin+1)
	writeLargeFile(t, filepath.Join(root, "dump.sql"), unknownLargeFileMin+1)

	result := scanForTest(t, root)
	candidate := candidateByRule(result, "unknown-large-file")
	if candidate.Path == "" {
		t.Fatalf("expected unknown large file candidate")
	}
	if candidate.Risk != RiskDangerous || candidate.CleanableByDefault {
		t.Fatalf("unknown large file should be dangerous report-only: %#v", candidate)
	}
	assertNoCandidateContains(t, result, "archive.zip")
	assertNoCandidateContains(t, result, "dump.sql")
}

func TestLocalAIModelsGetCautiousAdvice(t *testing.T) {
	root := t.TempDir()
	ollama := filepath.Join(root, ".ollama", "models", "blobs", "sha256-abc")
	lmStudio := filepath.Join(root, ".cache", "lm-studio", "models", "TheBloke", "zephyr", "model.gguf")
	writeAllocatedFile(t, ollama, unknownLargeFileMin+1)
	writeAllocatedFile(t, lmStudio, unknownLargeFileMin+1)

	result := scanForTest(t, root)

	ollamaCandidate := candidateByRule(result, "ollama-model-blob")
	if ollamaCandidate.Path == "" {
		t.Fatalf("expected Ollama candidate, got %#v", result.Candidates)
	}
	if ollamaCandidate.Category != CategoryLocalAIModels || ollamaCandidate.Risk != RiskCautious || ollamaCandidate.CleanableByDefault {
		t.Fatalf("wrong Ollama classification: %#v", ollamaCandidate)
	}
	if !strings.Contains(ollamaCandidate.Consequence, "Do not delete random blob files by hand") {
		t.Fatalf("Ollama JSON advice missing footgun warning: %#v", ollamaCandidate)
	}

	lmCandidate := candidateByRule(result, "lm-studio-model-file")
	if lmCandidate.Path == "" {
		t.Fatalf("expected LM Studio candidate, got %#v", result.Candidates)
	}
	if lmCandidate.Category != CategoryLocalAIModels || lmCandidate.Risk != RiskCautious || lmCandidate.CleanableByDefault {
		t.Fatalf("wrong LM Studio classification: %#v", lmCandidate)
	}
	if !strings.Contains(lmCandidate.Consequence, "removing a specific local model") {
		t.Fatalf("LM Studio JSON advice missing model-removal warning: %#v", lmCandidate)
	}
}

func TestScanSkipsGitAndDoesNotReportInsideIt(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git/objects/pack/large.pack"), string(make([]byte, unknownLargeFileMin+1)))

	result := scanForTest(t, root)
	if len(result.Candidates) != 0 {
		t.Fatalf("expected no candidates from .git, got %#v", result.Candidates)
	}
	if !slices.ContainsFunc(result.Skipped, func(skipped SkippedPath) bool {
		return filepath.Base(skipped.Path) == ".git"
	}) {
		t.Fatalf("expected .git skip, got %#v", result.Skipped)
	}
}

func TestScanRefusesRootAndHome(t *testing.T) {
	if _, err := NewScanner().Scan(ScanOptions{Roots: []string{"/"}, ExplicitPath: true}); err != nil {
		t.Fatalf("scan should return result with embedded error, got %v", err)
	}
	result, err := NewScanner().Scan(ScanOptions{Roots: []string{"/"}, ExplicitPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected refusal error for root")
	}
	root, err := prepareRoot("/", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if root != "/" {
		t.Fatalf("expected allowed root, got %s", root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	result, err = NewScanner().Scan(ScanOptions{Roots: []string{home}, ExplicitPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected refusal error for home")
	}
}

func TestScanResultShapeAndReadOnly(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "node", "dist", "app.js")
	writeFile(t, filepath.Join(root, "node", "package.json"), "{}")
	writeFile(t, file, "dist")
	before, err := snapshot(root)
	if err != nil {
		t.Fatal(err)
	}

	result := scanForTest(t, root)

	after, err := snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(before, after) {
		t.Fatalf("scan mutated fixture\nbefore: %#v\nafter: %#v", before, after)
	}
	if result.SchemaVersion != SchemaVersion {
		t.Fatalf("wrong schema: %s", result.SchemaVersion)
	}
	if result.Candidates == nil || result.Skipped == nil || result.Errors == nil {
		t.Fatalf("result collections should be empty arrays, not nil: %#v", result)
	}
	if len(result.ScanRoots) != 1 || result.GeneratedAt.IsZero() || len(result.ActiveRules) == 0 {
		t.Fatalf("result missing required shape: %#v", result)
	}
}

func TestGeneratedCandidatesIncludeDisplayOnlyMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	writeFile(t, filepath.Join(root, "dist", "app.js"), "dist")

	result := scanForTest(t, root)
	candidate := candidateByRule(result, "node-build-output")
	if candidate.RebuildCost == "" || candidate.ReclaimDurability == "" || candidate.PreferredCleanup == "" {
		t.Fatalf("candidate missing report metadata: %#v", candidate)
	}
	if strings.Contains(candidate.PreferredCleanup, "ratatoskr clean") || strings.Contains(candidate.PreferredCleanup, "rm ") {
		t.Fatalf("preferred cleanup should not contain deletion commands: %q", candidate.PreferredCleanup)
	}

	rule := ruleByName(result.ActiveRules, "node-build-output")
	if rule.RebuildCost == "" || rule.ReclaimDurability == "" || rule.PreferredCleanup == "" {
		t.Fatalf("rule missing report metadata: %#v", rule)
	}
}

func TestNodeAndComposerArtifactsIncludeProjectMetadata(t *testing.T) {
	root := t.TempDir()
	node := filepath.Join(root, "node")
	composer := filepath.Join(root, "php")
	writeFile(t, filepath.Join(node, "package.json"), "{}")
	for _, rel := range []string{"node_modules/pkg/index.js", ".next/cache.bin", ".nuxt/cache.bin", ".turbo/state", ".vite/cache", "coverage/index.html", "dist/app.js", "build/app.js"} {
		writeFile(t, filepath.Join(node, rel), "artifact")
	}
	writeFile(t, filepath.Join(composer, "composer.json"), "{}")
	writeFile(t, filepath.Join(composer, "vendor/pkg/file.php"), "dep")

	result := scanForTest(t, root)

	resolvedNode, err := filepath.EvalSymlinks(node)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"node-build-output", "node-modules"} {
		for _, candidate := range candidatesByRule(result, rule) {
			if candidate.ProjectRoot != resolvedNode || candidate.ProjectType != "node" || candidate.MarkerFile != "package.json" || candidate.ArtifactPath == "" {
				t.Fatalf("node candidate missing project metadata: %#v", candidate)
			}
		}
	}
	vendor := candidateByRule(result, "composer-vendor")
	resolvedComposer, err := filepath.EvalSymlinks(composer)
	if err != nil {
		t.Fatal(err)
	}
	if vendor.ProjectRoot != resolvedComposer || vendor.ProjectType != "composer" || vendor.MarkerFile != "composer.json" || vendor.ArtifactPath != "vendor" {
		t.Fatalf("composer candidate missing project metadata: %#v", vendor)
	}
}

func TestRustTargetIsCautiousProjectArtifact(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Cargo.toml"), "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n")
	writeFile(t, filepath.Join(root, "target", "debug", "demo"), "binary")

	result := scanForTest(t, root)
	candidate := candidateByRule(result, "rust-target")
	if candidate.Path == "" {
		t.Fatalf("expected Rust target candidate, got %#v", result.Candidates)
	}
	if candidate.Risk != RiskCautious || candidate.CleanableByDefault {
		t.Fatalf("Rust target should be cautious and not default-cleanable: %#v", candidate)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ProjectRoot != resolvedRoot || candidate.MarkerFile != "Cargo.toml" || candidate.ProjectType != "rust" || candidate.ArtifactPath != "target" {
		t.Fatalf("Rust target missing project metadata: %#v", candidate)
	}
	if candidate.RebuildCost == "" {
		t.Fatalf("Rust target missing rebuild cost: %#v", candidate)
	}
}

func TestTargetDirectoryWithoutCargoTomlIsNotRustArtifact(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}")
	writeFile(t, filepath.Join(root, "target", "debug", "not-rust"), "artifact")

	result := scanForTest(t, root)
	if hasCandidateRule(result, "rust-target") {
		t.Fatalf("target/ without Cargo.toml should not use Rust rule: %#v", result.Candidates)
	}
}

func TestExtraProjectArtifactsRequireTheirMarkerFiles(t *testing.T) {
	root := t.TempDir()
	swift := filepath.Join(root, "swift")
	terraform := filepath.Join(root, "terraform")
	serverless := filepath.Join(root, "serverless")
	gradle := filepath.Join(root, "gradle")
	maven := filepath.Join(root, "maven")
	loose := filepath.Join(root, "loose")

	writeFile(t, filepath.Join(swift, "Package.swift"), "// swift-tools-version: 5.9")
	writeFile(t, filepath.Join(swift, ".build", "debug", "app"), "artifact")
	writeFile(t, filepath.Join(terraform, "main.tf"), "terraform {}")
	writeFile(t, filepath.Join(terraform, ".terraform", "providers", "provider"), "artifact")
	writeFile(t, filepath.Join(serverless, "serverless.yml"), "service: demo")
	writeFile(t, filepath.Join(serverless, ".serverless", "demo.zip"), "artifact")
	writeFile(t, filepath.Join(gradle, "build.gradle"), "plugins {}")
	writeFile(t, filepath.Join(gradle, "build", "classes", "Demo.class"), "artifact")
	writeFile(t, filepath.Join(maven, "pom.xml"), "<project></project>")
	writeFile(t, filepath.Join(maven, "target", "demo.jar"), "artifact")
	writeFile(t, filepath.Join(loose, ".build", "thing"), "artifact")
	writeFile(t, filepath.Join(loose, ".terraform", "thing"), "artifact")
	writeFile(t, filepath.Join(loose, ".serverless", "thing"), "artifact")
	writeFile(t, filepath.Join(loose, "build", "thing"), "artifact")
	writeFile(t, filepath.Join(loose, "target", "thing"), "artifact")

	result := scanForTest(t, root)

	for _, test := range []struct {
		rule        string
		projectType string
		marker      string
		artifact    string
	}{
		{"swift-build", "swift", "Package.swift", ".build"},
		{"terraform-working-dir", "terraform", "*.tf", ".terraform"},
		{"serverless-build-state", "serverless", "serverless.yml", ".serverless"},
		{"gradle-build-output", "gradle", "build.gradle", "build"},
		{"maven-target", "maven", "pom.xml", "target"},
	} {
		candidate := candidateByRule(result, test.rule)
		if candidate.Path == "" {
			t.Fatalf("missing %s candidate: %#v", test.rule, result.Candidates)
		}
		if candidate.ProjectType != test.projectType || candidate.MarkerFile != test.marker || candidate.ArtifactPath != test.artifact {
			t.Fatalf("%s missing project metadata: %#v", test.rule, candidate)
		}
		if candidate.Risk != RiskCautious || candidate.CleanableByDefault {
			t.Fatalf("%s should be cautious and not default-cleanable: %#v", test.rule, candidate)
		}
	}
	for _, candidate := range result.Candidates {
		if strings.Contains(candidate.Path, loose) {
			t.Fatalf("loose artifact without marker should not be a candidate: %#v", candidate)
		}
	}
}

func TestExplicitApplicationCacheRootReportsCacheChildren(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Library", "Caches")
	writeFile(t, filepath.Join(root, "Homebrew", "downloads", "bottle.tar.gz"), "cache")
	writeFile(t, filepath.Join(root, "ms-playwright", "chromium", "browser"), "cache")

	result := scanForTest(t, root)

	for _, rule := range []string{"homebrew-cache", "playwright-cache"} {
		cacheRule := candidateByRule(result, rule)
		if cacheRule.Path == "" {
			t.Fatalf("expected %s candidate, got %#v", rule, result.Candidates)
		}
		if cacheRule.Risk != RiskCautious || cacheRule.CleanableByDefault {
			t.Fatalf("application caches should be cautious and not default-cleanable: %#v", cacheRule)
		}
	}
}

func TestExplicitPackageCacheRootReportsWholeRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".npm")
	writeFile(t, filepath.Join(root, "_cacache", "content-v2", "sha512", "a"), "cache")

	result := scanForTest(t, root)

	candidate := candidateByRule(result, "npm-cache")
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Path != resolvedRoot {
		t.Fatalf("expected package cache root candidate, got %#v", result.Candidates)
	}
	if candidate.Risk != RiskCautious || candidate.CleanableByDefault {
		t.Fatalf("package cache should be cautious and not default-cleanable: %#v", candidate)
	}
}

func TestNamedCacheRulesWinBeforeBroadFallbacks(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Library", "Caches")
	writeFile(t, filepath.Join(root, "Homebrew", "downloads", "bottle.tar.gz"), "cache")
	writeFile(t, filepath.Join(root, "Cypress", "13.0.0", "binary"), "cache")
	writeFile(t, filepath.Join(root, "ms-playwright", "chromium", "browser"), "cache")
	writeFile(t, filepath.Join(root, "puppeteer", "chrome", "browser"), "cache")
	writeFile(t, filepath.Join(root, "MysteryApp", "cache.bin"), "cache")

	result := scanForTest(t, root)

	for _, rule := range []string{"homebrew-cache", "cypress-cache", "playwright-cache", "puppeteer-cache", "explicit-application-cache"} {
		candidate := candidateByRule(result, rule)
		if candidate.Path == "" {
			t.Fatalf("expected %s candidate, got %#v", rule, result.Candidates)
		}
		if candidate.CleanableByDefault || candidate.Risk != RiskCautious {
			t.Fatalf("%s should stay cautious and non-cleanable: %#v", rule, candidate)
		}
	}
}

func TestKnownDangerousStateRulesBeatUnknownLargeFile(t *testing.T) {
	root := t.TempDir()
	dbnginData := filepath.Join(root, "Library", "Application Support", "com.tinyapp.DBngin", "Engines", "mysql", "data", "ibdata1")
	dbnginRecovery := filepath.Join(root, "Library", "Application Support", "com.tinyapp.DBngin", "Engines", "mysql", "recovery", "ibdata1")
	chromeModel := filepath.Join(root, "Library", "Application Support", "Google", "Chrome", "Default", "OptimizationGuidePredictionModels", "model.bin")
	cursorSnapshot := filepath.Join(root, "Library", "Application Support", "Cursor", "User", "History", "snapshots", "state.bin")
	for _, path := range []string{dbnginData, dbnginRecovery, chromeModel, cursorSnapshot} {
		writeAllocatedFile(t, path, unknownLargeFileMin+1)
	}

	result := scanForTest(t, root)

	for _, rule := range []string{"dbngin-database-state", "dbngin-engine-recovery-state", "chrome-on-device-model", "cursor-snapshot-state"} {
		candidate := candidateByRule(result, rule)
		if candidate.Path == "" {
			t.Fatalf("expected %s candidate, got %#v", rule, result.Candidates)
		}
		if candidate.CleanableByDefault || candidate.Risk == RiskSafe {
			t.Fatalf("%s must not be safe/default-cleanable: %#v", rule, candidate)
		}
	}
	if hasCandidateRule(result, "unknown-large-file") {
		t.Fatalf("known state paths should not fall back to unknown-large-file: %#v", result.Candidates)
	}
}

func TestUsesAllocatedSizeForSparseFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sparse.bin")
	writeLargeFile(t, path, unknownLargeFileMin+1)

	size, err := sizeOf(path)
	if err != nil {
		t.Fatal(err)
	}
	if size.apparent < unknownLargeFileMin {
		t.Fatalf("expected apparent size to keep original file length: %#v", size)
	}
	if size.allocated >= size.apparent {
		t.Skipf("filesystem did not create a sparse file: %#v", size)
	}
}

func TestStreamingReportSkipsDuplicatePhysicalFiles(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	aliasDir := filepath.Join(root, "alias")
	writeLargeFile(t, filepath.Join(realDir, "large.bin"), unknownLargeFileMin+1)
	if err := os.Symlink(realDir, aliasDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	var out bytes.Buffer
	err := NewScanner().StreamJSON(&out, StreamOptions{
		ScanOptions: ScanOptions{
			Roots:           []string{realDir, aliasDir},
			ExplicitPath:    true,
			AllowSystemRoot: false,
			Now:             time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var result ScanResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("stream emitted invalid JSON: %v\n%s", err, out.String())
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected one candidate after duplicate filtering, got %#v", result.Candidates)
	}
	if !slices.ContainsFunc(result.Skipped, func(skipped SkippedPath) bool {
		return strings.Contains(skipped.Reason, "duplicate")
	}) {
		t.Fatalf("expected duplicate skip, got %#v", result.Skipped)
	}
}

func TestStreamingReportIncludesDisplayOnlyMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Cargo.toml"), "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n")
	writeFile(t, filepath.Join(root, "target", "debug", "demo"), "binary")

	var out bytes.Buffer
	err := NewScanner().StreamJSON(&out, StreamOptions{
		ScanOptions: ScanOptions{
			Roots:        []string{root},
			ExplicitPath: true,
			Now:          time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var result ScanResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("stream emitted invalid JSON: %v\n%s", err, out.String())
	}
	candidate := candidateByRule(result, "rust-target")
	if candidate.RebuildCost == "" || candidate.ReclaimDurability == "" || candidate.PreferredCleanup == "" {
		t.Fatalf("streamed candidate missing report metadata: %#v", candidate)
	}
	if candidate.ProjectType != "rust" || candidate.MarkerFile != "Cargo.toml" {
		t.Fatalf("streamed candidate missing project metadata: %#v", candidate)
	}
}

func scanForTest(t *testing.T, root string) ScanResult {
	t.Helper()
	result, err := NewScanner().Scan(ScanOptions{
		Roots:        []string{root},
		ExplicitPath: true,
		Now:          time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected scan errors: %#v", result.Errors)
	}
	return result
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

func hasCandidateRule(result ScanResult, rule string) bool {
	return slices.ContainsFunc(result.Candidates, func(candidate Candidate) bool {
		return candidate.Rule == rule
	})
}

func candidateByRule(result ScanResult, rule string) Candidate {
	for _, candidate := range result.Candidates {
		if candidate.Rule == rule {
			return candidate
		}
	}
	return Candidate{}
}

func candidatesByRule(result ScanResult, rule string) []Candidate {
	candidates := []Candidate{}
	for _, candidate := range result.Candidates {
		if candidate.Rule == rule {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func assertNoCandidateContains(t *testing.T, result ScanResult, fragment string) {
	t.Helper()
	for _, candidate := range result.Candidates {
		if strings.Contains(candidate.Path, fragment) {
			t.Fatalf("candidate should not contain %q: %#v", fragment, candidate)
		}
	}
}

func writeLargeFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
}

func writeAllocatedFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("x"), 1024*1024)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var written int64
	for written < size {
		chunk := data
		remaining := size - written
		if remaining < int64(len(chunk)) {
			chunk = chunk[:remaining]
		}
		n, err := file.Write(chunk)
		if err != nil {
			t.Fatal(err)
		}
		written += int64(n)
	}
}

func snapshot(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		paths = append(paths, path+"|"+info.Mode().String())
		return nil
	})
	slices.Sort(paths)
	return paths, err
}
