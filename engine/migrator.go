package engine

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// annotationBoundaryOpen and annotationBoundaryClose are the delimiter lines that wrap
// every injected upgrade-note annotation, making the engine-generated guidance
// visually distinct from operator-authored comments in both active config and
// commented-out templates. Both lines are the same width (65 chars).
const (
	annotationBoundaryOpen  = "# ---------------- Config Upgrade Note --------------------------"
	annotationBoundaryClose = "# ---------------------------------------------------------------"
)

// ApplyMigration applies the migration for a detected effect to the raw YAML content.
//
// Strategy controls behaviour:
//   - "auto" (default): use key_moves when present, else string replacement
//   - "guided":         report-only; no automated change applied
//   - "inform_only":    detection-only; nothing to migrate in collector YAML
//
// Returns (updatedContent, warnings, error). Warnings are non-fatal notes for the
// report (e.g. a key was not found so a default was injected).
func ApplyMigration(rawContent string, effect Effect) (string, []string, error) {
	migration := effect.Rule.Migration
	strategy := strings.ToLower(migration.Strategy)
	if strategy == "" {
		strategy = "auto"
	}

	// Comment-block effects are handled FIRST, before the strategy gate. Editing
	// commented-out config only ever adds or rewrites '#' lines — it never
	// touches the live YAML structure — so it is always safe regardless of the
	// rule's strategy (auto/guided/inform_only). A guided rule still carries its
	// upgrade guidance into the commented template; the "needs manual review"
	// classification is unchanged because no active config is modified.
	if effect.IsComment {
		if len(migration.KeyMoves) > 0 {
			return applyCommentBlock(rawContent, effect)
		}
		// No key_moves: nothing to change in comment blocks.
		return rawContent, nil, nil
	}

	// Guided and inform_only rules produce report entries but no active file
	// changes, EXCEPT for comment-only key_moves applied to the live tree. Comment
	// injection is always safe (it only adds # lines — it never changes YAML
	// structure) so we apply it even for guided/inform_only rules. This ensures
	// upgrade guidance travels with the config file without changing the "needs
	// manual review" classification.
	if strategy == "guided" || strategy == "inform_only" {
		commentMoves := commentOnlyKeyMoves(migration.KeyMoves)
		if len(commentMoves) > 0 {
			commentEffect := effect
			commentEffect.Rule = shallowCopyRuleWithMoves(effect.Rule, commentMoves)
			return applyKeyMoves(rawContent, commentEffect)
		}
		return rawContent, nil, nil
	}

	// Option A: key_moves (preferred — preserves user values).
	if len(migration.KeyMoves) > 0 {
		return applyKeyMoves(rawContent, effect)
	}

	// Fallback: string-based replacement.
	if migration.Before == "" || migration.After == "" {
		return rawContent, nil, nil
	}
	// If the before block is purely illustrative comments (every non-blank line
	// starts with #), there is nothing to search for in the YAML file.
	// Treat it like inform_only — report the detection but make no file change.
	if isCommentOnlyBlock(migration.Before) {
		return rawContent, nil, nil
	}
	before := strings.ReplaceAll(strings.TrimSpace(migration.Before), "\r\n", "\n")
	after := strings.TrimSpace(migration.After)
	if strings.Contains(rawContent, before) {
		return strings.Replace(rawContent, before, after, 1), nil, nil
	}
	return rawContent, nil, fmt.Errorf("rule %s: could not locate migration target in %s (manual review required)",
		effect.Rule.ID, effect.FilePath)
}

// applyCommentBlock applies a rule's comment-path migration to commented-out
// configuration via text manipulation (no YAML parse — the block is not valid
// YAML while commented). It is the comment-path counterpart to applyKeyMoves and
// runs for every comment effect regardless of strategy. Two operations, in order:
//
//  1. Renames — "# oldKey:" → "# newKey:" (applyCommentBlockKeyMoves). Done first
//     so that step 2 can target the post-rename component name.
//  2. Annotations — for each comment_path key_move, the comment_text is injected
//     as '#'-prefixed lines immediately above the matching commented component,
//     indented to align with it. Injection is idempotent (skipped when the same
//     note already sits in the comment run above the target).
//
// Structural moves that cannot be expressed safely as text edits on commented
// content (deletes, child injections that add new keys, sequence operations) are
// still skipped — re-enabling the template surfaces them on the active path.
func applyCommentBlock(rawContent string, effect Effect) (string, []string, error) {
	result, warnings, err := applyCommentBlockKeyMoves(rawContent, effect)
	if err != nil {
		return rawContent, warnings, err
	}
	result = injectCommentAnnotations(result, effect)
	return result, warnings, nil
}

