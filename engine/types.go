// Package engine implements the core State/Effect upgrade processing pipeline.
//
// The engine uses a tick-per-version model inspired by game-physics State/Effect patterns:
//   - State  — immutable snapshot of parsed YAML config(s)
//   - Effect — a detected rule match (pure data, never applied immediately)
//   - Tick   — one version step: evaluate rules → accumulate effects → apply
//
// Two-phase execution:
//  1. DryRun  — full tick chain in read-only mode → produces PreAssessment report
//  2. Apply   — full tick chain writing changes    → produces updated config files
package engine

import "gopkg.in/yaml.v3"

// Category classifies the priority of a rule.
// P1 = Breaking (collector fails or key function stops).
// P2 = Degrading (collector runs but behaves sub-optimally).
// P3 = Advisory (no functional impact; best practice / future-proofing).
type Category string

const (
	CategoryP1 Category = "p1"
	CategoryP2 Category = "p2"
	CategoryP3 Category = "p3"
)

// Phase identifies which processing stage a rule belongs to.
// Rules are loaded from phase-named subdirectories under the rules root
// (e.g. rules/security/, rules/pipeline/) and carry this field so the engine
// can filter or order phases independently.
//
// Current phases:
//
//	config   — versioned breaking-change rules for the collector migration
//	           (default; all P1-xx / P2-xx / P3-xx rules at the rules root)
//	security — evergreen security checks (hardcoded credentials, insecure
//	           settings) that run on every config regardless of version
//	pipeline — structural / topology checks on the pipeline graph
//	post     — checks that run against the already-upgraded output
type Phase string

const (
	PhaseConfig   Phase = "config"
	PhaseSecurity Phase = "security"
	PhasePipeline Phase = "pipeline"
	PhasePost     Phase = "post"
)

// MatchType describes how a look_for selector determines a match.
type MatchType string

const (
	MatchExists  MatchType = "exists"  // key is present with any value
	MatchAbsent  MatchType = "absent"  // key is NOT present
	MatchValue   MatchType = "value"   // key equals the specified Value
	MatchPattern MatchType = "pattern" // key value matches the specified regex Pattern
)

// LookFor is a single selector within a rule's look_for list.
type LookFor struct {
	// Path is a YAMLPath expression (e.g. "$.exporters.*.sending_queue.blocking").
	Path string `yaml:"path"`
	// Match describes the condition that triggers the rule.
	Match MatchType `yaml:"match"`
	// Value is the expected value when Match == MatchValue.
	Value string `yaml:"value,omitempty"`
	// Pattern is a regex string when Match == MatchPattern.
	Pattern string `yaml:"pattern,omitempty"`
	// InComments indicates this selector should also be evaluated against
	// commented-out YAML blocks extracted by the comment scanner.
	InComments bool `yaml:"in_comments,omitempty"`
	// RawPattern, when non-empty, evaluates as a Go regexp against the raw file
	// text rather than the parsed YAML tree. Useful for security rules that need
	// to detect credential patterns at arbitrary depth (e.g. inside sequences
	// or deeply nested scrape_config blocks). When set, Path and Match are
	// ignored for this selector. Combine with logic: "and" and a path-based
	// selector to require both a structural key and a textual pattern match.
	RawPattern string `yaml:"raw_pattern,omitempty"`
}

