package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// LoadRules reads all *.yaml files from rulesDir and parses them into Rule structs.
// Returns an error if any file cannot be read or parsed.
func LoadRules(rulesDir string) ([]*Rule, error) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read rules dir %q: %w", rulesDir, err)
	}

	var rules []*Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(rulesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read rule file %q: %w", path, err)
		}
		var rule Rule
		if err := yaml.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("cannot parse rule file %q: %w", path, err)
		}
		if rule.ID == "" {
			return nil, fmt.Errorf("rule file %q is missing required field 'id'", path)
		}
		rules = append(rules, &rule)
	}
	return rules, nil
}

// LoadRulesTree loads all rules from rootDir (flat *.yaml files) AND from every
// immediate subdirectory under rootDir. Each subdirectory is treated as a rule
// phase namespace (e.g. "security", "pipeline"). Rules loaded from subdirectories
// have their Phase field set to the subdirectory name when the rule's own Phase
// field is empty.
//
// This allows the rules directory to be structured as:
//
//	rules/              — config rules (P1-xx, P2-xx, P3-xx)
//	rules/security/     — security / credential checks (SEC-P1-xx)
//	rules/pipeline/     — pipeline topology checks (PIPE-P1-xx, future)
//	rules/post/         — post-upgrade checks (POST-P1-xx, future)
func LoadRulesTree(rootDir string) ([]*Rule, error) {
	// Load root-level config rules.
	rules, err := LoadRules(rootDir)
	if err != nil {
		return nil, err
	}
	// Stamp root-level rules as PhaseConfig when the rule didn't set it.
	for _, rule := range rules {
		if rule.Phase == "" {
			rule.Phase = PhaseConfig
		}
	}

	// Walk immediate subdirectories.
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read rules root %q: %w", rootDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(rootDir, entry.Name())
		subRules, err := LoadRules(subDir)
		if err != nil {
			return nil, err
		}
		// Stamp Phase from the directory name when the rule didn't set it.
		dirPhase := Phase(entry.Name())
		for _, rule := range subRules {
			if rule.Phase == "" {
				rule.Phase = dirPhase
			}
		}
		rules = append(rules, subRules...)
	}
	return rules, nil
}

// FilterByVersion partitions rules into two slices:
//   - applicable: rules with Introduced <= targetVersion (should be applied)
//   - future:     rules with Introduced > targetVersion  (report-only)
//
// If targetVersion is empty, all rules are considered applicable.
func FilterByVersion(rules []*Rule, targetVersion string) (applicable []*Rule, future []*Rule, err error) {
	if targetVersion == "" {
		return rules, nil, nil
	}

	target, err := semver.NewVersion(targetVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid target version %q: %w", targetVersion, err)
	}

	for _, rule := range rules {
		if rule.Introduced == "" {
			applicable = append(applicable, rule)
			continue
		}
		introduced, err := semver.NewVersion(rule.Introduced)
		if err != nil {
			return nil, nil, fmt.Errorf("rule %s has invalid introduced version %q: %w", rule.ID, rule.Introduced, err)
		}
		if introduced.Compare(target) <= 0 {
			applicable = append(applicable, rule)
		} else {
			future = append(future, rule)
		}
	}
	return applicable, future, nil
}

// GroupByVersion returns a map of version string → rules introduced at that version.
// This is used by the ticker to process one version increment at a time.
func GroupByVersion(rules []*Rule) map[string][]*Rule {
	grouped := make(map[string][]*Rule)
	for _, rule := range rules {
		version := rule.Introduced
		if version == "" {
			version = "0.0.0"
		}
		grouped[version] = append(grouped[version], rule)
	}
	return grouped
}

// SortedVersions returns the version strings from a grouped map in ascending semver order.
func SortedVersions(grouped map[string][]*Rule) ([]string, error) {
	type versionEntry struct {
		parsed   *semver.Version
		original string
	}

	entries := make([]versionEntry, 0, len(grouped))
	for key := range grouped {
		v, err := semver.NewVersion(key)
		if err != nil {
			return nil, fmt.Errorf("cannot parse version %q: %w", key, err)
		}
		entries = append(entries, versionEntry{parsed: v, original: key})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].parsed.LessThan(entries[j].parsed)
	})

	sorted := make([]string, len(entries))
	for i, entry := range entries {
		sorted[i] = entry.original
	}
	return sorted, nil
}
