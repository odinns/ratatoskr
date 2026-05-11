package ratatoskr

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	SchemaVersion       = "ratatoskr.scan.v1"
	unknownLargeFileMin = 100 * 1024 * 1024
)

type ScanOptions struct {
	Roots           []string
	ExplicitPath    bool
	AllowSystemRoot bool
	Now             time.Time
}

type Scanner struct {
	rules []Rule
}

type scanState struct {
	result        *ScanResult
	seen          map[fileIdentity]string
	seenDirs      map[fileIdentity]string
	candidateDirs []string
}

func NewScanner() Scanner {
	return Scanner{rules: BuiltInRules()}
}

func (s Scanner) Scan(options ScanOptions) (ScanResult, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	roots := options.Roots
	if len(roots) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return ScanResult{}, err
		}
		roots = []string{cwd}
	}

	result := ScanResult{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   now,
		ActiveRules:   s.rules,
		ScanRoots:     []string{},
		Candidates:    []Candidate{},
		Skipped:       []SkippedPath{},
		Errors:        []ScanError{},
	}

	state := scanState{
		result:   &result,
		seen:     map[fileIdentity]string{},
		seenDirs: map[fileIdentity]string{},
	}

	seenRoots := map[string]bool{}
	for _, root := range roots {
		cleanRoot, err := prepareRoot(root, options.ExplicitPath, options.AllowSystemRoot)
		if err != nil {
			result.Errors = append(result.Errors, ScanError{Path: root, Error: err.Error()})
			continue
		}
		if seenRoots[cleanRoot] {
			result.Skipped = append(result.Skipped, SkippedPath{Path: cleanRoot, Reason: "duplicate scan root"})
			continue
		}
		seenRoots[cleanRoot] = true
		result.ScanRoots = append(result.ScanRoots, cleanRoot)
		s.scanRoot(cleanRoot, rootKind(cleanRoot), options.ExplicitPath, &state)
	}

	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].Path < result.Candidates[j].Path
	})
	result.Totals = calculateTotals(result.Candidates)

	return result, nil
}

func prepareRoot(root string, explicit bool, allowSystemRoot bool) (string, error) {
	expanded, err := expandHome(root)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	if resolved == string(filepath.Separator) && !allowSystemRoot {
		return "", errors.New("refusing to scan filesystem root")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeResolved, _ := filepath.EvalSymlinks(home)
		if resolved == homeResolved {
			return "", errors.New("refusing to scan home directory as a root")
		}
	}
	if !explicit {
		if cwd, err := os.Getwd(); err == nil {
			cwdResolved, _ := filepath.EvalSymlinks(cwd)
			if resolved != cwdResolved {
				return "", errors.New("implicit scans only use the current directory")
			}
		}
	}
	return resolved, nil
}

func (s Scanner) scanRoot(root string, rootKind scanRootKind, explicit bool, state *scanState) {
	if explicit {
		s.addExplicitRootCandidates(root, rootKind, state)
	}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if isPermissionError(err) {
				state.addSkipped(SkippedPath{Path: path, Reason: "permission denied"})
			} else {
				state.addError(ScanError{Path: path, Error: err.Error()})
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && state.isDuplicateDir(path, d) {
				return filepath.SkipDir
			}
			if path != root && state.isUnderCandidate(path) {
				return filepath.SkipDir
			}
			if name == ".git" {
				state.addSkipped(SkippedPath{Path: path, Reason: "repository metadata is protected"})
				return filepath.SkipDir
			}
			if isProtectedPersonalPath(path) && path != root {
				state.addSkipped(SkippedPath{Path: path, Reason: "personal folder is report-only"})
				return filepath.SkipDir
			}
			if kind := detectProject(path); kind.any() {
				s.addProjectCandidates(path, kind, explicit, state)
			}
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			if isPermissionError(infoErr) {
				state.addSkipped(SkippedPath{Path: path, Reason: "permission denied"})
			} else {
				state.addError(ScanError{Path: path, Error: infoErr.Error()})
			}
			return nil
		}
		size := fileSizeOf(info)
		if info.Mode().IsRegular() && size.allocated >= unknownLargeFileMin && !state.isUnderCandidate(path) {
			if rule, ok := s.specialLargeFileRule(path); ok {
				state.addCandidate(candidateFromRule(path, size, rule), info)
				return nil
			}
			if !isKnownPersonalOrProtectedFile(path) {
				state.addCandidate(candidateFromRule(path, size, ruleByName(s.rules, "unknown-large-file")), info)
			}
		}
		return nil
	})
}

