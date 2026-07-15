package engine

import (
	"bufio"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractCommentedBlocks scans raw YAML content for contiguous blocks of
// comment lines (lines starting with optional whitespace + '#'). Each block
// is stripped of its '#' prefix characters and re-parsed as YAML. Only blocks
// that successfully parse into a map are returned.
//
// This enables the scanner to detect rules that match commented-out config
// sections — a common pattern where operators maintain a full config template
// with disabled options commented out.
func ExtractCommentedBlocks(rawContent string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	scanner := bufio.NewScanner(strings.NewReader(rawContent))
	var block []string

	flush := func() {
		if len(block) == 0 {
			return
		}
		content := strings.Join(block, "\n")
		var m map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &m); err == nil && m != nil {
			results = append(results, m)
		}
		block = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			// Strip leading '#' and optional single space
			stripped := strings.TrimPrefix(trimmed, "#")
			stripped = strings.TrimPrefix(stripped, " ")
			block = append(block, stripped)
		} else {
			// Non-comment line: flush current block
			flush()
		}
	}
	flush()

	return results, scanner.Err()
}

// ============================================================
// Gap A: prose-tolerant line/regex comment detector
// ============================================================
//
// The comment path is processed independently from the active path. Unlike the
// active path (which parses the full YAML tree), the comment detector below is
// line/regex based so that prose and separator comments ("# IIS — replaces
// SCOM…", "# ----") interleaved with commented-out config do NOT break
// detection — the failure mode of ExtractCommentedBlocks (which re-parses the
// whole contiguous block as YAML and silently drops it when prose is present).
//
// It also handles the common real-world shape that the YAML-reparse approach
// cannot: a component commented out at the CHILD level inside an otherwise
// active section (e.g. "# iis:" sitting under a live "receivers:" block). Such
// comments carry no parent key, so a path selector like "$.receivers.iis" can
// never resolve against a block rooted at "iis". The detector infers the section
// from the nearest enclosing active (or commented) section key instead.

// commentedSectionKeys are the top-level collector sections under which
// commented components are recognised.
var commentedSectionKeys = map[string]bool{
	"receivers":  true,
	"processors": true,
	"exporters":  true,
	"connectors": true,
	"extensions": true,
	"service":    true,
}

// CommentedComponent describes a single commented-out component definition found
// in a config file (e.g. a disabled "# iis:" receiver template). Detection is
// line/regex based and tolerant of interleaved prose/separator comments.
type CommentedComponent struct {
	Section   string   // enclosing section: receivers/processors/exporters/connectors/extensions/service ("" if unknown)
	Key       string   // component key incl. any named-instance suffix, e.g. "windowseventlog/security"
	Path      string   // synthesized base YAMLPath, e.g. "$.receivers.windowseventlog/security"
	LineStart int      // 0-based index of the "# <key>:" line in the raw file
	LineEnd   int      // 0-based inclusive index of the last line of this component's commented sub-block
	Lines     []string // raw file lines (verbatim, still commented) for this component block
	// Body is the clean, parseable YAML for this component only — the '#' prefixes
	// removed and indentation re-based so the component key sits at column 0, with
	// any trailing prose/separator lines excluded. It is the component key plus its
	// strictly-more-indented children, e.g. "iis:\n  collection_interval: 60s".
	// Used by the comment scan path to re-parse just this component and evaluate
	// rule selectors against it without the prose that breaks a whole-block parse.
	Body string
}