// KeyMove describes a single key-level structural migration (Option A).
// The engine uses these to transform the user's YAML tree, preserving the
// user's actual leaf values while moving, renaming, or injecting keys.
//
// There are six operation modes — use the fields that match your intent:
//
//   - Move/rename:   From + To — value at From is written to To, From is deleted.
//   - Delete key:    From only (To empty) — key at From is removed.
//   - Inject scalar: To + Default (no From) — Default written at To only if To absent.
//   - Inject block:  To + Default containing newlines — Default parsed as a YAML
//     sub-tree and merged at To only if To is absent.
//   - Sequence ops:  SequencePath + OldValue/NewValue — operates on array items:
//     replace: both OldValue and NewValue set — rename matching items.
//     delete:  OldValue set, NewValue empty — remove matching items.
//     add:     OldValue empty, NewValue set — append item if not already present.
//   - Comment:       CommentPath + CommentText — prepend CommentText as a HeadComment
//     on the key node at CommentPath (useful for inline upgrade guidance).
//
// Paths use the same YAMLPath syntax as LookFor (e.g. "$.exporters.*.batcher.min_size_items").
// A * wildcard in From is tracked and substituted into the matching position of To,
// so the same concrete component name is used on both sides of the move.
type KeyMove struct {
	// From is a YAMLPath to the source key. Leave empty for inject-only operations.
	From string `yaml:"from,omitempty"`
	// To is a YAMLPath to the destination. Leave empty to delete the From key.
	To string `yaml:"to,omitempty"`
	// Default is written at To when From is absent and To does not yet exist.
	// A multi-line value is parsed as a YAML block and merged as a sub-tree.
	Default string `yaml:"default,omitempty"`

	// Sequence operations (mutually exclusive with From/To/CommentPath):
	// Finds all sequence nodes at SequencePath and operates on their scalar items.
	//   replace: OldValue + NewValue — rename OldValue → NewValue (preserves /suffix).
	//   delete:  OldValue + NewValue:"" — remove items equal to OldValue (or OldValue/*).
	//   add:     OldValue:"" + NewValue — append NewValue if not already present.
	SequencePath string `yaml:"sequence_path,omitempty"`
	OldValue     string `yaml:"old_value,omitempty"`
	NewValue     string `yaml:"new_value,omitempty"`

	// Comment injection (mutually exclusive with all other fields):
	// Finds the key node at CommentPath and prepends CommentText as a HeadComment.
	// Used to add inline upgrade guidance lines into the YAML output.
	CommentPath string `yaml:"comment_path,omitempty"`
	CommentText string `yaml:"comment_text,omitempty"`
	// CommentOnce, when true, restricts comment injection to the FIRST match
	// returned by CommentPath even when multiple nodes match (e.g. prometheus,
	// prometheus/internal, prometheus/netscaler). Without this flag every named
	// instance would receive the comment, producing duplicates when the comment
	// is about the receiver type in general rather than a specific instance.
	CommentOnce bool `yaml:"comment_once,omitempty"`

	// InjectAtEach, when true, changes inject-if-absent behaviour for the
	// "to + default (no from)" mode. Instead of checking the full path once and
	// injecting at an exact key, the engine finds all instances of the parent
	// path (including named instances such as filter/exclude_dev) and injects the
	// leaf key with Default into each parent mapping where the leaf is absent.
	//
	// Example — inject error_mode: ignore into every filter/XXX instance:
	//   to:             $.processors.filter.error_mode
	//   default:        ignore
	//   inject_at_each: true
	InjectAtEach bool `yaml:"inject_at_each,omitempty"`
}

// Migration holds migration guidance and optional automated key moves.
type Migration struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Notes  string `yaml:"notes,omitempty"`
	// Strategy controls how ApplyMigration behaves:
	//   "auto"        — use key_moves if present, else string replacement (default)
	//   "guided"      — report the change with guidance; no automated apply
	//   "inform_only" — detection only; nothing to migrate in collector YAML
	Strategy string `yaml:"strategy,omitempty"`
	// KeyMoves lists structural key operations for Option A migration.
	// When present, these are used instead of string-based before/after replacement.
	KeyMoves []KeyMove `yaml:"key_moves,omitempty"`
}

