package engine

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scan evaluates all rules against the provided state and returns the effects
// that fired. If includeComments is true, commented-out YAML blocks are also
// scanned and matching effects are tagged with IsComment=true.
//
// Each rule's Logic field controls how its look_for selectors are combined:
//   - "or"  (default) — first matching selector fires the rule
//   - "and"           — ALL selectors must match for the rule to fire
func Scan(state *State, rules []*Rule, tick string, isFuture bool, includeComments bool) []Effect {
	var effects []Effect

	for filePath, node := range state.Files {
		rawContent := state.Raw[filePath]

		// Comment path (separate from the active path): when --include-comments is
		// set, detect commented-out component blocks once per file via the
		// line/regex detector and pre-build a parseable YAML node for each. Rules
		// are then evaluated against these nodes by auto-deriving from their own
		// look_for selectors (see commentScanSelectors), so a rule does not need an
		// explicit in_comments selector to be checked against comments. This is the
		// failure mode the old ExtractCommentedBlocks branch could not handle:
		// child-level comments (e.g. "# iis:" inside a live receivers:) and blocks
		// with interleaved prose.
		var commentNodes []commentScanTarget
		if includeComments {
			commentNodes = buildCommentScanTargets(rawContent)
		}

		for _, rule := range rules {
			logic := strings.ToLower(rule.Logic)
			if logic == "" {
				logic = "or"
			}

			// Active path: evaluate the rule's NON-comment selectors against the
			// parsed YAML tree and raw text. in_comments selectors are excluded here
			// (they are evaluated only on the comment path) so a rule whose only
			// selector is in_comments never fires against active config. raw_pattern
			// selectors stay in the active group so logic: "and" can combine both.
			var normalSelectors []LookFor
			for _, lookFor := range rule.LookFor {
				if !lookFor.InComments {
					normalSelectors = append(normalSelectors, lookFor)
				}
			}
			if len(normalSelectors) > 0 && matchesSelectors(logic, normalSelectors, node, rawContent) {
				effects = append(effects, Effect{
					Rule:        rule,
					FiredAtTick: tick,
					IsFuture:    isFuture,
					IsComment:   false,
					FilePath:    filePath,
					MatchedPath: firstMatchedPath(normalSelectors, node, rawContent),
				})
			}

			// Comment path: auto-derive comment matching from the rule's own
			// selectors (both normal and in_comments) unless disabled via
			// scan_comments: false.
			if !includeComments || len(commentNodes) == 0 {
				continue
			}
			if rule.ScanComments != nil && !*rule.ScanComments {
				continue
			}
			commentSelectors := commentScanSelectors(rule.LookFor)
			if len(commentSelectors) == 0 {
				continue
			}
			for _, commentTarget := range commentNodes {
				if matchesSelectors(logic, commentSelectors, commentTarget.node, "") {
					effects = append(effects, Effect{
						Rule:        rule,
						FiredAtTick: tick,
						IsFuture:    isFuture,
						IsComment:   true,
						FilePath:    filePath,
						MatchedPath: firstMatchedPath(commentSelectors, commentTarget.node, ""),
					})
					break
				}
			}
		}
	}

	return effects
}

// commentScanTarget pairs a detected commented-out component with its parseable
// YAML node (wrapped under its inferred section) for selector evaluation.
type commentScanTarget struct {
	comp CommentedComponent
	node *yaml.Node
}

// buildCommentScanTargets detects commented-out component blocks in raw content
// and pre-builds a YAML node for each so rule selectors can be evaluated against
// them. Components whose body fails to parse or whose section is unknown are
// skipped (their path cannot be resolved reliably).
func buildCommentScanTargets(rawContent string) []commentScanTarget {
	var targets []commentScanTarget
	for _, component := range DetectCommentedComponents(rawContent) {
		parsedNode := commentComponentNode(component)
		if parsedNode != nil {
			targets = append(targets, commentScanTarget{comp: component, node: parsedNode})
		}
	}
	return targets
}

