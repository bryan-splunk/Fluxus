package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bryan-splunk/Fluxus/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Per-rule data-driven fixtures
// ---------------------------------------------------------------------------

// ruleFixture is the schema for testdata/rules/<rule-id>.test.yaml files.
type ruleFixture struct {
	Cases []fixtureCase `yaml:"cases"`
}

type fixtureCase struct {
	Name             string   `yaml:"name"`
	Config           string   `yaml:"config"`
	ShouldFire       bool     `yaml:"should_fire"`
	ApplyContains    []string `yaml:"apply_contains"`
	ApplyNotContains []string `yaml:"apply_not_contains"`
}

// TestRuleFixtures auto-discovers every *.test.yaml file under testdata/rules/,
// resolves the named rule from the live rules tree, and runs each case as a
// named sub-test.
func TestRuleFixtures(t *testing.T) {
	rulesDir := filepath.Join("..", "rules")
	fixtureDir := filepath.Join("..", "testdata", "rules")

	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Skip("rules dir not present")
	}
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("testdata/rules dir not present")
	}

	allRules, err := engine.LoadRulesTree(rulesDir)
	require.NoError(t, err)

	byID := make(map[string]*engine.Rule, len(allRules))
	for _, r := range allRules {
		byID[strings.ToUpper(r.ID)] = r
	}

	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	fixtureCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".test.yaml") {
			continue
		}
		fixtureCount++

		// Derive rule ID from filename, e.g. "p2-01.test.yaml" → "P2-01"
		stem := strings.TrimSuffix(entry.Name(), ".test.yaml")
		ruleID := strings.ToUpper(stem)

		rule, ok := byID[ruleID]
		if !ok {
			t.Errorf("fixture %s: no rule with id %q found in rules tree", entry.Name(), ruleID)
			continue
		}

		raw, readErr := os.ReadFile(filepath.Join(fixtureDir, entry.Name()))
		require.NoError(t, readErr, "reading fixture %s", entry.Name())

		var fixture ruleFixture
		require.NoError(t, yaml.Unmarshal(raw, &fixture), "parsing fixture %s", entry.Name())
		require.NotEmpty(t, fixture.Cases, "fixture %s has no cases", entry.Name())

		// tick: use the rule's own introduced version so the rule is always applicable.
		tick := rule.Introduced
		if tick == "" || tick == "0.0.0" {
			tick = "0.0.0"
		}

		t.Run(ruleID, func(t *testing.T) {
			for _, tc := range fixture.Cases {
				tc := tc
				t.Run(tc.Name, func(t *testing.T) {
					rawMap := map[string]string{"test.yaml": tc.Config}
					state, stateErr := engine.NewState(rawMap)
					require.NoError(t, stateErr)

					effects := engine.Scan(state, []*engine.Rule{rule}, tick, false, false)

					if tc.ShouldFire {
						assert.NotEmpty(t, effects, "rule %s should fire but did not", ruleID)
					} else {
						assert.Empty(t, effects, "rule %s should not fire but did", ruleID)
					}

					// Apply checks — only run when the rule fired and we have expectations.
					if tc.ShouldFire && len(effects) > 0 && (len(tc.ApplyContains) > 0 || len(tc.ApplyNotContains) > 0) {
						eff := effects[0]
						eff.FilePath = "test.yaml"
						updated, _, applyErr := engine.ApplyMigration(tc.Config, eff)
						require.NoError(t, applyErr, "ApplyMigration failed for rule %s", ruleID)

						for _, want := range tc.ApplyContains {
							assert.Contains(t, updated, want, "rule %s apply output missing %q", ruleID, want)
						}
						for _, notWant := range tc.ApplyNotContains {
							assert.NotContains(t, updated, notWant, "rule %s apply output should not contain %q", ruleID, notWant)
						}
					}
				})
			}
		})
	}

	assert.Positive(t, fixtureCount, "no fixture files found in testdata/rules/")
}

