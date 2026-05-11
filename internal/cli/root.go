package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/odinn/ratatoskr/internal/ratatoskr"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	return NewRootCommandWithOutput(os.Stdout)
}

func NewRootCommandWithOutput(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ratatoskr",
		Short:        "A conservative local disk scout.",
		SilenceUsage: true,
	}
	cmd.AddCommand(newScanCommand(out), newReportCommand(out), newSummaryCommand(out), newRulesCommand(out))
	return cmd
}

func newScanCommand(out io.Writer) *cobra.Command {
	var path string
	var format string
	var allowSystemRoot bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a project tree without changing anything.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := scanPath(path, allowSystemRoot)
			if err != nil {
				return err
			}
			return writeScan(out, result, format)
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "path to scan")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&allowSystemRoot, "allow-system-root", false, "allow an explicit read-only scan from /")
	return cmd
}

func newReportCommand(out io.Writer) *cobra.Command {
	var path string
	var format string
	var allowSystemRoot bool
	var stream bool
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Run a fresh scan and render a report.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if stream {
				if format != "json" {
					return fmt.Errorf("streaming reports only support json")
				}
				buffered := bufio.NewWriter(out)
				err := ratatoskr.NewScanner().StreamJSON(buffered, ratatoskr.StreamOptions{
					ScanOptions: ratatoskr.ScanOptions{
						Roots:           rootsForPath(path),
						ExplicitPath:    path != "",
						AllowSystemRoot: allowSystemRoot,
					},
					Progress: os.Stderr,
				})
				if flushErr := buffered.Flush(); flushErr != nil && err == nil {
					err = flushErr
				}
				return err
			}
			result, err := scanPath(path, allowSystemRoot)
			if err != nil {
				return err
			}
			return writeScan(out, result, format)
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "path to scan")
	cmd.Flags().StringVar(&format, "format", "json", "output format: json")
	cmd.Flags().BoolVar(&allowSystemRoot, "allow-system-root", false, "allow an explicit read-only scan from /")
	cmd.Flags().BoolVar(&stream, "stream", false, "stream JSON candidates and print progress to stderr")
	return cmd
}

func newRulesCommand(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List active built-in rules.",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, rule := range ratatoskr.BuiltInRules() {
				fmt.Fprintf(out, "%s\n", rule.Name)
				fmt.Fprintf(out, "  source: %s\n", rule.Source)
				fmt.Fprintf(out, "  category: %s\n", rule.Category)
				fmt.Fprintf(out, "  risk: %s\n", rule.Risk)
				fmt.Fprintf(out, "  matches: %s\n", strings.Join(rule.PathPatterns, ", "))
				fmt.Fprintf(out, "  applies when: %s\n", strings.Join(rule.AppliesWhen, ", "))
				fmt.Fprintf(out, "  exclusions: %s\n", strings.Join(rule.Exclusions, ", "))
				fmt.Fprintf(out, "  reason: %s\n", rule.Reason)
				fmt.Fprintf(out, "  consequence: %s\n", rule.Consequence)
				fmt.Fprintf(out, "  cleanable by default: %t\n\n", rule.CleanableByDefault)
			}
			return nil
		},
	}
	return cmd
}

func newSummaryCommand(out io.Writer) *cobra.Command {
	var file string
	var limit int
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize an existing scan report.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			handle, err := os.Open(file)
			if err != nil {
				return err
			}
			defer handle.Close()

			summary, err := ratatoskr.ReadSummary(handle)
			if err != nil {
				return err
			}
			writeSummary(out, summary, limit)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "scan report JSON file")
	cmd.Flags().IntVar(&limit, "limit", 20, "number of candidates per section")
	return cmd
}

func scanPath(path string, allowSystemRoot bool) (ratatoskr.ScanResult, error) {
	options := ratatoskr.ScanOptions{Roots: rootsForPath(path), ExplicitPath: path != "", AllowSystemRoot: allowSystemRoot}
	return ratatoskr.NewScanner().Scan(options)
}

func rootsForPath(path string) []string {
	if path == "" {
		return nil
	}
	return []string{path}
}

func writeScan(out io.Writer, result ratatoskr.ScanResult, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "text", "":
		writeTextScan(out, result)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeTextScan(out io.Writer, result ratatoskr.ScanResult) {
	fmt.Fprintf(out, "Ratatoskr found %s of disk waste.\n", humanBytes(result.Totals.TotalBytes))
	fmt.Fprintf(out, "%s is safe to clean later.\n", humanBytes(result.Totals.SafeBytes))
	fmt.Fprintf(out, "%s requires explicit selection.\n", humanBytes(result.Totals.CautiousBytes))
	fmt.Fprintf(out, "%s is report-only.\n\n", humanBytes(result.Totals.DangerousBytes))
	fmt.Fprintln(out, "Reports contain local file paths. Treat them as private.")
	for _, candidate := range result.Candidates {
		fmt.Fprintf(out, "\n%s\n  %s, %s, %s\n  rule: %s\n  %s\n", candidate.Path, humanBytes(candidate.SizeBytes), candidate.Category, candidate.Risk, candidate.Rule, candidate.Reason)
	}
	if len(result.Skipped) > 0 {
		fmt.Fprintln(out, "\nSkipped:")
		for _, skipped := range result.Skipped {
			fmt.Fprintf(out, "  %s: %s\n", skipped.Path, skipped.Reason)
		}
	}
	if len(result.Errors) > 0 {
		fmt.Fprintln(out, "\nErrors:")
		for _, scanErr := range result.Errors {
			fmt.Fprintf(out, "  %s: %s\n", scanErr.Path, scanErr.Error)
		}
	}
}

func writeSummary(out io.Writer, summary ratatoskr.Summary, limit int) {
	result := summary.Result
	fmt.Fprintln(out, "Ratatoskr scan summary")
	fmt.Fprintf(out, "Candidates: %d\n", result.Totals.CandidateCount)
	fmt.Fprintf(out, "Total: %s\n", humanBytes(result.Totals.TotalBytes))
	fmt.Fprintf(out, "Safe: %s\n", humanBytes(result.Totals.SafeBytes))
	fmt.Fprintf(out, "Cautious: %s\n", humanBytes(result.Totals.CautiousBytes))
	fmt.Fprintf(out, "Dangerous: %s\n", humanBytes(result.Totals.DangerousBytes))
	fmt.Fprintf(out, "Skipped: %d\n", len(result.Skipped))
	fmt.Fprintf(out, "Errors: %d\n\n", len(result.Errors))
	fmt.Fprintln(out, "Reports contain local file paths. Treat them as private.")

	writeCandidateSection(out, "Top candidates", summary.TopCandidates(limit))
	writeCandidateSection(out, "Top cautious", summary.TopCandidatesByRisk(ratatoskr.RiskCautious, limit))
	writeCandidateSection(out, "Top safe", summary.TopCandidatesByRisk(ratatoskr.RiskSafe, limit))
}

func writeCandidateSection(out io.Writer, title string, candidates []ratatoskr.Candidate) {
	fmt.Fprintf(out, "\n%s\n", title)
	if len(candidates) == 0 {
		fmt.Fprintln(out, "  none")
		return
	}
	for _, candidate := range candidates {
		fmt.Fprintf(out, "  %8s  %-9s  %-28s  %s\n", humanBytes(candidate.SizeBytes), candidate.Risk, candidate.Rule, candidate.Path)
	}
}

func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KiB", "MiB", "GiB", "TiB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/unit)
}