// commentComponentNode reconstructs a parseable YAML document for a single
// commented-out component, wrapped under its inferred section so that path
// selectors like "$.receivers.iis" resolve. Returns nil when the section is
// unknown or the body does not parse.
func commentComponentNode(component CommentedComponent) *yaml.Node {
	if component.Body == "" || component.Section == "" {
		return nil
	}
	var builder strings.Builder
	builder.WriteString(component.Section)
	builder.WriteString(":\n")
	for _, line := range strings.Split(component.Body, "\n") {
		if line == "" {
			builder.WriteByte('\n')
			continue
		}
		builder.WriteString("  ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(builder.String()), &node); err != nil {
		return nil
	}
	return &node
}

// commentScanSelectors returns the subset of a rule's selectors that are
// meaningful when evaluated against an isolated commented-out component block.
// raw_pattern selectors (used by security rules — deferred) and absent selectors
// (meaningless against a single extracted component) are excluded.
func commentScanSelectors(lookFors []LookFor) []LookFor {
	var results []LookFor
	for _, lookFor := range lookFors {
		if lookFor.RawPattern != "" || lookFor.Match == MatchAbsent {
			continue
		}
		results = append(results, lookFor)
	}
	return results
}

// matchesSelectors evaluates look_for selectors against a YAML node using the
// supplied logic ("or" or "and"). rawContent is the raw file text; it is used
// when a selector has RawPattern set (empty string disables raw matching).
func matchesSelectors(logic string, selectors []LookFor, node *yaml.Node, rawContent string) bool {
	if logic == "and" {
		// ALL selectors must match.
		for _, lookFor := range selectors {
			var hit bool
			if lookFor.RawPattern != "" {
				compiledPattern, err := regexp.Compile(lookFor.RawPattern)
				hit = err == nil && compiledPattern.MatchString(rawContent)
			} else {
				found, _ := evaluatePath(lookFor.Path, node)
				hit = checkMatch(lookFor, node, found)
			}
			if !hit {
				return false
			}
		}
		return len(selectors) > 0
	}
	// OR: first match wins.
	for _, lookFor := range selectors {
		if lookFor.RawPattern != "" {
			compiledPattern, err := regexp.Compile(lookFor.RawPattern)
			if err == nil && compiledPattern.MatchString(rawContent) {
				return true
			}
			continue
		}
		found, _ := evaluatePath(lookFor.Path, node)
		if checkMatch(lookFor, node, found) {
			return true
		}
	}
	return false
}

// firstMatchedPath returns the matched path for the first selector that resolves
// to a found node or matches a raw_pattern. Used to populate Effect.MatchedPath.
func firstMatchedPath(selectors []LookFor, node *yaml.Node, rawContent string) string {
	for _, lookFor := range selectors {
		if lookFor.RawPattern != "" {
			compiledPattern, err := regexp.Compile(lookFor.RawPattern)
			if err == nil && compiledPattern.MatchString(rawContent) {
				pattern := lookFor.RawPattern
				if len(pattern) > 40 {
					pattern = pattern[:40] + "…"
				}
				return "<raw_pattern: " + pattern + ">"
			}
			continue
		}
		if found, matchedPath := evaluatePath(lookFor.Path, node); found {
			return matchedPath
		}
	}
	return ""
}

// checkMatch evaluates the match condition of a LookFor selector.
// found reports whether evaluatePath located the node.
func checkMatch(lookFor LookFor, root *yaml.Node, found bool) bool {
	switch lookFor.Match {
	case MatchExists:
		return found

	case MatchAbsent:
		return !found // fires when the path is NOT present in the config

	case MatchValue:
		if !found {
			return false
		}
		_, resolvedNode := resolveNode(lookFor.Path, root)
		if resolvedNode == nil {
			return false
		}
		return resolvedNode.Value == lookFor.Value

	case MatchPattern:
		if !found {
			return false
		}
		_, resolvedNode := resolveNode(lookFor.Path, root)
		if resolvedNode == nil {
			return false
		}
		matched, err := regexp.MatchString(lookFor.Pattern, resolvedNode.Value)
		if err != nil {
			return false
		}
		return matched

	default:
		return false
	}
}