func TestLoadRules(t *testing.T) {
	rulesDir := filepath.Join("..", "rules")
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Skip("rules dir not present")
	}
	// Use LoadRulesTree so security/pipeline subdirectory rules are also loaded.
	rules, err := engine.LoadRulesTree(rulesDir)
	require.NoError(t, err)
	assert.NotEmpty(t, rules)

	for _, r := range rules {
		assert.NotEmpty(t, r.ID, "rule missing id")
		assert.NotEmpty(t, r.Title, "rule %s missing title", r.ID)
		assert.NotEmpty(t, r.Introduced, "rule %s missing introduced version", r.ID)
	}
}

func TestFilterByVersion(t *testing.T) {
	rules := []*engine.Rule{
		{ID: "A", Introduced: "0.120", Category: engine.CategoryP1},
		{ID: "B", Introduced: "0.130", Category: engine.CategoryP2},
		{ID: "C", Introduced: "0.153", Category: engine.CategoryP3},
	}

	applicable, future, err := engine.FilterByVersion(rules, "0.145")
	require.NoError(t, err)
	assert.Len(t, applicable, 2) // A and B
	assert.Len(t, future, 1)     // C (0.153 > 0.145)
	assert.Equal(t, "C", future[0].ID)
}

func TestEvalPath_Exists(t *testing.T) {
	raw := map[string]string{
		"test.yaml": `
exporters:
  otlp:
    sending_queue:
      blocking: true
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	rules := []*engine.Rule{
		{
			ID:         "P1-20",
			Category:   engine.CategoryP1,
			Introduced: "0.129",
			Title:      "sending_queue::blocking removed",
			LookFor: []engine.LookFor{
				{Path: "$.exporters.*.sending_queue.blocking", Match: engine.MatchExists},
			},
		},
	}

	effects := engine.Scan(state, rules, "0.129", false, false)
	assert.Len(t, effects, 1)
	assert.Equal(t, "P1-20", effects[0].Rule.ID)
}

func TestEvalPath_Absent(t *testing.T) {
	raw := map[string]string{
		"test.yaml": `
exporters:
  otlp:
    endpoint: localhost:4317
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	rules := []*engine.Rule{
		{
			ID:         "P1-20",
			Category:   engine.CategoryP1,
			Introduced: "0.129",
			Title:      "sending_queue::blocking removed",
			LookFor: []engine.LookFor{
				{Path: "$.exporters.*.sending_queue.blocking", Match: engine.MatchExists},
			},
		},
	}

	effects := engine.Scan(state, rules, "0.129", false, false)
	assert.Empty(t, effects) // field not present → no match
}

func TestCommentScanner(t *testing.T) {
	raw := `
receivers:
  otlp: {}

# kafka:
#   brokers: [kafka:9092]
#   client_id: my-collector
`
	blocks, err := engine.ExtractCommentedBlocks(raw)
	require.NoError(t, err)
	assert.NotEmpty(t, blocks)

	// The commented block should have parsed a kafka key
	found := false
	for _, b := range blocks {
		if _, ok := b["kafka"]; ok {
			found = true
		}
	}
	assert.True(t, found, "expected to find kafka key in commented blocks")
}

func TestTopologyValidation_EmptyExporters(t *testing.T) {
	raw := map[string]string{
		"agent.yaml": `
receivers:
  otlp: {}
processors:
  batch: {}
exporters: {}
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: []
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	issues := engine.ValidateTopology(state)
	assert.NotEmpty(t, issues)
	assert.Equal(t, "error", issues[0].Severity)
}

// ============================================================
// Fix 1: match: absent fires when the path is NOT present
// ============================================================

func TestMatchAbsent_FiresWhenKeyMissing(t *testing.T) {
	// kafka receiver present, but client_id is absent — rule should fire.
	raw := map[string]string{
		"test.yaml": `
receivers:
  kafka:
    brokers: [kafka:9092]
    topic: my-topic
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	rules := []*engine.Rule{
		{
			ID:         "P2-01",
			Category:   engine.CategoryP2,
			Introduced: "0.130",
			Title:      "kafka client_id absent",
			Logic:      "and",
			LookFor: []engine.LookFor{
				{Path: "$.receivers.kafka", Match: engine.MatchExists},
				{Path: "$.receivers.kafka.client_id", Match: engine.MatchAbsent},
			},
		},
	}

	effects := engine.Scan(state, rules, "0.130", false, false)
	assert.Len(t, effects, 1, "should fire when client_id absent")
}

func TestMatchAbsent_SilentWhenKeyPresent(t *testing.T) {
	// client_id already set — rule must NOT fire.
	raw := map[string]string{
		"test.yaml": `
receivers:
  kafka:
    brokers: [kafka:9092]
    client_id: sarama
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	rules := []*engine.Rule{
		{
			ID:         "P2-01",
			Category:   engine.CategoryP2,
			Introduced: "0.130",
			Title:      "kafka client_id absent",
			Logic:      "and",
			LookFor: []engine.LookFor{
				{Path: "$.receivers.kafka", Match: engine.MatchExists},
				{Path: "$.receivers.kafka.client_id", Match: engine.MatchAbsent},
			},
		},
	}

	effects := engine.Scan(state, rules, "0.130", false, false)
	assert.Empty(t, effects, "should not fire when client_id is already set")
}

// ============================================================
// Fix 3: named component instances (kafka/consumer, etc.)
// ============================================================

func TestNamedComponentInstance_Detected(t *testing.T) {
	// Named instance "kafka/consumer" — should be detected by $.receivers.kafka selector.
	raw := map[string]string{
		"test.yaml": `
receivers:
  kafka/consumer:
    brokers: [kafka:9092]
    topic: my-topic
`,
	}
	state, err := engine.NewState(raw)
	require.NoError(t, err)

	rules := []*engine.Rule{
		{
			ID:         "P2-01",
			Category:   engine.CategoryP2,
			Introduced: "0.130",
			Title:      "kafka named instance",
			LookFor: []engine.LookFor{
				{Path: "$.receivers.kafka", Match: engine.MatchExists},
			},
		},
	}

	effects := engine.Scan(state, rules, "0.130", false, false)
	assert.Len(t, effects, 1, "named instance kafka/consumer should match $.receivers.kafka")
}

// ============================================================
// Option A: key-level migration
// ============================================================

func TestApplyMigration_KeyMove_Rename(t *testing.T) {
	// Simulate the P1-07 batcher → sending_queue.batch rename.
	raw := `
exporters:
  splunk_hec:
    token: abc123
    batcher:
      min_size_items: 80
      min_size_bytes: 512
`
	rule := &engine.Rule{
		ID:         "P1-07",
		Category:   engine.CategoryP1,
		Introduced: "0.151",
		Title:      "splunk_hec batcher removed",
		Migration: engine.Migration{
			Strategy: "auto",
			KeyMoves: []engine.KeyMove{
				{
					From: "$.exporters.splunk_hec.batcher.min_size_items",
					To:   "$.exporters.splunk_hec.sending_queue.batch.min_size_items",
				},
				{
					From: "$.exporters.splunk_hec.batcher.min_size_bytes",
					To:   "$.exporters.splunk_hec.sending_queue.batch.min_size_bytes",
				},
				{From: "$.exporters.splunk_hec.batcher", To: ""},
			},
		},
	}

	effect := engine.Effect{Rule: rule, FilePath: "test.yaml"}
	updated, warns, err := engine.ApplyMigration(raw, effect)

	require.NoError(t, err)
	assert.Empty(t, warns)

	// User's values (80, 512) must be present in the output.
	assert.Contains(t, updated, "min_size_items: 80", "user value 80 must be preserved")
	assert.Contains(t, updated, "min_size_bytes: 512", "user value 512 must be preserved")
	// Keys must be under the new path.
	assert.Contains(t, updated, "sending_queue:")
	assert.Contains(t, updated, "batch:")
	// Old batcher block must be gone.
	assert.NotContains(t, updated, "batcher:", "old batcher key must be removed")
}

func TestApplyMigration_InjectDefault(t *testing.T) {
	// Simulate the P2-01 inject: client_id added with default "sarama".
	raw := `
receivers:
  kafka:
    brokers: [kafka:9092]
    topic: otel-spans
`
	rule := &engine.Rule{
		ID:         "P2-01",
		Category:   engine.CategoryP2,
		Introduced: "0.130",
		Title:      "kafka client_id inject",
		Migration: engine.Migration{
			Strategy: "auto",
			KeyMoves: []engine.KeyMove{
				// to-only form: inject default when key is absent, leave it alone when present
				{To: "$.receivers.kafka.client_id", Default: "sarama"},
			},
		},
	}

	effect := engine.Effect{Rule: rule, FilePath: "test.yaml"}
	updated, warns, err := engine.ApplyMigration(raw, effect)

	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Contains(t, updated, "client_id: sarama", "default client_id must be injected")
	// Original keys preserved.
	assert.Contains(t, updated, "brokers:")
	assert.Contains(t, updated, "otel-spans")
}

func TestApplyMigration_InjectDefault_NoOverwrite(t *testing.T) {
	// When client_id is already set, the inject must not overwrite it.
	raw := `
receivers:
  kafka:
    brokers: [kafka:9092]
    client_id: otel-collector
`
	rule := &engine.Rule{
		ID:         "P2-01",
		Category:   engine.CategoryP2,
		Introduced: "0.130",
		Title:      "kafka client_id inject — already set",
		Migration: engine.Migration{
			Strategy: "auto",
			KeyMoves: []engine.KeyMove{
				{To: "$.receivers.kafka.client_id", Default: "sarama"},
			},
		},
	}

	effect := engine.Effect{Rule: rule, FilePath: "test.yaml"}
	updated, _, err := engine.ApplyMigration(raw, effect)

	require.NoError(t, err)
	// Existing value must be preserved.
	assert.Contains(t, updated, "client_id: otel-collector")
	assert.NotContains(t, updated, "client_id: sarama")
}

func TestApplyMigration_GuidedStrategy_NoChange(t *testing.T) {
	raw := `
processors:
  routing:
    from_attribute: env
`
	rule := &engine.Rule{
		ID:       "P1-01",
		Category: engine.CategoryP1,
		Migration: engine.Migration{
			Strategy: "guided",
			Before:   "processors:\n  routing:",
			After:    "connectors:\n  routing:",
		},
	}

	effect := engine.Effect{Rule: rule, FilePath: "test.yaml"}
	updated, warns, err := engine.ApplyMigration(raw, effect)

	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, raw, updated, "guided strategy must not modify the file")
}

func TestApplyMigration_InformOnly_NoChange(t *testing.T) {
	raw := `receivers:
  prometheus: {}
`
	rule := &engine.Rule{
		ID:       "P2-18",
		Category: engine.CategoryP2,
		Migration: engine.Migration{
			Strategy: "inform_only",
		},
	}

	effect := engine.Effect{Rule: rule, FilePath: "test.yaml"}
	updated, warns, err := engine.ApplyMigration(raw, effect)

	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, raw, updated, "inform_only strategy must not modify the file")
}

// ============================================================

func TestConflictDetection(t *testing.T) {
	rule1 := &engine.Rule{ID: "A", Category: engine.CategoryP1}
	rule2 := &engine.Rule{ID: "B", Category: engine.CategoryP2}

	ticks := []engine.TickResult{
		{Version: "0.123", Effects: []engine.Effect{
			{Rule: rule1, FilePath: "a.yaml", MatchedPath: "$.exporters.kafka.client_id", FiredAtTick: "0.123"},
		}},
		{Version: "0.130", Effects: []engine.Effect{
			{Rule: rule2, FilePath: "a.yaml", MatchedPath: "$.exporters.kafka.client_id", FiredAtTick: "0.130"},
		}},
	}

	conflicts := engine.DetectConflicts(ticks)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "A", conflicts[0].Effect1.Rule.ID)
	assert.Equal(t, "B", conflicts[0].Effect2.Rule.ID)
}