// injectCommentAnnotations inserts each comment_path key_move's comment_text into
// the commented-out config, immediately above the commented component the
// comment_path points at. The component is located by its leaf name (the last
// path segment) appearing as a commented key line ("# <leaf>:" or
// "# <leaf>/<variant>:"). comment_text lines (already '#'-prefixed) are indented
// to match the commented key's leading whitespace. CommentOnce restricts the
// injection to the first matching component; otherwise every match is annotated,
// mirroring the active-path behaviour.
func injectCommentAnnotations(rawContent string, effect Effect) string {
	lines := strings.Split(rawContent, "\n")
	for _, keyMove := range effect.Rule.Migration.KeyMoves {
		if keyMove.CommentPath == "" || keyMove.CommentText == "" {
			continue
		}
		annotationLines := splitAnnotationLines(keyMove.CommentText)
		if len(annotationLines) == 0 {
			continue
		}
		leaf := lastPathSegment(keyMove.CommentPath)
		if leaf == "" || leaf == "*" {
			continue
		}
		keyRegexp := regexp.MustCompile(`^([ \t]*)#[ \t]+` + regexp.QuoteMeta(leaf) + `(?:/[^:\s]*)?[ \t]*:`)

		output := make([]string, 0, len(lines)+len(annotationLines)+2)
		injected := false
		for _, line := range lines {
			match := keyRegexp.FindStringSubmatch(line)
			if match != nil && !(keyMove.CommentOnce && injected) {
				indent := match[1]
				// Use the opening boundary as the idempotency anchor — more
				// stable than the first annotation line and unique enough that
				// false-positive matches are extremely unlikely.
				if !annotationPresentAbove(output, indent+annotationBoundaryOpen) {
					output = append(output, indent+annotationBoundaryOpen)
					for _, annotationLine := range annotationLines {
						if annotationLine == "" {
							output = append(output, "")
							continue
						}
						output = append(output, indent+annotationLine)
					}
					output = append(output, indent+annotationBoundaryClose)
					injected = true
				}
			}
			output = append(output, line)
		}
		lines = output
	}
	return strings.Join(lines, "\n")
}

