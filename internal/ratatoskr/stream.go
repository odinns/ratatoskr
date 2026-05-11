package ratatoskr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type StreamOptions struct {
	ScanOptions
	Progress io.Writer
}

type streamReport struct {
	out            io.Writer
	encoder        *json.Encoder
	progress       io.Writer
	rules          []Rule
	totals         Totals
	skipped        []SkippedPath
	errors         []ScanError
	candidateRoots []string
	seen           map[fileIdentity]string
	seenDirs       map[fileIdentity]string
	pathsSeen      int64
	candidatesSeen int64
	firstCandidate bool
	lastProgress   time.Time
}

func (s Scanner) StreamJSON(out io.Writer, options StreamOptions) error {
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	roots := options.Roots
	if len(roots) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		roots = []string{cwd}
	}

	report := &streamReport{
		out:            out,
		encoder:        json.NewEncoder(out),
		progress:       options.Progress,
		rules:          s.rules,
		skipped:        []SkippedPath{},
		errors:         []ScanError{},
		candidateRoots: []string{},
		seen:           map[fileIdentity]string{},
		seenDirs:       map[fileIdentity]string{},
		firstCandidate: true,
		lastProgress:   time.Now(),
	}

	scanRoots := []string{}
	seenRoots := map[string]bool{}
	for _, root := range roots {
		cleanRoot, err := prepareRoot(root, options.ExplicitPath, options.AllowSystemRoot)
		if err != nil {
			report.addError(ScanError{Path: root, Error: err.Error()})
			continue
		}
		if seenRoots[cleanRoot] {
			report.addSkipped(SkippedPath{Path: cleanRoot, Reason: "duplicate scan root"})
			continue
		}
		seenRoots[cleanRoot] = true
		scanRoots = append(scanRoots, cleanRoot)
	}

	if err := report.begin(now, scanRoots); err != nil {
		return err
	}
	for _, root := range scanRoots {
		s.streamRoot(root, rootKind(root), options.ExplicitPath, report)
	}
	return report.end()
}

func (s Scanner) streamRoot(root string, kind scanRootKind, explicit bool, report *streamReport) {
	if explicit {
		switch kind {
		case scanRootApplicationCache:
			streamChildrenCandidate("", root, ruleByName(s.rules, "explicit-application-cache"), report)
		case scanRootPackageCache:
			streamPathCandidate(root, ruleByName(s.rules, "explicit-package-cache"), report)
		}
	}

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		report.progressPath(path)
		if err != nil {
			if isPermissionError(err) {
				report.addSkipped(SkippedPath{Path: path, Reason: "permission denied"})
			} else {
				report.addError(ScanError{Path: path, Error: err.Error()})
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && report.isDuplicateDir(path, d) {
				return filepath.SkipDir
			}
			if path != root && report.isUnderCandidate(path) {
				return filepath.SkipDir
			}
			if d.Name() == ".git" {
				report.addSkipped(SkippedPath{Path: path, Reason: "repository metadata is protected"})
				return filepath.SkipDir
			}
			if isProtectedPersonalPath(path) && path != root {
				report.addSkipped(SkippedPath{Path: path, Reason: "personal folder is report-only"})
				return filepath.SkipDir
			}
			if project := detectProject(path); project.any() {
				s.streamProjectCandidates(path, project, explicit, report)
			}
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			if isPermissionError(infoErr) {
				report.addSkipped(SkippedPath{Path: path, Reason: "permission denied"})
			} else {
				report.addError(ScanError{Path: path, Error: infoErr.Error()})
			}
			return nil
		}
		size := fileSizeOf(info)
		if info.Mode().IsRegular() && size.allocated >= unknownLargeFileMin && !report.isUnderCandidate(path) {
			if rule, ok := s.specialLargeFileRule(path); ok {
				report.addCandidate(candidateFromRule(path, size, rule), info)
				return nil
			}
			if !isKnownPersonalOrProtectedFile(path) {
				report.addCandidate(candidateFromRule(path, size, ruleByName(s.rules, "unknown-large-file")), info)
			}
		}
		return nil
	})
}

func (s Scanner) streamProjectCandidates(project string, kind projectKind, explicit bool, report *streamReport) {
	if kind.laravel {
		streamGlobCandidates(project, "storage/logs/*.log", ruleByName(s.rules, "laravel-storage-logs"), report)
		streamGlobCandidates(project, "bootstrap/cache/*.php", ruleByName(s.rules, "laravel-bootstrap-cache"), report)
		streamChildrenCandidate(project, "storage/framework/views", ruleByName(s.rules, "laravel-compiled-views"), report)
		streamChildrenCandidate(project, "storage/framework/cache", ruleByName(s.rules, "laravel-framework-cache"), report)
	}
	if kind.node {
		rule := ruleByName(s.rules, "node-build-output")
		for _, rel := range []string{".next", "dist", "build", ".turbo", ".vite", "coverage"} {
			streamPathCandidate(filepath.Join(project, rel), rule, report)
		}
		streamPathCandidate(filepath.Join(project, "node_modules"), ruleByName(s.rules, "node-modules"), report)
	}
	if kind.composer {
		streamPathCandidate(filepath.Join(project, "vendor"), ruleByName(s.rules, "composer-vendor"), report)
	}
	if explicit {
		rule := ruleByName(s.rules, "explicit-package-cache")
		for _, rel := range []string{"npm-cache", "pnpm-store", "yarn-cache", "composer-cache"} {
			streamPathCandidate(filepath.Join(project, rel), rule, report)
		}
	}
}