// topLevelSectionKeyRe matches a non-indented mapping key (a top-level section
// like "receivers:"). Used to track the active section context while walking.
var topLevelSectionKeyRegexp = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):\s*(#.*)?$`)

// commentLineRe splits a comment line into its leading file indentation and the
// payload after the leading '#'. group2 is everything after '#'.
var commentLineRegexp = regexp.MustCompile(`^\s*#(.*)$`)

// commentKeyRe matches a YAML mapping key payload (key immediately followed by a
// colon, no internal spaces) plus any inline value. Prose sentences never match
// because a key word is always followed by a space before any colon.
var commentKeyRegexp = regexp.MustCompile(`^([A-Za-z0-9_./-]+):(\s*)(.*)$`)

// DetectCommentedComponents scans raw YAML content for commented-out component
// definitions using a line/regex model. It returns one entry per detected
// component, with the enclosing section inferred from context.
func DetectCommentedComponents(rawContent string) []CommentedComponent {
	lines := strings.Split(rawContent, "\n")
	var components []CommentedComponent

	activeSection := ""
	i := 0
	for i < len(lines) {
		if isCommentLine(lines[i]) {
			j := i
			for j < len(lines) && isCommentLine(lines[j]) {
				j++
			}
			components = append(components, detectInRun(lines, i, j, activeSection)...)
			i = j
			continue
		}
		// Non-comment line: update the active section context.
		if match := topLevelSectionKeyRegexp.FindStringSubmatch(lines[i]); match != nil {
			if commentedSectionKeys[match[1]] {
				activeSection = match[1]
			} else {
				activeSection = ""
			}
		}
		i++
	}
	return components
}

// isCommentLine reports whether a raw line is a comment line (first non-space
// rune is '#').
func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// commentPayload returns the comment payload (text after the leading '#' with a
// single following space removed) and its indentation, preserving the relative
// indentation of any commented-out YAML structure.
func commentPayload(line string) (payload string, indent int, ok bool) {
	m := commentLineRegexp.FindStringSubmatch(line)
	if m == nil {
		return "", 0, false
	}
	p := m[1]
	p = strings.TrimPrefix(p, " ") // drop the single space conventionally after '#'
	trimmed := strings.TrimLeft(p, " ")
	return trimmed, len(p) - len(trimmed), true
}

// commentPayloadLine is a parsed comment payload line within a run.
type commentPayloadLine struct {
	lineIndex  int    // 0-based index in the raw file
	indent     int    // payload indentation (relative to the comment, '#' stripped)
	payload    string // payload text with leading spaces removed
	isKey      bool   // payload looks like a YAML mapping key ("word:")
	key        string // the key name when isKey
	emptyValue bool   // key has no inline value (heads a block mapping)
}

// detectInRun classifies a contiguous comment run [start, end) and emits the
// commented components it contains. activeSection is the enclosing live section.
func detectInRun(lines []string, start, end int, activeSection string) []CommentedComponent {
	// Parse every payload line in the run (keys and children alike) so component
	// bodies can be reconstructed, and prose/separators can be ignored.
	var payloadLines []commentPayloadLine
	for index := start; index < end; index++ {
		payload, indent, ok := commentPayload(lines[index])
		if !ok {
			continue
		}
		payloadLine := commentPayloadLine{lineIndex: index, indent: indent, payload: payload}
		if payload != "" {
			if keyMatch := commentKeyRegexp.FindStringSubmatch(payload); keyMatch != nil {
				payloadLine.isKey = true
				payloadLine.key = keyMatch[1]
				payloadLine.emptyValue = strings.TrimSpace(keyMatch[3]) == ""
			}
		}
		payloadLines = append(payloadLines, payloadLine)
	}

	// baseIndent is the shallowest key indentation in the run.
	baseIndent := -1
	for _, payloadLine := range payloadLines {
		if payloadLine.isKey && (baseIndent == -1 || payloadLine.indent < baseIndent) {
			baseIndent = payloadLine.indent
		}
	}
	if baseIndent == -1 {
		return nil // no key-like lines: pure prose
	}

	// Section mode: if a base-indent key is itself a section name, the run is a
	// fully commented-out section block; components live one level deeper.
	section := activeSection
	componentIndent := baseIndent
	sectionMode := false
	for _, payloadLine := range payloadLines {
		if payloadLine.isKey && payloadLine.indent == baseIndent && commentedSectionKeys[payloadLine.key] {
			section = payloadLine.key
			sectionMode = true
			break
		}
	}
	if sectionMode {
		componentIndent = -1
		for _, payloadLine := range payloadLines {
			if payloadLine.isKey && payloadLine.indent > baseIndent && (componentIndent == -1 || payloadLine.indent < componentIndent) {
				componentIndent = payloadLine.indent
			}
		}
		if componentIndent == -1 {
			return nil // section header only, no components
		}
	}

	// Emit a component for each key at componentIndent that heads a block
	// (empty value, or followed by a more-indented child line).
	var components []CommentedComponent
	for payloadIndex, payloadLine := range payloadLines {
		if !payloadLine.isKey || payloadLine.indent != componentIndent {
			continue
		}
		hasChild := payloadIndex+1 < len(payloadLines) && payloadLines[payloadIndex+1].indent > payloadLine.indent
		if !payloadLine.emptyValue && !hasChild {
			continue // an inline scalar at component level — treat as prose-with-colon
		}

		// Reconstruct the clean body: the key plus its strictly-deeper children,
		// re-based so the key sits at column 0. Stops at the first line whose
		// indentation returns to <= componentIndent (next component / separator).
		var body []string
		for bodyIndex := payloadIndex; bodyIndex < len(payloadLines); bodyIndex++ {
			current := payloadLines[bodyIndex]
			if bodyIndex > payloadIndex && current.indent <= componentIndent {
				break
			}
			if current.payload == "" {
				continue
			}
			body = append(body, strings.Repeat(" ", current.indent-componentIndent)+current.payload)
		}

		// Raw line range: up to the line before the next key at <= componentIndent.
		blockEnd := end - 1
		for nextIndex := payloadIndex + 1; nextIndex < len(payloadLines); nextIndex++ {
			if payloadLines[nextIndex].isKey && payloadLines[nextIndex].indent <= componentIndent {
				blockEnd = payloadLines[nextIndex].lineIndex - 1
				break
			}
		}

		path := "$." + payloadLine.key
		if section != "" {
			path = "$." + section + "." + payloadLine.key
		}
		components = append(components, CommentedComponent{
			Section:   section,
			Key:       payloadLine.key,
			Path:      path,
			LineStart: payloadLine.lineIndex,
			LineEnd:   blockEnd,
			Lines:     append([]string(nil), lines[payloadLine.lineIndex:blockEnd+1]...),
			Body:      strings.Join(body, "\n"),
		})
	}
	return components
}