// splitAnnotationLines normalises a comment_text block into individual lines with
// the trailing newline (from YAML block scalars) removed. Each returned line is
// expected to already begin with '#'.
func splitAnnotationLines(commentText string) []string {
	trimmed := strings.TrimRight(strings.ReplaceAll(commentText, "\r\n", "\n"), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// annotationPresentAbove reports whether firstLine already appears in the
// contiguous run of comment/blank lines at the tail of out. Used to keep
// annotation injection idempotent across repeated apply runs.
func annotationPresentAbove(out []string, firstLine string) bool {
	want := strings.TrimSpace(firstLine)
	for i := len(out) - 1; i >= 0; i-- {
		t := strings.TrimSpace(out[i])
		if t == "" {
			continue
		}
		if !strings.HasPrefix(t, "#") {
			return false
		}
		if t == want {
			return true
		}
	}
	return false
}

// applyCommentBlockKeyMoves performs text-based substitutions on comment lines
// for rules whose effect fired via an in_comments: true selector.
//
// Only rename key_moves (From != "" && To != "", different leaf names) are
// applied. Deletes, injections, sequence operations, and comment injections are
// intentionally skipped — they either do not apply to commented text or would
// require full re-parsing which is not safe on arbitrary comment content.
func applyCommentBlockKeyMoves(rawContent string, effect Effect) (string, []string, error) {
	result := rawContent
	for _, keyMove := range effect.Rule.Migration.KeyMoves {
		// Skip non-rename moves.
		if keyMove.From == "" || keyMove.To == "" || keyMove.From == keyMove.To {
			continue
		}
		if keyMove.CommentPath != "" || keyMove.SequencePath != "" {
			continue
		}
		fromLeaf := lastPathSegment(keyMove.From)
		toLeaf := lastPathSegment(keyMove.To)
		if fromLeaf == toLeaf {
			continue
		}
		// Replace occurrences in comment lines only:
		//   # <indent>fromLeaf:  →  # <indent>toLeaf:
		//   # <indent>fromLeaf/variant:  →  # <indent>toLeaf/variant:
		//
		// The regex matches the key name delimited by "/" or ":" so partial
		// matches (e.g. "hostmetrics_extra") are not affected.
		pattern := regexp.MustCompile(`(?m)(^[ \t]*#[ \t]+(?:[a-zA-Z0-9_./-]*[ \t]+)?)` +
			regexp.QuoteMeta(fromLeaf) + `((?:/[^: \t]*)?[ \t]*:)`)
		result = pattern.ReplaceAllString(result, "${1}"+toLeaf+"${2}")
	}
	return result, nil, nil
}

// lastPathSegment returns the final dot-separated segment of a YAMLPath string.
// e.g. "$.receivers.hostmetrics" → "hostmetrics"
func lastPathSegment(path string) string {
	// Strip leading $. prefix.
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return path
	}
	return path[i+1:]
}

// isCommentOnlyBlock reports whether every non-blank line in s starts with '#'.
// Used to detect illustrative migration.before blocks that have no YAML to match.
func isCommentOnlyBlock(s string) bool {
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return true
}

// applyKeyMoves parses the YAML, applies all KeyMove operations, and re-marshals.
// yaml.v3 preserves HeadComment/LineComment/FootComment on nodes during round-trip,
// so inline comments are retained. Block indentation is normalised to 2 spaces.
func applyKeyMoves(rawContent string, effect Effect) (string, []string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(rawContent), &root); err != nil {
		return rawContent, nil, fmt.Errorf("parse YAML for key moves: %w", err)
	}

	var warnings []string
	for _, keyMove := range effect.Rule.Migration.KeyMoves {
		moveWarnings := executeKeyMove(&root, keyMove)
		warnings = append(warnings, moveWarnings...)
	}

	// Fix BUG-1/BUG-2: after renames, yaml.v3 sometimes stores inter-component
	// section-separator comments as FootComments on the LAST LEAF of the preceding
	// component's subtree rather than as HeadComments on the following key. When
	// encoded, these comments are written at the leaf's deep indentation (12+ spaces)
	// instead of at the section's two-space level. Promote them to the correct position.
	fixSectionCommentDrift(&root)

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	// Normalize comments before encoding: collapse blank lines that yaml.v3 inserts
	// between consecutive # lines within the same comment block.
	normalizeComments(unwrapDocument(&root))
	if err := encoder.Encode(&root); err != nil {
		return rawContent, warnings, fmt.Errorf("re-marshal after key moves: %w", err)
	}

	result := buffer.String()

	// ── Post-processing: fix yaml.v3 round-trip formatting artefacts ──────────

	// 1. Collapse 3+ consecutive newlines to at most 2 (one blank line).
	tripleNewline := regexp.MustCompile(`\n{3,}`)
	result = tripleNewline.ReplaceAllString(result, "\n\n")

	// 2. Collapse blank lines between consecutive comment-only lines.
	//    Uses indentation as a secondary signal: only blank lines separating two
	//    comment lines at the *same* indent level are removed. A blank line between
	//    comment lines at different indentation levels is treated as an intentional
	//    section boundary and is preserved.
	result = collapseCommentBlankLinesInString(result)

	// 3. Ensure a blank line precedes each standard top-level collector section
	//    key. yaml.v3 can drop these blank-line separators on round-trip when the
	//    adjacent node had its comments cleared (e.g. via deleteFromParent).
	sectionSep := regexp.MustCompile(`\n(extensions:|receivers:|processors:|exporters:|service:)\n`)
	result = sectionSep.ReplaceAllString(result, "\n\n$1\n")

	// Re-apply the triple-newline collapse: sectionSep can introduce 3+ newlines
	// when a section key was already preceded by a blank line.
	result = tripleNewline.ReplaceAllString(result, "\n\n")

	return result, warnings, nil
}