func (s Scanner) addProjectCandidates(project string, kind projectKind, explicit bool, state *scanState) {
	if kind.laravel {
		addGlobCandidates(project, "storage/logs/*.log", ruleByName(s.rules, "laravel-storage-logs"), state)
		addGlobCandidates(project, "bootstrap/cache/*.php", ruleByName(s.rules, "laravel-bootstrap-cache"), state)
		addChildrenCandidate(project, "storage/framework/views", ruleByName(s.rules, "laravel-compiled-views"), state)
		addChildrenCandidate(project, "storage/framework/cache", ruleByName(s.rules, "laravel-framework-cache"), state)
	}
	if kind.node {
		rule := ruleByName(s.rules, "node-build-output")
		for _, rel := range []string{".next", ".nuxt", "dist", "build", ".turbo", ".vite", "coverage"} {
			addProjectPathCandidate(project, rel, "node", "package.json", rule, state)
		}
		addProjectPathCandidate(project, "node_modules", "node", "package.json", ruleByName(s.rules, "node-modules"), state)
	}
	if kind.composer {
		addProjectPathCandidate(project, "vendor", "composer", "composer.json", ruleByName(s.rules, "composer-vendor"), state)
	}
	if kind.rust {
		addProjectPathCandidate(project, "target", "rust", "Cargo.toml", ruleByName(s.rules, "rust-target"), state)
	}
	if kind.swift {
		addProjectPathCandidate(project, ".build", "swift", "Package.swift", ruleByName(s.rules, "swift-build"), state)
	}
	if kind.terraform {
		addProjectPathCandidate(project, ".terraform", "terraform", "*.tf", ruleByName(s.rules, "terraform-working-dir"), state)
	}
	if kind.serverless {
		addProjectPathCandidate(project, ".serverless", "serverless", "serverless.yml", ruleByName(s.rules, "serverless-build-state"), state)
	}
	if kind.gradle {
		addProjectPathCandidate(project, "build", "gradle", "build.gradle", ruleByName(s.rules, "gradle-build-output"), state)
	}
	if kind.maven {
		addProjectPathCandidate(project, "target", "maven", "pom.xml", ruleByName(s.rules, "maven-target"), state)
	}
	if explicit {
		for _, rel := range []string{"npm-cache", "pnpm-store", "yarn-cache", "composer-cache"} {
			addPathCandidate(filepath.Join(project, rel), s.ruleForCachePath(filepath.Join(project, rel), scanRootPackageCache), state)
		}
	}
}

func (s Scanner) addExplicitRootCandidates(root string, kind scanRootKind, state *scanState) {
	switch kind {
	case scanRootApplicationCache:
		addCacheChildrenCandidate(root, kind, s, state)
	case scanRootPackageCache:
		addPathCandidate(root, s.ruleForCachePath(root, kind), state)
	}
}

