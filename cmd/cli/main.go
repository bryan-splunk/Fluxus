package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryan-splunk/fluxus/cmd/server"
	"github.com/bryan-splunk/fluxus/engine"
	"github.com/spf13/cobra"
)

const defaultRulesDir = "rules"
const defaultTargetVersion = "" // empty = latest (all rules)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fluxus",
	Short: "FLUXUS — Splunk OTel Collector config upgrade tool",
	Long: `FLUXUS scans Splunk OpenTelemetry Collector YAML configuration files for
breaking changes and applies upgrades safely using a tick-per-version
State/Effect pattern.

Basic workflow:
  1. fluxus assess   — dry-run scan, generates PreAssessment.md
  2. Review PreAssessment.md and decide which changes to apply
  3. fluxus apply    — apply approved changes, generates OperationalAssessment.md`,
}

// ---- assess ---------------------------------------------------------------

var assessFlags struct {
	targetVersion   string
	includeComments bool
	includeGuidance bool
	rulesDir        string
	outputDir       string
}

var assessCmd = &cobra.Command{
	Use:   "assess [config-files-or-dirs-or-globs...]",
	Short: "Dry-run scan — generate PreAssessment.md without modifying files",
	Long: `Scan one or more Splunk OTel Collector config files and generate PreAssessment.md.

Arguments accept:
  - Exact file paths          fluxus assess agent.yaml gateway.yaml
  - A directory               fluxus assess /etc/otel/configs/
  - Shell glob patterns       fluxus assess configs/*.yaml
  - Any combination           fluxus assess agent.yaml /etc/otel/ "k8s/*.yaml"

Directories are scanned non-recursively for *.yaml and *.yml files.
Use a glob pattern for nested directories: "configs/**/*.yaml".`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAssess,
}

func init() {
	assessCmd.Flags().StringVar(&assessFlags.targetVersion, "target-version", defaultTargetVersion,
		"Target collector version (default: latest — all rules)")
	assessCmd.Flags().BoolVar(&assessFlags.includeComments, "include-comments", false,
		"Also scan commented-out config sections (rules auto-match commented components; "+
			"see the 'Commented-Out Config' section of PreAssessment.md)")
	assessCmd.Flags().BoolVar(&assessFlags.includeGuidance, "include-guidance", false,
		"Include extended operational guidance in the report (Action: prose, accuracy checks, "+
			"transition-period notes, and actions outside the config file)")
	assessCmd.Flags().StringVar(&assessFlags.rulesDir, "rules-dir", defaultRulesDir,
		"Path to rules directory")
	assessCmd.Flags().StringVar(&assessFlags.outputDir, "output-dir", ".",
		"Directory for output files (created if it does not exist)")
	rootCmd.AddCommand(assessCmd)
}