// replaceSequenceValues finds all sequence nodes at sequencePath and performs one
// of three operations on their scalar items based on oldValue/newValue:
//
//	replace (both set):          rename items equal to oldValue → newValue (preserves /suffix).
//	delete  (newValue == ""):      remove all items equal to oldValue (or oldValue/suffix).
//	add     (oldValue == ""):      append newValue if not already present in the sequence.
func replaceSequenceValues(root *yaml.Node, sequencePath, oldValue, newValue string) []string {
	hits := findAll(root, normalizePath(sequencePath))
	for _, hit := range hits {
		seqNode := hit.parent.Content[hit.keyIndex+1]
		if seqNode.Kind != yaml.SequenceNode {
			continue
		}

		switch {
		case oldValue == "" && newValue != "":
			// ADD: append newValue when not already present.
			already := false
			for _, item := range seqNode.Content {
				if item.Kind == yaml.ScalarNode && item.Value == newValue {
					already = true
					break
				}
			}
			if !already {
				seqNode.Content = append(seqNode.Content, scalarNode(newValue))
			}

		case oldValue != "" && newValue == "":
			// DELETE: remove items matching oldValue or oldValue/suffix.
			kept := make([]*yaml.Node, 0, len(seqNode.Content))
			for _, item := range seqNode.Content {
				if item.Kind == yaml.ScalarNode &&
					(item.Value == oldValue || strings.HasPrefix(item.Value, oldValue+"/")) {
					continue // drop
				}
				kept = append(kept, item)
			}
			seqNode.Content = kept

		default:
			// REPLACE: rename oldValue → newValue, preserving named-instance suffixes.
			for _, item := range seqNode.Content {
				if item.Kind != yaml.ScalarNode {
					continue
				}
				if item.Value == oldValue {
					item.Value = newValue
				} else if strings.HasPrefix(item.Value, oldValue+"/") {
					item.Value = newValue + item.Value[len(oldValue):]
				}
			}
		}
	}
	return nil // non-fatal: component may simply not be in any pipeline
}

// deleteSequenceMapItems finds all sequence nodes at move.SequenceMapPath and removes
// every mapping item in each sequence where move.MatchKey == move.MatchValue.
// Returns a slice of warning strings (non-fatal; an empty sequence is not an error).
func deleteSequenceMapItems(root *yaml.Node, move KeyMove) []string {
	hits := findAll(root, normalizePath(move.SequenceMapPath))
	for _, hit := range hits {
		seqNode := hit.parent.Content[hit.keyIndex+1]
		if seqNode.Kind != yaml.SequenceNode {
			continue
		}
		kept := make([]*yaml.Node, 0, len(seqNode.Content))
		for _, item := range seqNode.Content {
			if item.Kind != yaml.MappingNode {
				kept = append(kept, item)
				continue
			}
			matched := false
			for i := 0; i+1 < len(item.Content); i += 2 {
				keyNode := item.Content[i]
				valNode := item.Content[i+1]
				if keyNode.Value == move.MatchKey && valNode.Value == move.MatchValue {
					matched = true
					break
				}
			}
			if !matched {
				kept = append(kept, item)
			}
		}
		seqNode.Content = kept
	}
	return nil
}