func (s Scanner) specialLargeFileRule(path string) (Rule, bool) {
	normalized := filepath.ToSlash(path)
	lower := strings.ToLower(normalized)
	if strings.Contains(normalized, "/Library/Application Support/com.tinyapp.DBngin/Engines/") {
		if strings.Contains(normalized, "/data/") {
			return ruleByName(s.rules, "dbngin-database-state"), true
		}
		if strings.Contains(normalized, "/broken/") || strings.Contains(normalized, "/recovery/") {
			return ruleByName(s.rules, "dbngin-engine-recovery-state"), true
		}
	}
	if strings.Contains(normalized, "/Library/Application Support/Google/Chrome/") && strings.Contains(normalized, "/OptimizationGuidePredictionModels/") {
		return ruleByName(s.rules, "chrome-on-device-model"), true
	}
	if strings.Contains(lower, "/library/application support/cursor/") && strings.Contains(lower, "/snapshots/") {
		return ruleByName(s.rules, "cursor-snapshot-state"), true
	}
	if strings.Contains(normalized, "/.ollama/models/blobs/") {
		return ruleByName(s.rules, "ollama-model-blob"), true
	}
	if strings.Contains(normalized, "/.cache/lm-studio/models/") && strings.HasSuffix(strings.ToLower(normalized), ".gguf") {
		return ruleByName(s.rules, "lm-studio-model-file"), true
	}
	if strings.Contains(normalized, "/Library/Application Support/LM Studio/models/") && strings.HasSuffix(strings.ToLower(normalized), ".gguf") {
		return ruleByName(s.rules, "lm-studio-model-file"), true
	}
	return Rule{}, false
}

func (s Scanner) ruleForCachePath(path string, kind scanRootKind) Rule {
	normalized := filepath.ToSlash(path)
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.Contains(normalized, "/Library/Caches/Homebrew") || base == "homebrew":
		return ruleByName(s.rules, "homebrew-cache")
	case strings.Contains(normalized, "/Library/Caches/Cypress") || strings.Contains(normalized, "/.cache/Cypress") || base == "cypress":
		return ruleByName(s.rules, "cypress-cache")
	case strings.Contains(normalized, "/Library/Caches/ms-playwright") || strings.Contains(normalized, "/.cache/ms-playwright") || base == "ms-playwright":
		return ruleByName(s.rules, "playwright-cache")
	case strings.Contains(normalized, "/Library/Caches/puppeteer") || strings.Contains(normalized, "/.cache/puppeteer") || base == "puppeteer":
		return ruleByName(s.rules, "puppeteer-cache")
	case base == ".npm" || base == "npm-cache":
		return ruleByName(s.rules, "npm-cache")
	case strings.HasSuffix(normalized, "/.composer/cache") || base == "composer-cache":
		return ruleByName(s.rules, "composer-cache")
	case kind == scanRootPackageCache:
		return ruleByName(s.rules, "explicit-package-cache")
	default:
		return ruleByName(s.rules, "explicit-application-cache")
	}
}

func addGlobCandidates(project, pattern string, rule Rule, state *scanState) {
	matches, _ := filepath.Glob(filepath.Join(project, filepath.FromSlash(pattern)))
	for _, match := range matches {
		addPathCandidate(match, rule, state)
	}
}

func addChildrenCandidate(project, rel string, rule Rule, state *scanState) {
	dir := filepath.Join(project, filepath.FromSlash(rel))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			state.addError(ScanError{Path: dir, Error: err.Error()})
		}
		return
	}
	for _, entry := range entries {
		addPathCandidate(filepath.Join(dir, entry.Name()), rule, state)
	}
}

func addCacheChildrenCandidate(root string, kind scanRootKind, scanner Scanner, state *scanState) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			state.addError(ScanError{Path: root, Error: err.Error()})
		}
		return
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		addPathCandidate(path, scanner.ruleForCachePath(path, kind), state)
	}
}

func addPathCandidate(path string, rule Rule, state *scanState) {
	addCandidateWithProjectMetadata(path, rule, projectMetadata{}, state)
}

type projectMetadata struct {
	root     string
	marker   string
	kind     string
	artifact string
}