func streamGlobCandidates(project, pattern string, rule Rule, report *streamReport) {
	matches, _ := filepath.Glob(filepath.Join(project, filepath.FromSlash(pattern)))
	for _, match := range matches {
		streamPathCandidate(match, rule, report)
	}
}

func streamChildrenCandidate(project, rel string, rule Rule, report *streamReport) {
	dir := filepath.Join(project, filepath.FromSlash(rel))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			report.addError(ScanError{Path: dir, Error: err.Error()})
		}
		return
	}
	for _, entry := range entries {
		streamPathCandidate(filepath.Join(dir, entry.Name()), rule, report)
	}
}

func streamPathCandidate(path string, rule Rule, report *streamReport) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		report.addSkipped(SkippedPath{Path: path, Reason: "symlinks are report-only in Phase 1"})
		return
	}
	size, err := sizeOf(path)
	if err != nil {
		if isPermissionError(err) {
			report.addSkipped(SkippedPath{Path: path, Reason: "permission denied while sizing candidate"})
		} else {
			report.addError(ScanError{Path: path, Error: err.Error()})
		}
		return
	}
	report.addCandidate(candidateFromRule(path, size, rule), info)
	if info.IsDir() {
		report.candidateRoots = append(report.candidateRoots, path)
	}
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "permission denied") || strings.Contains(text, "operation not permitted")
}

func (r *streamReport) begin(generatedAt time.Time, roots []string) error {
	if _, err := fmt.Fprintf(r.out, "{\n  \"schema_version\": %q,\n  \"generated_at\": %q,\n  \"scan_roots\": ", SchemaVersion, generatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := r.encoder.Encode(roots); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.out, ",  \"active_rules\": "); err != nil {
		return err
	}
	if err := r.encoder.Encode(r.rules); err != nil {
		return err
	}
	_, err := fmt.Fprint(r.out, ",  \"candidates\": [\n")
	return err
}

func (r *streamReport) addCandidate(candidate Candidate, info os.FileInfo) {
	if id, ok := identityOf(info); ok {
		if firstPath, exists := r.seen[id]; exists {
			r.addSkipped(SkippedPath{Path: candidate.Path, Reason: "duplicate of " + firstPath})
			return
		}
		r.seen[id] = candidate.Path
	}
	if !r.firstCandidate {
		fmt.Fprint(r.out, ",\n")
	}
	r.firstCandidate = false
	r.encoder.Encode(candidate)
	r.candidatesSeen++
	r.totals.CandidateCount++
	r.totals.TotalBytes += candidate.SizeBytes
	switch candidate.Risk {
	case RiskSafe:
		r.totals.SafeBytes += candidate.SizeBytes
	case RiskCautious:
		r.totals.CautiousBytes += candidate.SizeBytes
	case RiskDangerous:
		r.totals.DangerousBytes += candidate.SizeBytes
	}
	if candidate.CleanableByDefault {
		r.totals.CleanableByDefault++
		r.totals.CleanableDefaultSize += candidate.SizeBytes
	}
	if flusher, ok := r.out.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
}

func (r *streamReport) addSkipped(skipped SkippedPath) {
	r.skipped = append(r.skipped, skipped)
}

func (r *streamReport) addError(scanErr ScanError) {
	r.errors = append(r.errors, scanErr)
}

func (r *streamReport) progressPath(path string) {
	r.pathsSeen++
	if r.progress == nil || time.Since(r.lastProgress) < time.Second {
		return
	}
	r.lastProgress = time.Now()
	fmt.Fprintf(r.progress, "\rscanned %d paths, found %d candidates: %s", r.pathsSeen, r.candidatesSeen, path)
}

func (r *streamReport) isUnderCandidate(path string) bool {
	for _, candidateRoot := range r.candidateRoots {
		if path == candidateRoot || strings.HasPrefix(path, candidateRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (r *streamReport) isDuplicateDir(path string, d fs.DirEntry) bool {
	info, err := d.Info()
	if err != nil {
		return false
	}
	id, ok := identityOf(info)
	if !ok {
		return false
	}
	if firstPath, exists := r.seenDirs[id]; exists {
		r.addSkipped(SkippedPath{Path: path, Reason: "duplicate directory of " + firstPath})
		return true
	}
	r.seenDirs[id] = path
	return false
}

func (r *streamReport) end() error {
	if _, err := fmt.Fprint(r.out, "\n  ],\n  \"skipped\": "); err != nil {
		return err
	}
	if err := r.encoder.Encode(r.skipped); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.out, ",  \"errors\": "); err != nil {
		return err
	}
	if err := r.encoder.Encode(r.errors); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.out, ",  \"totals\": "); err != nil {
		return err
	}
	if err := r.encoder.Encode(r.totals); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.out, "}\n"); err != nil {
		return err
	}
	if r.progress != nil {
		fmt.Fprintf(r.progress, "\rscanned %d paths, found %d candidates. done.\n", r.pathsSeen, r.candidatesSeen)
	}
	return nil
}
