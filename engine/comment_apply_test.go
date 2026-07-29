package engine_test

import (
	"testing"

	"github.com/bryan-splunk/fluxus/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// iisAnnotateRule builds an IIS rule that injects an upgrade-note annotation via
// a comment_path key_move. strategy lets a test exercise auto vs guided.
func iisAnnotateRule(strategy string) *engine.Rule {
	return &engine.Rule{
		ID:         "P2-15",
		Category:   engine.CategoryP2,
		Introduced: "0.131",
		Title:      "IIS metrics defaults",
		LookFor: []engine.LookFor{
			{Path: "$.receivers.iis", Match: engine.MatchExists},
		},
		Migration: engine.Migration{
			Strategy: strategy,
			KeyMoves: []engine.KeyMove{
				{
					CommentPath: "$.receivers.iis",
					CommentText: "# [I-15 UPGRADE v0.131] iis metrics now enabled by default\n# review the defaults before re-enabling",
				},
			},
		},
	}
}

// TestGapC_AnnotationInjectedAboveCommentedComponent verifies comment_path
// annotations are injected as '#'-prefixed lines, indented to align with, and
// positioned immediately above, the commented component they target.
func TestGapC_AnnotationInjectedAboveCommentedComponent(t *testing.T) {
	rule := iisAnnotateRule("auto")
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{rule}, true)
	require.Len(t, effects, 1)
	require.True(t, effects[0].IsComment)

	eff := effects[0]
	eff.FilePath = "test.yaml"
	updated, _, err := engine.ApplyMigration(configWithCommentedIIS, eff)
	require.NoError(t, err)

	assert.Contains(t, updated, "  "+engine.AnnotBoundaryOpen,
		"opening boundary should be injected")
	assert.Contains(t, updated, "  # [I-15 UPGRADE v0.131] iis metrics now enabled by default",
		"annotation should be injected with the commented component's indentation")
	assert.Contains(t, updated, "  # review the defaults before re-enabling")
	assert.Contains(t, updated, "  "+engine.AnnotBoundaryClose,
		"closing boundary should be injected")

	// The opening boundary must sit directly above the annotation block, which
	// sits directly above the commented key. Layout expected:
	//   idxAnnot+0  # -- Config Upgrade Note ---...
	//   idxAnnot+1  # [I-15 UPGRADE ...]
	//   idxAnnot+2  # review the defaults...
	//   idxAnnot+3  # -------...
	//   idxAnnot+4  # iis:
	idxAnnot := indexOfLine(updated, "  "+engine.AnnotBoundaryOpen)
	idxKey := indexOfLine(updated, "  # iis:")
	require.GreaterOrEqual(t, idxAnnot, 0)
	require.GreaterOrEqual(t, idxKey, 0)
	assert.Less(t, idxAnnot, idxKey, "opening boundary must appear above the commented component key")
	assert.Equal(t, idxKey-4, idxAnnot, "boundary+note+close should be the four lines immediately above the key")
}

// TestGapC_AnnotationInjectionIsIdempotent verifies re-running apply on already
// annotated content does not duplicate the note.
func TestGapC_AnnotationInjectionIsIdempotent(t *testing.T) {
	rule := iisAnnotateRule("auto")
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{rule}, true)
	require.Len(t, effects, 1)
	eff := effects[0]
	eff.FilePath = "test.yaml"

	once, _, err := engine.ApplyMigration(configWithCommentedIIS, eff)
	require.NoError(t, err)
	twice, _, err := engine.ApplyMigration(once, eff)
	require.NoError(t, err)
	assert.Equal(t, once, twice, "annotation injection must be idempotent on re-apply")
}