// Rule represents a single breaking-change rule loaded from a rules/*.yaml file.
type Rule struct {
	ID         string   `yaml:"id"`
	Category   Category `yaml:"category"`
	Introduced string   `yaml:"introduced"` // semver string, e.g. "0.129"
	// Phase identifies which processing stage this rule belongs to.
	// Defaults to "config" when empty. Security rules use "security",
	// pipeline checks use "pipeline", etc. See Phase constants.
	Phase Phase  `yaml:"phase,omitempty"`
	Title string `yaml:"title"`
	// Logic controls how multiple look_for selectors are combined:
	//   "or"  — any selector firing is sufficient (default)
	//   "and" — ALL selectors must match for the rule to fire
	Logic string `yaml:"logic,omitempty"`
	// Order controls execution sequence within a tick when multiple rules share
	// the same Introduced version. Lower values run first (default 0). Rules
	// with equal Order values retain their alphabetical file-load order.
	Order   int       `yaml:"order,omitempty"`
	LookFor []LookFor `yaml:"look_for"`
	// ScanComments overrides the comment-scan behaviour for this rule when the
	// engine runs with --include-comments. It is the per-rule override on top of
	// the default auto-derivation (the rule's own look_for selectors are evaluated
	// against commented-out component blocks discovered by DetectCommentedComponents):
	//   nil   — auto: scan comments using the rule's look_for selectors (default)
	//   true  — force: same as auto (explicit opt-in / documents intent)
	//   false — disable: never evaluate this rule against commented-out blocks
	// Use false to silence noisy rules (e.g. ones whose commented match is not
	// actionable) without removing their active-config behaviour.
	ScanComments *bool     `yaml:"scan_comments,omitempty"`
	Migration    Migration `yaml:"migration"`
	Description  string    `yaml:"description"`
	// Guidance holds the extended operational guidance for this rule — the
	// "Action:" prose and any post-YAML notes from the knowledge base (accuracy
	// checks, transition-period notes, actions outside the config file such as
	// firewall updates or env var changes). It is distinct from Description
	// (engine-focused) and comment_text (injected inline into the user's YAML).
	// Rendered in reports only when --include-guidance / include_guidance is set.
	Guidance string   `yaml:"guidance,omitempty"`
	SeeAlso  []string `yaml:"see_also,omitempty"`
}

// Effect is a detected rule match produced during scanning. It is pure data —
// it describes what needs to change but does not apply the change itself.
type Effect struct {
	Rule        *Rule
	FiredAtTick string // version tick where this effect was detected
	IsFuture    bool   // true if this tick is beyond the user's target version
	IsComment   bool   // true if the match was found in a commented-out block
	FilePath    string // config file where the match was found
	MatchedPath string // the YAMLPath expression that triggered the match
}

// TickResult captures the effects produced during a single version tick.
type TickResult struct {
	Version string
	Effects []Effect
}

// State is an immutable snapshot of one or more parsed YAML config files.
// Files maps a filename to the parsed YAML node tree.
// Raw maps a filename to the original file content (for comment extraction).
type State struct {
	Files map[string]*yaml.Node
	Raw   map[string]string
}

// Clone returns a deep copy of the State so ticks can mutate their working copy
// without affecting the original.
func (s *State) Clone() *State {
	clone := &State{
		Files: make(map[string]*yaml.Node, len(s.Files)),
		Raw:   make(map[string]string, len(s.Raw)),
	}
	for fileName, content := range s.Raw {
		clone.Raw[fileName] = content
	}
	for fileName, node := range s.Files {
		clone.Files[fileName] = node // yaml.Node is treated as read-only between ticks
	}
	return clone
}

// NewState constructs a State from a map of filename → raw YAML content.
func NewState(rawFiles map[string]string) (*State, error) {
	state := &State{
		Files: make(map[string]*yaml.Node, len(rawFiles)),
		Raw:   make(map[string]string, len(rawFiles)),
	}
	for name, content := range rawFiles {
		state.Raw[name] = content
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(content), &node); err != nil {
			return nil, &ParseError{File: name, Err: err}
		}
		state.Files[name] = &node
	}
	return state, nil
}

// Finding pairs an Effect with the user's approval decision.
type Finding struct {
	Effect   Effect
	Approved bool
}

// Conflict describes two effects at different ticks that modify the same config key.
type Conflict struct {
	Effect1 Effect
	Effect2 Effect
	Key     string // the conflicting YAMLPath
	Message string
}

// ValidationIssue is a problem found by the topology validator.
type ValidationIssue struct {
	Severity string // "error" | "warning" | "info"
	File     string
	Message  string
	Detail   string
}

// ParseError wraps a YAML parse failure with its source filename.
type ParseError struct {
	File string
	Err  error
}

func (e *ParseError) Error() string {
	return "parse error in " + e.File + ": " + e.Err.Error()
}
