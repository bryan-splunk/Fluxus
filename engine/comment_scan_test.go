package engine_test

import (
	"testing"

	"github.com/bryan-splunk/Fluxus/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanOne is a helper that builds a single-file State and scans it.
func scanOne(t *testing.T, config string, rules []*engine.Rule, includeComments bool) []engine.Effect {
	t.Helper()
	state, err := engine.NewState(map[string]string{"test.yaml": config})
	require.NoError(t, err)
	return engine.Scan(state, rules, "0.153", false, includeComments)
}

// iisRule is a minimal component-level rule with NO in_comments selector, used
// to prove auto-derivation: the rule is evaluated against commented blocks even
// though it never opted in.
func iisRule() *engine.Rule {
	return &engine.Rule{
		ID:         "P2-15",
		Category:   engine.CategoryP2,
		Introduced: "0.131",
		Title:      "IIS receiver",
		LookFor: []engine.LookFor{
			{Path: "$.receivers.iis", Match: engine.MatchExists},
		},
	}
}

const configWithCommentedIIS = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

  # IIS — Enable on web servers - This is a comment that should be ignored in the test
  # iis:
  #   collection_interval: 60s

  jaeger:
    protocols:
      grpc:
        endpoint: 0.0.0.0:14250
`

// TestGapB_AutoDerivesCommentMatch verifies a rule with only a normal selector
// (no in_comments) fires as an IsComment effect when --include-comments is set
// and its target component is present commented-out at the child level.
func TestGapB_AutoDerivesCommentMatch(t *testing.T) {
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{iisRule()}, true)
	require.Len(t, effects, 1, "expected one comment effect")
	assert.True(t, effects[0].IsComment, "effect should be flagged as a comment match")
	assert.Equal(t, "P2-15", effects[0].Rule.ID)
}

// TestGapB_DisabledWithoutIncludeComments verifies the comment path is inert
// unless --include-comments is supplied.
func TestGapB_DisabledWithoutIncludeComments(t *testing.T) {
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{iisRule()}, false)
	assert.Empty(t, effects, "comment match must not fire without include-comments")
}

// TestGapB_PerRuleOverrideDisables verifies scan_comments: false suppresses the
// comment match while leaving active-config behaviour untouched.
func TestGapB_PerRuleOverrideDisables(t *testing.T) {
	rule := iisRule()
	rule.ScanComments = new(false)
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{rule}, true)
	assert.Empty(t, effects, "scan_comments: false must suppress the comment match")
}

// TestGapB_ActiveStillFiresAndNoDuplicate verifies that an ACTIVE component still
// produces a normal (non-comment) effect, and a commented copy elsewhere does not
// create a duplicate beyond the single comment effect.
func TestGapB_ActiveComponentFiresNormally(t *testing.T) {
	config := `receivers:
  iis:
    collection_interval: 30s
`
	effects := scanOne(t, config, []*engine.Rule{iisRule()}, true)
	require.Len(t, effects, 1)
	assert.False(t, effects[0].IsComment, "active component should fire a non-comment effect")
}

// TestGapB_RenameAppliesInComments verifies that once a rename rule fires as a
// comment effect, ApplyMigration rewrites the commented key (B-01 behaviour).
func TestGapB_RenameAppliesInComments(t *testing.T) {
	rule := &engine.Rule{
		ID:         "P3-01",
		Category:   engine.CategoryP3,
		Introduced: "0.153",
		Title:      "rename windowseventlog",
		LookFor: []engine.LookFor{
			{Path: "$.receivers.windowseventlog", Match: engine.MatchExists},
		},
		Migration: engine.Migration{
			Strategy: "auto",
			KeyMoves: []engine.KeyMove{
				{From: "$.receivers.windowseventlog", To: "$.receivers.windows_event_log"},
			},
		},
	}

	config := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

  # [OPTIONAL] often handled by SIEM.  - This is a comment that should be ignored in the test
  # windowseventlog/security:
  #   channel: Security
  #   max_reads: 100
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
}

// TestGapB_AbsentSelectorDoesNotFireInComments verifies that a MatchAbsent
// selector (meaningless against an isolated commented component) is excluded
// from comment scanning and does not spuriously fire.
func TestGapB_AbsentSelectorDoesNotFireInComments(t *testing.T) {
	rule := &engine.Rule{
		ID:         "X-ABSENT",
		Category:   engine.CategoryP3,
		Introduced: "0.153",
		Title:      "absent selector",
		LookFor: []engine.LookFor{
			{Path: "$.receivers.iis.metrics", Match: engine.MatchAbsent},
		},
	}
	// An absent selector legitimately fires on the ACTIVE path (the field is not
	// present in active config); that behaviour is unchanged. What must NOT happen
	// is a COMMENT effect driven by the absent selector.
	effects := scanOne(t, configWithCommentedIIS, []*engine.Rule{rule}, true)
	for _, e := range effects {
		assert.False(t, e.IsComment, "absent selectors must not drive comment matches")
	}
}
