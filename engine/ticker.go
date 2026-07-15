package engine

import (
	"fmt"
	"sort"
	"strings"
)

// DryRunResult holds the full output of a dry-run tick chain.
type DryRunResult struct {
	// ApplicableTicks are ticks at or before the target version.
	ApplicableTicks []TickResult
	// FutureTicks are ticks beyond the target version (report-only).
	FutureTicks []TickResult
	// Conflicts are same-key effect pairs detected across ticks.
	Conflicts []Conflict
}

// DryRun runs the full tick chain in read-only mode and returns all detected
// effects without writing any files. Effects beyond targetVersion are collected
// as future changes.
//
// If targetVersion is empty, all rules are treated as applicable.
func DryRun(state *State, rules []*Rule, targetVersion string, includeComments bool) (*DryRunResult, error) {
	applicable, future, err := FilterByVersion(rules, targetVersion)
	if err != nil {
		return nil, err
	}

	applicableTicks, err := runTicks(state, applicable, false, includeComments)
	if err != nil {
		return nil, err
	}

	futureTicks, err := runTicks(state, future, true, includeComments)
	if err != nil {
		return nil, err
	}

	allTicks := append(applicableTicks, futureTicks...)
	conflicts := DetectConflicts(allTicks)

	return &DryRunResult{
		ApplicableTicks: applicableTicks,
		FutureTicks:     futureTicks,
		Conflicts:       conflicts,
	}, nil
}

// ApplyOptions controls which effects are written during the apply pass.
type ApplyOptions struct {
	TargetVersion   string
	IncludeComments bool
	// ApprovedIDs is the set of rule IDs the user approved.
	// Special values: "all", "p1", "p2", "p3" (matched against Rule.Category), or exact rule IDs.
	ApprovedIDs []string
}

// ApplyResult holds the output of the apply pass.
type ApplyResult struct {
	// UpdatedFiles maps filename to the new raw YAML content (all files, modified or not).
	UpdatedFiles map[string]string
	// AppliedEffects are ACTIVE-config effects where the file content was actually
	// changed. Comment-block effects are tracked separately in CommentEffects.
	AppliedEffects []Effect
	// GuidedEffects are ACTIVE-config effects that fired but were not auto-applied
	// because their strategy is "guided" or "inform_only" — they require manual
	// action.
	GuidedEffects []Effect
	// CommentEffects are effects that matched commented-out config (IsComment).
	// They are applied via text edits (renames + injected upgrade-note
	// annotations) regardless of strategy, since editing commented text never
	// changes the live config. Reported separately so commented-out findings are
	// visible without being conflated with active changes.
	CommentEffects []Effect
	// Warnings are non-fatal issues encountered during application.
	Warnings []string
}