// executeKeyMove executes a single KeyMove operation on the YAML node tree.
func executeKeyMove(root *yaml.Node, keyMove KeyMove) []string {
	// Sequence map-item delete: remove mapping items from a sequence by key=value match.
	if keyMove.SequenceMapPath != "" {
		return deleteSequenceMapItems(root, keyMove)
	}

	// Comment injection: prepend CommentText as a HeadComment on the key at CommentPath.
	if keyMove.CommentPath != "" {
		commentPath := normalizePath(keyMove.CommentPath)
		if keyMove.CommentText == "" {
			return nil
		}
		// Wrap with visual boundary delimiters so the engine-generated guidance
		// is visually distinct from operator-authored comments. Trim trailing
		// newlines from | block scalars first so yaml.v3 doesn't insert an
		// extra blank line between the last boundary line and the key node.
		commentText := annotationBoundaryOpen + "\n" +
			strings.TrimRight(keyMove.CommentText, "\n") + "\n" +
			annotationBoundaryClose
		hits := findAll(root, commentPath)
		// comment_once: true — when the comment describes the receiver/component type
		// in general (not a specific named instance), inject only on the first match
		// to avoid duplicates when multiple named instances exist (GAP-9).
		if keyMove.CommentOnce && len(hits) > 1 {
			hits = hits[:1]
		}
		for _, hit := range hits {
			keyNode := hit.parent.Content[hit.keyIndex]
			if keyNode.HeadComment == "" {
				keyNode.HeadComment = commentText
			} else {
				// Append the upgrade note to the END of any existing head comment
				// so it lands directly above the key rather than at the top of
				// whatever large operator comment block yaml.v3 attached to the node.
				keyNode.HeadComment = keyNode.HeadComment + "\n" + commentText
			}
		}
		return nil
	}

	// Sequence operations: add, delete, or replace items in array nodes.
	if keyMove.SequencePath != "" {
		return replaceSequenceValues(root, keyMove.SequencePath, keyMove.OldValue, keyMove.NewValue)
	}

	var warnings []string
	fromPath := normalizePath(keyMove.From)
	toPath := normalizePath(keyMove.To)

	switch {
	case keyMove.From == "" && keyMove.To != "" && keyMove.Default != "":
		// Inject-if-absent: write Default at To only when To does not exist.
		if keyMove.InjectAtEach {
			// Find every named instance of the parent path (e.g. processors.filter,
			// processors.filter/exclude_dev, …) and inject the leaf key into each
			// one individually only when the leaf is absent in that instance.
			leafKey, parentPattern := splitLastSegment(toPath)
			for _, hit := range findAll(root, parentPattern) {
				// Build the concrete leaf path for this specific instance.
				concreteTo := hit.fullPath + "." + leafKey
				if len(findAll(root, concreteTo)) == 0 {
					if err := setAtPath(root, concreteTo, defaultToNode(keyMove.Default)); err != nil {
						warnings = append(warnings, fmt.Sprintf("inject %s: %v", concreteTo, err))
					}
				}
			}
		} else if len(findAll(root, toPath)) == 0 {
			if err := setAtPath(root, toPath, defaultToNode(keyMove.Default)); err != nil {
				warnings = append(warnings, fmt.Sprintf("inject %s: %v", keyMove.To, err))
			}
		}

	case keyMove.From != "" && keyMove.To == "":
		// Delete: remove all nodes matching From.
		// Sort descending by keyIndex within each parent to prevent index invalidation
		// when multiple keys in the same mapping are deleted in sequence.
		delResults := findAll(root, fromPath)
		sort.Slice(delResults, func(i, j int) bool {
			if delResults[i].parent == delResults[j].parent {
				return delResults[i].keyIndex > delResults[j].keyIndex
			}
			return false
		})
		for _, hit := range delResults {
			deleteFromParent(hit.parent, hit.keyIndex)
		}

	case keyMove.From != "" && keyMove.To != "":
		// When from == to the intent is "inject default if absent" — it is not a
		// real move and we must NOT delete the key after writing it back.
		if fromPath == toPath {
			if len(findAll(root, fromPath)) == 0 && keyMove.Default != "" {
				if err := setAtPath(root, toPath, defaultToNode(keyMove.Default)); err != nil {
					warnings = append(warnings, fmt.Sprintf("inject %s: %v", keyMove.To, err))
				}
			}
			break
		}
		// Move: transfer the value from From to To, then delete From.
		results := findAll(root, fromPath)
		if len(results) == 0 {
			if keyMove.Default != "" {
				if err := setAtPath(root, toPath, defaultToNode(keyMove.Default)); err != nil {
					warnings = append(warnings, fmt.Sprintf("inject default for %s: %v", keyMove.To, err))
				}
			}
			break
		}
		// Optimisation: when the rename only changes the leaf key name and both
		// from and to share the same parent path (e.g. receivers.hostmetrics →
		// receivers.host_metrics), do an in-place key rename. This preserves the
		// original key order and avoids comment drift.
		// Skip the optimisation when WrapAsSequence is set — that operation must
		// rewrite the value node, which the in-place rename path does not do.
		if isSameParentRename(fromPath, toPath) && !keyMove.WrapAsSequence {
			toLeaf := toPath[strings.LastIndex(toPath, ".")+1:]
			fromLeaf := fromPath[strings.LastIndex(fromPath, ".")+1:]
			for _, hit := range results {
				keyNode := hit.parent.Content[hit.keyIndex]
				if keyNode.Value == fromLeaf {
					keyNode.Value = toLeaf
				} else if strings.HasPrefix(keyNode.Value, fromLeaf+"/") {
					// Named instance: hostmetrics/process → host_metrics/process
					keyNode.Value = toLeaf + keyNode.Value[len(fromLeaf):]
				}
			}
			break
		}
		// Cross-parent move: sort hits by keyIdx descending within each parent so
		// that deletions of higher-index keys never invalidate the stored keyIdx of
		// lower-index keys in the same mapping node (classic index-invalidation prevention).
		sort.Slice(results, func(i, j int) bool {
			if results[i].parent == results[j].parent {
				return results[i].keyIndex > results[j].keyIndex
			}
			return false
		})
		for _, hit := range results {
			valueNode := hit.parent.Content[hit.keyIndex+1]
			movedNode := cloneNode(valueNode)
			// WrapAsSequence: wrap the moved scalar in a single-item sequence node.
			// Used to rename a scalar field to a list field (e.g. group_rebalance_strategy → group_rebalance_strategies).
			if keyMove.WrapAsSequence {
				seqNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
				seqNode.Content = []*yaml.Node{movedNode}
				movedNode = seqNode
			}
			concreteTo := resolveWildcards(hit.fullPath, fromPath, toPath)
			if err := setAtPath(root, concreteTo, movedNode); err != nil {
				warnings = append(warnings, fmt.Sprintf("set %s: %v", concreteTo, err))
				continue
			}
			deleteFromParent(hit.parent, hit.keyIndex)
		}
	}
	return warnings
}