// TestGapC_GuidedCommentEffectStillAnnotates verifies that a guided/inform_only
// rule matched ONLY in comments is applied (its annotation is injected) and
// reported under CommentEffects rather than the active "manual action" list.
func TestGapC_GuidedCommentEffectStillAnnotates(t *testing.T) {
	rule := iisAnnotateRule("guided")
	state, err := engine.NewState(map[string]string{"f.yaml": configWithCommentedIIS})
	require.NoError(t, err)

	res, err := engine.Apply(state, []*engine.Rule{rule}, engine.ApplyOptions{
		TargetVersion:   "0.153",
		IncludeComments: true,
		ApprovedIDs:     []string{"all"},
	})
	require.NoError(t, err)

	require.Len(t, res.CommentEffects, 1, "guided comment match should be reported as a comment finding")
	assert.Empty(t, res.GuidedEffects, "comment-only match must not appear as active manual action")
	assert.Empty(t, res.AppliedEffects, "no active config changed")

	out := res.UpdatedFiles["f.yaml"]
	assert.Contains(t, out, engine.AnnotBoundaryOpen,
		"opening boundary should be injected for guided comment match")
	assert.Contains(t, out, "# [I-15 UPGRADE v0.131] iis metrics now enabled by default",
		"guided rule should still carry its guidance into the commented template")
	assert.Contains(t, out, engine.AnnotBoundaryClose,
		"closing boundary should be injected for guided comment match")
}

// TestGapC_RenameThenAnnotateOrdering verifies that when a rule both renames a
// commented component and annotates it, the rename runs first so the annotation
// lands above the NEW (post-rename) key.
func TestGapC_RenameThenAnnotateOrdering(t *testing.T) {
	rule := &engine.Rule{
		ID:         "P3-01",
		Category:   engine.CategoryP3,
		Introduced: "0.153",
		Title:      "rename windowseventlog + note",
		LookFor: []engine.LookFor{
			{Path: "$.receivers.windowseventlog", Match: engine.MatchExists},
		},
		Migration: engine.Migration{
			Strategy: "auto",
			KeyMoves: []engine.KeyMove{
				{From: "$.receivers.windowseventlog", To: "$.receivers.windows_event_log"},
				{
					CommentPath: "$.receivers.windows_event_log",
					CommentText: "# [B-01 UPGRADE v0.149] windowseventlog renamed to windows_event_log",
				},
			},
		},
	}

	config := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

  # Security event log — often handled by SIEM.
  # windowseventlog/security:
  #   channel: Security
`
	effects := scanOne(t, config, []*engine.Rule{rule}, true)
	require.Len(t, effects, 1)
	require.True(t, effects[0].IsComment)

	eff := effects[0]
	eff.FilePath = "test.yaml"
	updated, _, err := engine.ApplyMigration(config, eff)
	require.NoError(t, err)

	assert.Contains(t, updated, "# windows_event_log/security:", "commented key should be renamed")
	assert.NotContains(t, updated, "# windowseventlog/security:", "old commented name should be gone")
	assert.Contains(t, updated, engine.AnnotBoundaryOpen,
		"opening boundary should be injected")
	assert.Contains(t, updated, "# [B-01 UPGRADE v0.149]", "annotation should be injected")
	assert.Contains(t, updated, engine.AnnotBoundaryClose,
		"closing boundary should be injected")

	// Layout expected (boundary + 1 content line + close = 3 lines above key):
	//   idxAnnot+0  # -- Config Upgrade Note ---...
	//   idxAnnot+1  # [B-01 UPGRADE v0.149] ...
	//   idxAnnot+2  # -------...
	//   idxAnnot+3  # windows_event_log/security:
	idxAnnot := indexOfLine(updated, "  "+engine.AnnotBoundaryOpen)
	idxKey := indexOfLine(updated, "  # windows_event_log/security:")
	require.GreaterOrEqual(t, idxAnnot, 0)
	require.GreaterOrEqual(t, idxKey, 0)
	assert.Equal(t, idxKey-3, idxAnnot, "boundary+note+close should be three lines above the renamed commented key")
}

// indexOfLine returns the 0-based line index of the first line in content that
// equals want exactly, or -1 if none match.
func indexOfLine(content, want string) int {
	idx := 0
	start := 0
	for i := 0; i <= len(content); i++ {
		if i == len(content) || content[i] == '\n' {
			if content[start:i] == want {
				return idx
			}
			idx++
			start = i + 1
		}
	}
	return -1
}