// Apply runs the tick chain in write mode, applying only the effects approved
// by the user. Returns the modified file contents and a list of applied effects.
func Apply(state *State, rules []*Rule, opts ApplyOptions) (*ApplyResult, error) {
	applicable, _, err := FilterByVersion(rules, opts.TargetVersion)
	if err != nil {
		return nil, err
	}

	approved := buildApprovalSet(opts.ApprovedIDs)
	ticks, err := runTicks(state, applicable, false, opts.IncludeComments)
	if err != nil {
		return nil, err
	}

	// Collect all approved effects in tick order.
	var toApply []Effect
	for _, tick := range ticks {
		for _, effect := range tick.Effects {
			if isApproved(effect, approved) {
				toApply = append(toApply, effect)
			}
		}
	}

	// Apply effects to the raw file content.
	updatedFiles := make(map[string]string, len(state.Raw))
	for fileName, content := range state.Raw {
		updatedFiles[fileName] = content
	}

	var applied []Effect
	var guided []Effect
	var commentEffects []Effect
	var warnings []string

	for _, effect := range toApply {
		// Comment-block effects: always run through ApplyMigration (which routes
		// to the text-based comment path) regardless of strategy. Editing
		// commented-out config is always safe, so even guided/inform_only rules
		// carry their renames and upgrade-note annotations into the template.
		// They are reported separately and never counted as active changes.
		if effect.IsComment {
			current := updatedFiles[effect.FilePath]
			updated, migrationWarnings, err := ApplyMigration(current, effect)
			warnings = append(warnings, migrationWarnings...)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			if updated != current {
				updatedFiles[effect.FilePath] = updated
			}
			commentEffects = append(commentEffects, effect)
			continue
		}

		strategy := strings.ToLower(effect.Rule.Migration.Strategy)
		if strategy == "guided" || strategy == "inform_only" {
			// These rules are detected and reported but never auto-applied.
			guided = append(guided, effect)
			continue
		}

		current := updatedFiles[effect.FilePath]
		updated, migrationWarnings, err := ApplyMigration(current, effect)
		warnings = append(warnings, migrationWarnings...)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		// Only count as applied if the content actually changed.
		if updated != current {
			updatedFiles[effect.FilePath] = updated
			applied = append(applied, effect)
		}
	}

	return &ApplyResult{
		UpdatedFiles:   updatedFiles,
		AppliedEffects: applied,
		GuidedEffects:  guided,
		CommentEffects: commentEffects,
		Warnings:       warnings,
	}, nil
}

// runTicks executes the tick chain for a slice of rules, grouped by version.
// Returns one TickResult per version that had at least one matching effect.
// Within each version, rules are sorted by their Order field (ascending) before
// evaluation so authors can control execution sequence within a tick.
func runTicks(state *State, rules []*Rule, isFuture bool, includeComments bool) ([]TickResult, error) {
	grouped := GroupByVersion(rules)
	versions, err := SortedVersions(grouped)
	if err != nil {
		return nil, fmt.Errorf("sorting rule versions: %w", err)
	}

	var results []TickResult
	for _, version := range versions {
		versionRules := grouped[version]
		sortRulesByOrder(versionRules)
		effects := Scan(state, versionRules, version, isFuture, includeComments)
		if len(effects) > 0 {
			results = append(results, TickResult{
				Version: version,
				Effects: effects,
			})
		}
	}
	return results, nil
}

// sortRulesByOrder sorts a slice of rules by their Order field in ascending
// order, using a stable sort so rules with equal Order values retain their
// original alphabetical file-load sequence.
func sortRulesByOrder(rules []*Rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Order < rules[j].Order
	})
}

// buildApprovalSet normalises the user's approval list into a set for O(1) lookup.
// Special tokens "all", "p1", "p2", "p3" are preserved as-is and matched against Rule.Category.
func buildApprovalSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[strings.ToLower(id)] = true
	}
	return set
}

// isApproved reports whether an effect should be applied given the approval set.
func isApproved(effect Effect, approved map[string]bool) bool {
	if approved["all"] {
		return true
	}
	if approved[string(effect.Rule.Category)] {
		return true
	}
	if approved[strings.ToLower(effect.Rule.ID)] {
		return true
	}
	return false
}

// AllEffects flattens a slice of TickResults into a single Effect slice.
func AllEffects(ticks []TickResult) []Effect {
	var all []Effect
	for _, tickResult := range ticks {
		all = append(all, tickResult.Effects...)
	}
	return all
}

// SortEffects sorts effects: P1 Breaking → P2 Degrading → P3 Advisory, then by FiredAtTick.
func SortEffects(effects []Effect) {
	order := map[Category]int{
		CategoryP1: 0,
		CategoryP2: 1,
		CategoryP3: 2,
	}
	sort.SliceStable(effects, func(i, j int) bool {
		categoryOrderI := order[effects[i].Rule.Category]
		categoryOrderJ := order[effects[j].Rule.Category]
		if categoryOrderI != categoryOrderJ {
			return categoryOrderI < categoryOrderJ
		}
		return effects[i].FiredAtTick < effects[j].FiredAtTick
	})
}