// deleteFromParent removes the key-value pair at keyIdx from a MappingNode while
// preserving any commented-out content that yaml.v3 attached around the deleted
// node but that does not actually belong to it.
//
// The problem: a commented-out block sitting physically above a live key (e.g.
// disabled `# windowsperfcounters:` / `# iis:` template receivers above a live
// `fluentforward:` receiver) has no YAML key of its own, so yaml.v3 stores it in
// the *following* key's HeadComment and/or the *preceding* value's FootComment.
// Earlier revisions cleared both fields to stop comment drift (D-4/GAP-3), which
// silently destroyed that unrelated commented config whenever the live key was
// deleted (e.g. P1-17 removing fluentforward took ~56 lines of template receivers
// with it).
//
// The fix: extract those orphaned comments and re-home them onto the next
// surviving sibling key's HeadComment (or, when deleting the last pair in the
// mapping, onto the predecessor value's FootComment, which fixSectionCommentDrift
// then promotes to the correct indentation). Only the deleted node's *own* inline
// (LineComment) and trailing (FootComment) comments are dropped, since those
// genuinely describe the removed value.
func deleteFromParent(parent *yaml.Node, keyIndex int) {
	if parent == nil || keyIndex+1 >= len(parent.Content) {
		return
	}
	keyNode := parent.Content[keyIndex]
	valueNode := parent.Content[keyIndex+1]

	// Collect orphaned comments (in top-to-bottom file order) so they survive the
	// splice: the predecessor value's FootComment sits above the deleted key, and
	// the deleted key's HeadComment sits immediately above it.
	var preserved []string
	if keyIndex >= 2 {
		previousValue := parent.Content[keyIndex-1]
		if previousValue.FootComment != "" {
			preserved = append(preserved, previousValue.FootComment)
			previousValue.FootComment = ""
		}
	}
	if keyNode.HeadComment != "" {
		preserved = append(preserved, keyNode.HeadComment)
	}

	// Drop the deleted node's own comments so they cannot drift to adjacent keys.
	keyNode.HeadComment = ""
	keyNode.LineComment = ""
	keyNode.FootComment = ""
	valueNode.HeadComment = ""
	valueNode.LineComment = ""
	valueNode.FootComment = ""

	// Splice out the key/value pair.
	parent.Content = append(parent.Content[:keyIndex], parent.Content[keyIndex+2:]...)

	if len(preserved) == 0 {
		return
	}
	merged := strings.Join(preserved, "\n")

	// Re-home onto the next surviving sibling key (the element that shifted up
	// into the deleted slot), keeping the commented content at sibling indentation.
	if keyIndex < len(parent.Content) {
		nextKey := parent.Content[keyIndex]
		if nextKey.HeadComment == "" {
			nextKey.HeadComment = merged
		} else {
			nextKey.HeadComment = merged + "\n" + nextKey.HeadComment
		}
		return
	}

	// No surviving sibling (deleted the last pair): re-home onto the predecessor
	// value's FootComment. fixSectionCommentDrift promotes it to the next section.
	if keyIndex >= 2 {
		previousValue := parent.Content[keyIndex-1]
		if previousValue.FootComment == "" {
			previousValue.FootComment = merged
		} else {
			previousValue.FootComment = previousValue.FootComment + "\n" + merged
		}
	}
}