func runAssess(_ *cobra.Command, args []string) error {
	state, err := loadConfigs(args)
	if err != nil {
		return err
	}

	rules, err := engine.LoadRulesTree(assessFlags.rulesDir)
	if err != nil {
		return fmt.Errorf("loading rules: %w", err)
	}

	result, err := engine.DryRun(state, rules, assessFlags.targetVersion, assessFlags.includeComments)
	if err != nil {
		return fmt.Errorf("dry run: %w", err)
	}

	report, err := engine.RenderPreAssessment(result, assessFlags.targetVersion, args, assessFlags.includeGuidance)
	if err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	if err := ensureDir(assessFlags.outputDir); err != nil {
		return err
	}
	outPath := filepath.Join(assessFlags.outputDir, "PreAssessment.md")
	if err := os.WriteFile(outPath, []byte(report), 0644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	// Print summary to stdout
	applicable := engine.AllEffects(result.ApplicableTicks)
	future := engine.AllEffects(result.FutureTicks)
	fmt.Printf("✅ Pre-assessment complete → %s\n\n", outPath)
	fmt.Printf("   %d applicable change(s) found", len(applicable))
	if assessFlags.targetVersion != "" {
		fmt.Printf(" (target: %s)", assessFlags.targetVersion)
	}
	fmt.Println()
	if len(future) > 0 {
		fmt.Printf("   %d future change(s) identified beyond target version\n", len(future))
	}
	if len(result.Conflicts) > 0 {
		fmt.Printf("   ⚠ %d conflict(s) detected — review before applying\n", len(result.Conflicts))
	}
	fmt.Printf("\nTo apply: fluxus apply --select all")
	if assessFlags.targetVersion != "" {
		fmt.Printf(" --target-version %s", assessFlags.targetVersion)
	}
	fmt.Printf(" %s\n", strings.Join(args, " "))
	return nil
}

// ---- apply ----------------------------------------------------------------

var applyFlags struct {
	targetVersion   string
	includeComments bool
	includeGuidance bool
	rulesDir        string
	select_         string
	outputDir       string
}

var applyCmd = &cobra.Command{
	Use:   "apply [config-files-or-dirs-or-globs...]",
	Short: "Apply approved changes from a pre-assessment",
	Long: `Apply upgrade changes to one or more Splunk OTel Collector config files.

Arguments accept:
  - Exact file paths          fluxus apply agent.yaml gateway.yaml
  - A directory               fluxus apply /etc/otel/configs/
  - Shell glob patterns       fluxus apply configs/*.yaml
  - Any combination           fluxus apply agent.yaml /etc/otel/ "k8s/*.yaml"

Directories are scanned non-recursively for *.yaml and *.yml files.
Updated files are written to --output-dir; original files are never modified.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runApply,
}

func init() {
	applyCmd.Flags().StringVar(&applyFlags.targetVersion, "target-version", defaultTargetVersion,
		"Target collector version")
	applyCmd.Flags().BoolVar(&applyFlags.includeComments, "include-comments", false,
		"Process commented-out sections too: apply key renames inside comments and inject "+
			"each rule's upgrade-note annotation above the matching commented component. "+
			"Processed findings are listed under OperationalAssessment's "+
			"\"Commented-Out Config Processed\" section")
	applyCmd.Flags().BoolVar(&applyFlags.includeGuidance, "include-guidance", false,
		"Include extended operational guidance in the report (Action: prose, accuracy checks, "+
			"transition-period notes, and actions outside the config file)")
	applyCmd.Flags().StringVar(&applyFlags.rulesDir, "rules-dir", defaultRulesDir,
		"Path to rules directory")
	applyCmd.Flags().StringVar(&applyFlags.select_, "select", "all",
		`Changes to apply: "all", "p1" (breaking), "p2" (degrading), "p3" (advisory), or comma-separated rule IDs`)
	applyCmd.Flags().StringVar(&applyFlags.outputDir, "output-dir", ".",
		"Directory for upgraded config files and assessment report (created if it does not exist)")
	rootCmd.AddCommand(applyCmd)
}

func runApply(_ *cobra.Command, args []string) error {
	state, err := loadConfigs(args)
	if err != nil {
		return err
	}

	rules, err := engine.LoadRulesTree(applyFlags.rulesDir)
	if err != nil {
		return fmt.Errorf("loading rules: %w", err)
	}

	approved := strings.Split(applyFlags.select_, ",")
	for i, approvalID := range approved {
		approved[i] = strings.TrimSpace(approvalID)
	}

	result, err := engine.Apply(state, rules, engine.ApplyOptions{
		TargetVersion:   applyFlags.targetVersion,
		IncludeComments: applyFlags.includeComments,
		ApprovedIDs:     approved,
	})
	if err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	if err := ensureDir(applyFlags.outputDir); err != nil {
		return err
	}

	// Write all config files to the output directory.
	// Changed files get the migrated content; unchanged files are copied as-is.
	// Original files are never modified.
	writtenCount := 0
	for filePath, newContent := range result.UpdatedFiles {
		outPath := filepath.Join(applyFlags.outputDir, filepath.Base(filePath))
		if err := os.WriteFile(outPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		if newContent != state.Raw[filePath] {
			fmt.Printf("✅ Updated: %s\n", outPath)
			writtenCount++
		} else {
			fmt.Printf("📄 Copied:  %s (no changes)\n", outPath)
		}
	}
	if writtenCount == 0 {
		fmt.Println("ℹ No changes were applicable — all files copied unchanged.")
	}

	// Topology validation on the new state.
	newState, err := engine.NewState(result.UpdatedFiles)
	if err != nil {
		fmt.Printf("⚠ Could not parse updated configs for topology check: %v\n", err)
	}
	var topologyIssues []engine.ValidationIssue
	if newState != nil {
		topologyIssues = engine.ValidateTopology(newState)
	}

	// Render operational assessment.
	operationalReport, err := engine.RenderOperationalAssessment(engine.OperationalReportData{
		TargetVersion:   applyFlags.targetVersion,
		AppliedEffects:  result.AppliedEffects,
		GuidedEffects:   result.GuidedEffects,
		CommentEffects:  result.CommentEffects,
		UpdatedFiles:    result.UpdatedFiles,
		Warnings:        result.Warnings,
		TopologyIssues:  topologyIssues,
		IncludeGuidance: applyFlags.includeGuidance,
	})
	if err != nil {
		return fmt.Errorf("rendering operational assessment: %w", err)
	}

	if err := ensureDir(applyFlags.outputDir); err != nil {
		return err
	}
	operationalPath := filepath.Join(applyFlags.outputDir, "OperationalAssessment.md")
	if err := os.WriteFile(operationalPath, []byte(operationalReport), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", operationalPath, err)
	}

	fmt.Printf("\n📋 %d change(s) applied → %s\n", len(result.AppliedEffects), operationalPath)
	if len(result.CommentEffects) > 0 {
		fmt.Printf("📝 %d commented-out config finding(s) processed (renames + injected upgrade notes)\n", len(result.CommentEffects))
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("⚠ %d change(s) require manual review (see OperationalAssessment.md)\n", len(result.Warnings))
	}
	if len(topologyIssues) > 0 {
		fmt.Printf("⚠ %d topology issue(s) found — see OperationalAssessment.md\n", len(topologyIssues))
	}
	return nil
}

// ---- server ---------------------------------------------------------------

func init() {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the web UI server",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			rulesDirectory, _ := cmd.Flags().GetString("rules-dir")
			return server.Start(port, rulesDirectory)
		},
	}
	serverCmd.Flags().Int("port", 8080, "HTTP port")
	serverCmd.Flags().String("rules-dir", defaultRulesDir, "Path to rules directory")
	rootCmd.AddCommand(serverCmd)
}

// ---- helpers --------------------------------------------------------------

// loadConfigs reads config files from the provided paths and returns a State.
//
// Each element of paths may be:
//   - An exact file path            → loaded directly.
//   - A directory path              → all *.yaml / *.yml files in the directory are loaded
//     (non-recursive; use a glob pattern for nested dirs).
//   - A glob pattern (contains * or ?) → expanded via filepath.Glob; all matching files loaded.
//
// Duplicate resolved paths are silently deduplicated.
func loadConfigs(paths []string) (*engine.State, error) {
	seen := make(map[string]bool)
	raw := make(map[string]string)

	addFile := func(path string) error {
		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolving path %q: %w", path, err)
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		data, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("reading %q: %w", path, err)
		}
		raw[abs] = string(data)
		return nil
	}

	for _, path := range paths {
		info, statErr := os.Stat(path)
		switch {
		case statErr == nil && info.IsDir():
			// Directory: load all *.yaml and *.yml files directly inside it (non-recursive).
			for _, pattern := range []string{"*.yaml", "*.yml"} {
				matches, err := filepath.Glob(filepath.Join(path, pattern))
				if err != nil {
					return nil, fmt.Errorf("glob %q: %w", pattern, err)
				}
				for _, match := range matches {
					if err := addFile(match); err != nil {
						return nil, err
					}
				}
			}

		case strings.ContainsAny(path, "*?["):
			// Glob pattern: expand and load all matches.
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, fmt.Errorf("invalid glob %q: %w", path, err)
			}
			if len(matches) == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "⚠ warning: glob %q matched no files\n", path)
			}
			for _, match := range matches {
				// Skip directories that a broad glob might hit.
				if fileInfo, err := os.Stat(match); err == nil && fileInfo.IsDir() {
					continue
				}
				if err := addFile(match); err != nil {
					return nil, err
				}
			}

		default:
			// Exact file path.
			if err := addFile(path); err != nil {
				return nil, err
			}
		}
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("no config files found for the given paths/patterns")
	}
	return engine.NewState(raw)
}

// ensureDir creates dir (and any parents) if it does not already exist.
func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory %q: %w", dir, err)
	}
	return nil
}