func addProjectPathCandidate(project string, rel string, projectType string, marker string, rule Rule, state *scanState) {
	addCandidateWithProjectMetadata(filepath.Join(project, filepath.FromSlash(rel)), rule, projectMetadata{
		root:     project,
		marker:   marker,
		kind:     projectType,
		artifact: filepath.ToSlash(rel),
	}, state)
}

func addCandidateWithProjectMetadata(path string, rule Rule, metadata projectMetadata, state *scanState) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		state.addSkipped(SkippedPath{Path: path, Reason: "symlinks are report-only in Phase 1"})
		return
	}
	size, err := sizeOf(path)
	if err != nil {
		if isPermissionError(err) {
			state.addSkipped(SkippedPath{Path: path, Reason: "permission denied while sizing candidate"})
		} else {
			state.addError(ScanError{Path: path, Error: err.Error()})
		}
		return
	}
	candidate := candidateFromRule(path, size, rule)
	candidate.ProjectRoot = metadata.root
	candidate.MarkerFile = metadata.marker
	candidate.ProjectType = metadata.kind
	candidate.ArtifactPath = metadata.artifact
	state.addCandidate(candidate, info)
	if info.IsDir() {
		state.candidateDirs = append(state.candidateDirs, path)
	}
}

type projectKind struct {
	laravel    bool
	node       bool
	composer   bool
	rust       bool
	swift      bool
	terraform  bool
	serverless bool
	gradle     bool
	maven      bool
}

func (p projectKind) any() bool {
	return p.laravel || p.node || p.composer || p.rust || p.swift || p.terraform || p.serverless || p.gradle || p.maven
}

func detectProject(dir string) projectKind {
	composerPath := filepath.Join(dir, "composer.json")
	composer := fileExists(composerPath)
	return projectKind{
		laravel:    fileExists(filepath.Join(dir, "artisan")) && composerRequiresLaravel(composerPath),
		node:       fileExists(filepath.Join(dir, "package.json")) || anyFileExists(dir, "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lockb"),
		composer:   composer,
		rust:       fileExists(filepath.Join(dir, "Cargo.toml")),
		swift:      fileExists(filepath.Join(dir, "Package.swift")),
		terraform:  hasFileWithSuffix(dir, ".tf"),
		serverless: anyFileExists(dir, "serverless.yml", "serverless.yaml"),
		gradle:     anyFileExists(dir, "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"),
		maven:      fileExists(filepath.Join(dir, "pom.xml")),
	}
}

func hasFileWithSuffix(dir string, suffix string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			return true
		}
	}
	return false
}

type scanRootKind int

const (
	scanRootRegular scanRootKind = iota
	scanRootApplicationCache
	scanRootPackageCache
)

func rootKind(root string) scanRootKind {
	home, err := os.UserHomeDir()
	if err != nil {
		return scanRootRegular
	}
	if filepath.Base(root) == "Caches" && filepath.Base(filepath.Dir(root)) == "Library" {
		return scanRootApplicationCache
	}
	if filepath.Base(root) == ".cache" {
		return scanRootApplicationCache
	}
	switch filepath.Base(root) {
	case ".npm", ".pnpm-store", ".yarn":
		return scanRootPackageCache
	case "cache":
		if filepath.Base(filepath.Dir(root)) == ".composer" {
			return scanRootPackageCache
		}
	}
	for _, cacheRoot := range []string{
		filepath.Join(home, "Library", "Caches"),
		filepath.Join(home, ".cache"),
	} {
		if root == cacheRoot {
			return scanRootApplicationCache
		}
	}
	for _, cacheRoot := range []string{
		filepath.Join(home, ".npm"),
		filepath.Join(home, ".pnpm-store"),
		filepath.Join(home, ".yarn"),
		filepath.Join(home, ".composer", "cache"),
	} {
		if root == cacheRoot {
			return scanRootPackageCache
		}
	}
	return scanRootRegular
}