// resolveWildcards computes the concrete target path by substituting two classes
// of position in toPattern:
//
//  1. Explicit wildcards (* / **) — replaced with the concrete key captured from
//     the matching position in concreteSrc.
//  2. Named-instance variants — when fromPattern and toPattern share the same
//     segment at position i, but concreteSrc has a longer key at that position
//     (e.g. "splunk_hec/prod"), the concrete key is propagated to toPattern so
//     the move lands under the same named instance.
//
// Example — no wildcard, named instance:
//
//	concreteSrc = "exporters.splunk_hec/prod.batcher.min_size_items"
//	fromPattern = "exporters.splunk_hec.batcher.min_size_items"
//	toPattern   = "exporters.splunk_hec.sending_queue.batch.min_size_items"
//	→              "exporters.splunk_hec/prod.sending_queue.batch.min_size_items"
//
// Example — explicit wildcard:
//
//	concreteSrc = "exporters.otlp.batcher.min_size_items"
//	fromPattern = "exporters.*.batcher.min_size_items"
//	toPattern   = "exporters.*.sending_queue.batch.min_size_items"
//	→              "exporters.otlp.sending_queue.batch.min_size_items"
func resolveWildcards(concreteSource, fromPattern, toPattern string) string {
	sourceSegments := strings.Split(fromPattern, ".")
	concreteSegments := strings.Split(concreteSource, ".")
	toSegments := strings.Split(toPattern, ".")

	result := make([]string, len(toSegments))
	for i, toSegment := range toSegments {
		if toSegment == "*" || toSegment == "**" {
			// Explicit wildcard: substitute the concrete value at position i.
			if i < len(concreteSegments) {
				result[i] = concreteSegments[i]
			} else {
				result[i] = toSegment
			}
		} else if i < len(sourceSegments) && sourceSegments[i] == toSegment {
			// Same segment in both from and to patterns: propagate any named-instance
			// suffix that the user has on this key (e.g. splunk_hec/prod).
			if i < len(concreteSegments) {
				result[i] = concreteSegments[i]
			} else {
				result[i] = toSegment
			}
		} else {
			// Different segment in from vs to (the actual rename position).
			// Preserve any named-instance /suffix from the concrete source key so that
			// e.g. "hostmetrics/process" → "host_metrics/process" rather than "host_metrics".
			if i < len(concreteSegments) && i < len(sourceSegments) && strings.HasPrefix(concreteSegments[i], sourceSegments[i]+"/") {
				result[i] = toSegment + concreteSegments[i][len(sourceSegments[i]):]
			} else {
				result[i] = toSegment
			}
		}
	}
	return strings.Join(result, ".")
}

// commentOnlyKeyMoves returns the subset of moves that are pure comment injections
// (CommentPath is set; no From, To, SequencePath, or Default).
func commentOnlyKeyMoves(moves []KeyMove) []KeyMove {
	var results []KeyMove
	for _, keyMove := range moves {
		if keyMove.CommentPath != "" && keyMove.From == "" && keyMove.To == "" && keyMove.SequencePath == "" {
			results = append(results, keyMove)
		}
	}
	return results
}

// shallowCopyRuleWithMoves returns a shallow copy of r with Migration.KeyMoves replaced.
func shallowCopyRuleWithMoves(r *Rule, moves []KeyMove) *Rule {
	ruleCopy := *r
	ruleCopy.Migration.KeyMoves = moves
	return &ruleCopy
}

// normalizeComments recursively collapses blank lines that yaml.v3 inserts
// between consecutive comment lines during round-trip serialisation.
// The encoder adds an extra \n between each # line stored in a HeadComment/
// FootComment string; this walk removes those extra separators before encoding.
func normalizeComments(node *yaml.Node) {
	if node == nil {
		return
	}
	node.HeadComment = collapseCommentBlankLinesInString(node.HeadComment)
	node.LineComment = collapseCommentBlankLinesInString(node.LineComment)
	node.FootComment = collapseCommentBlankLinesInString(node.FootComment)
	for _, child := range node.Content {
		normalizeComments(child)
	}
}

// collapseCommentBlankLinesInString collapses blank lines that appear between
// consecutive comment-only lines in a string. Used for both individual comment
// node fields (HeadComment etc.) and the fully-serialised YAML output.
//
// Indentation equality is used as a secondary signal:
//   - same indent → same comment block → blank line is dropped
//   - different indent → intentional section boundary → blank line is kept
//
// This avoids merging a section-level header comment with an indented
// field-level comment that yaml.v3 happened to store on the adjacent node.
func collapseCommentBlankLinesInString(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		// Only inspect blank lines that sit between two comment lines.
		if strings.TrimSpace(line) == "" && len(out) > 0 && i+1 < len(lines) {
			prev := out[len(out)-1]
			next := lines[i+1]
			prevTrim := strings.TrimSpace(prev)
			nextTrim := strings.TrimSpace(next)
			if strings.HasPrefix(prevTrim, "#") && strings.HasPrefix(nextTrim, "#") {
				prevIndent := len(prev) - len(strings.TrimLeft(prev, " \t"))
				nextIndent := len(next) - len(strings.TrimLeft(next, " \t"))
				// Collapse only when indentation levels match exactly.
				if prevIndent == nextIndent {
					continue // drop the blank line
				}
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// ============================================================
// BUG-1/BUG-2: Section-level FootComment drift repair
// ============================================================

// fixSectionCommentDrift repairs FootComment placement after key renames.
//
// yaml.v3 sometimes attaches inter-component section-separator comments (e.g.
// "# --- jaeger ---", "# not being used") as FootComments on the LAST LEAF
// scalar deep inside the preceding component's sub-tree rather than as
// HeadComments on the following component key. After an in-place key rename
// (e.g. hostmetrics → host_metrics) the comment stays attached to that deep
// leaf and is then written at 12+ spaces indentation — visually broken.
//
// This function walks the top-level section mappings (receivers:, processors:,
// exporters:, connectors:, extensions:) and promotes any such sunk FootComments
// to HeadComments on the correct following sibling key within the section.
func fixSectionCommentDrift(root *yaml.Node) {
	doc := unwrapDocument(root)
	if doc == nil || doc.Kind != yaml.MappingNode {
		return
	}
	topSections := map[string]bool{
		"receivers":  true,
		"processors": true,
		"exporters":  true,
		"connectors": true,
		"extensions": true,
	}

	for i := 0; i+1 < len(doc.Content); i += 2 {
		sectionKey := doc.Content[i]
		if !topSections[sectionKey.Value] {
			continue
		}
		sectionMapping := doc.Content[i+1]
		if sectionMapping.Kind != yaml.MappingNode {
			continue
		}
		// Promote FootComments between components within this section.
		promoteSiblingFootComments(sectionMapping)
		// Also promote any FootComment trailing the last component in the section
		// up to the next top-level section key's HeadComment.
		if len(sectionMapping.Content) >= 2 && i+3 < len(doc.Content) {
			lastVal := sectionMapping.Content[len(sectionMapping.Content)-1]
			footComment := drainLastLeafFootComment(lastVal)
			if footComment != "" {
				nextTopKey := doc.Content[i+2]
				if nextTopKey.HeadComment == "" {
					nextTopKey.HeadComment = footComment
				} else {
					nextTopKey.HeadComment = footComment + "\n" + nextTopKey.HeadComment
				}
			}
		}
	}
}

// promoteSiblingFootComments walks a MappingNode and for each key-value pair,
// drains any FootComment from the last leaf of the value's sub-tree and
// prepends it to the HeadComment of the next sibling key.
func promoteSiblingFootComments(mapping *yaml.Node) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	// i iterates over key nodes (0, 2, 4, …); i+1 is the value, i+2 is the
	// next key. We need at least two key-value pairs for there to be a "next".
	for i := 0; i+3 < len(mapping.Content); i += 2 {
		valNode := mapping.Content[i+1]
		nextKey := mapping.Content[i+2]

		footComment := drainLastLeafFootComment(valNode)
		if footComment == "" {
			continue
		}
		if nextKey.HeadComment == "" {
			nextKey.HeadComment = footComment
		} else {
			nextKey.HeadComment = footComment + "\n" + nextKey.HeadComment
		}
	}
}

// drainLastLeafFootComment walks the yaml sub-tree rooted at node in
// reverse-child order and returns the FootComment of the first (deepest-last)
// node that has one, clearing it from the node so it does not get written
// at the original leaf's indentation level.
func drainLastLeafFootComment(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	// Recurse into children in reverse order so we find the rightmost/last leaf.
	for j := len(node.Content) - 1; j >= 0; j-- {
		if fc := drainLastLeafFootComment(node.Content[j]); fc != "" {
			return fc
		}
	}
	if node.FootComment != "" {
		footComment := node.FootComment
		node.FootComment = ""
		return footComment
	}
	return ""
}