func composerRequiresLaravel(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var parsed map[string]any
	if json.Unmarshal(data, &parsed) != nil {
		return strings.Contains(string(data), "laravel/framework")
	}
	for _, section := range []string{"require", "require-dev"} {
		deps, ok := parsed[section].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := deps["laravel/framework"]; ok {
			return true
		}
	}
	return false
}

func anyFileExists(dir string, names ...string) bool {
	for _, name := range names {
		if fileExists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func calculateTotals(candidates []Candidate) Totals {
	var totals Totals
	totals.CandidateCount = len(candidates)
	for _, candidate := range candidates {
		totals.TotalBytes += candidate.SizeBytes
		switch candidate.Risk {
		case RiskSafe:
			totals.SafeBytes += candidate.SizeBytes
		case RiskCautious:
			totals.CautiousBytes += candidate.SizeBytes
		case RiskDangerous:
			totals.DangerousBytes += candidate.SizeBytes
		}
		if candidate.CleanableByDefault {
			totals.CleanableByDefault++
			totals.CleanableDefaultSize += candidate.SizeBytes
		}
	}
	return totals
}

func ruleByName(rules []Rule, name string) Rule {
	for _, rule := range rules {
		if rule.Name == name {
			return rule
		}
	}
	return Rule{Name: name}
}

func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func isProtectedPersonalPath(path string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, name := range []string{"Desktop", "Documents", "Downloads", "Pictures", "Movies", ".ssh", ".gnupg"} {
		if path == filepath.Join(home, name) {
			return true
		}
	}
	return false
}

func isKnownPersonalOrProtectedFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if base == ".env" || strings.HasSuffix(base, ".sqlite") || strings.HasSuffix(base, ".db") || strings.HasSuffix(base, ".sql") {
		return true
	}
	for _, ext := range []string{".zip", ".tar", ".gz", ".tgz", ".rar", ".7z", ".jpg", ".jpeg", ".png", ".heic", ".mov", ".mp4", ".pdf", ".doc", ".docx", ".xls", ".xlsx"} {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	return isProtectedPersonalPath(filepath.Dir(path))
}

func candidateFromRule(path string, size pathSize, rule Rule) Candidate {
	return Candidate{
		Path:               path,
		SizeBytes:          size.allocated,
		ApparentSizeBytes:  size.apparent,
		Category:           rule.Category,
		Risk:               rule.Risk,
		Rule:               rule.Name,
		Reason:             rule.Reason,
		Consequence:        rule.Consequence,
		CleanableByDefault: rule.CleanableByDefault,
		RebuildCost:        rule.RebuildCost,
		ReclaimDurability:  rule.ReclaimDurability,
		PreferredCleanup:   rule.PreferredCleanup,
	}
}

func (s *scanState) addCandidate(candidate Candidate, info os.FileInfo) {
	if id, ok := identityOf(info); ok {
		if firstPath, exists := s.seen[id]; exists {
			s.addSkipped(SkippedPath{Path: candidate.Path, Reason: "duplicate of " + firstPath})
			return
		}
		s.seen[id] = candidate.Path
	}
	s.result.Candidates = append(s.result.Candidates, candidate)
}

func (s *scanState) addSkipped(skipped SkippedPath) {
	s.result.Skipped = append(s.result.Skipped, skipped)
}

func (s *scanState) addError(scanErr ScanError) {
	s.result.Errors = append(s.result.Errors, scanErr)
}

func (s *scanState) isUnderCandidate(path string) bool {
	for _, candidateDir := range s.candidateDirs {
		if path == candidateDir || strings.HasPrefix(path, candidateDir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (s *scanState) isDuplicateDir(path string, d fs.DirEntry) bool {
	info, err := d.Info()
	if err != nil {
		return false
	}
	id, ok := identityOf(info)
	if !ok {
		return false
	}
	if firstPath, exists := s.seenDirs[id]; exists {
		s.addSkipped(SkippedPath{Path: path, Reason: "duplicate directory of " + firstPath})
		return true
	}
	s.seenDirs[id] = path
	return false
}
