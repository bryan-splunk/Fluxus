package engine_test

import (
	"testing"

	"cd.splunkdev.com/bdavies/fluxus/engine"
	"github.com/stretchr/testify/assert"
)

// keysBySection collapses detected components into a section→keys lookup for
// concise assertions.
func keysBySection(components []engine.CommentedComponent) map[string][]string {
	out := map[string][]string{}
	for _, c := range components {
		out[c.Section] = append(out[c.Section], c.Key)
	}
	return out
}

func detectedKeySet(components []engine.CommentedComponent) map[string]bool {
	out := map[string]bool{}
	for _, c := range components {
		out[c.Key] = true
	}
	return out
}

// TestDetectCommentedComponents_ProseInterleaved is the Gap A regression test.
// It feeds the exact prose/separator-interleaved commented receiver templates
// from the windows-standard sample (child-level comments under a live
// receivers: block) and asserts that every commented component is detected and
// no prose line is mistaken for a component. ExtractCommentedBlocks dropped all
// of these because each contiguous block mixed prose with commented config.
func TestDetectCommentedComponents_ProseInterleaved(t *testing.T) {
	raw := `receivers:
  windowsservices:
    collection_interval: 30s

  windows_event_log/application:
    channel: Application
    max_reads: 100

  # [OPTIONAL] Security event log - This is a comment that should be ignored in the test
  # Coordinate with the Security team before enabling.  - This is a comment that should be ignored in the test
  # windowseventlog/security:
  #   channel: Security
  #   max_reads: 100

  # [OPTIONAL] uncomment per server role. - This is a comment that should be ignored in the test
  # windowseventlog/dns:
  #   channel: DNS Server
  # windowseventlog/dirsvc:
  #   channel: Directory Service

  # ----------------------------------------------------------------------------
  # [OPTIONAL — AGGRESSIVE] Windows Performance Counters  - This is a comment block that should be ignored in the test
  # Equivalent SCOM: Windows Performance Counter rules
  # Use for metrics not exposed by hostmetrics — e.g. specific app counters,
  # custom counters published by .NET apps, IIS-specific perf counters.
  # Uncomment and customize per host role.
  # ----------------------------------------------------------------------------
  # windowsperfcounters:
  #   collection_interval: 30s
  #   metrics:
  #     bytes.committed:
  #       description: Bytes committed to memory
  #       unit: By
  #       gauge:
  #   perfcounters:
  #     - object: Memory
  #       counters:
  #         - name: Committed Bytes
  #           metric: bytes.committed

  # ----------------------------------------------------------------------------
  # [OPTIONAL — AGGRESSIVE] Per-role specialty receivers
  # ----------------------------------------------------------------------------

  # IIS — Enable on web servers.  - This is a comment that should be ignored in the test
  # iis:
  #   collection_interval: 60s

  # SQL Server — Enable on DB servers.  - This is a comment that should be ignored in the test
  # Requires a service account with VIEW SERVER STATE permission.
  # sqlserver:
  #   collection_interval: 30s
  #   instance_name: MSSQLSERVER
  #   computer_name: TR1ASP115

  # Active Directory Domain Services —  Enable on DCs.  - This is a comment that should be ignored in the test
  # active_directory_ds:
  #   collection_interval: 30s

  fluentforward:
    endpoint: "0.0.0.0:8006"
`

	components := engine.DetectCommentedComponents(raw)
	found := detectedKeySet(components)

	expected := []string{
		"windowseventlog/security",
		"windowseventlog/dns",
		"windowseventlog/dirsvc",
		"windowsperfcounters",
		"iis",
		"sqlserver",
		"active_directory_ds",
	}
	for _, key := range expected {
		assert.Truef(t, found[key], "expected commented component %q to be detected", key)
	}

	// All detected components must be attributed to the enclosing active section.
	for _, c := range components {
		assert.Equalf(t, "receivers", c.Section,
			"component %q should be scoped to receivers, got %q", c.Key, c.Section)
		assert.Equalf(t, "$.receivers."+c.Key, c.Path, "unexpected path for %q", c.Key)
	}

	// Prose / separators / deep child keys must NOT be treated as components.
	for _, badKey := range []string{
		"description",     // child of a windowsperfcounters metric
		"unit",            // child scalar
		"gauge",           // deep child
		"bytes.committed", // nested metric key, not a component
		"collection_interval",
		"channel",
		"max_reads",
		"instance_name",
		"computer_name",
		"counters",
		"metrics",
		"perfcounters",
	} {
		assert.Falsef(t, found[badKey], "prose/child %q must not be detected as a component", badKey)
	}

	// Active (non-commented) receivers must never be reported by the comment detector.
	assert.False(t, found["windowsservices"], "active receiver must not be detected as commented")
	assert.False(t, found["fluentforward"], "active receiver must not be detected as commented")
	assert.False(t, found["windows_event_log/application"], "active receiver must not be detected as commented")
}

// TestDetectCommentedComponents_MergedIntoFollowingHeadComment reproduces the
// real CompletedTestRuns output shape: after the active scanner reserialized the
// file, every commented template got glued into the following key's (jaeger's)
// HeadComment, the blank-line separators were dropped, and an injected active
// annotation ("# UPGRADE(P3-05 ...)") was prepended on top. The result is ONE
// dense, prose-interleaved comment run. ExtractCommentedBlocks YAML-reparses the
// whole run and drops it (prose => parse failure), which is exactly why lines
// 195/199/201/210/226/230/235 were missed. The line/regex detector must still
// find every component despite the glued-on annotation and absent blank lines.
func TestDetectCommentedComponents_MergedIntoFollowingHeadComment(t *testing.T) {
	raw := `receivers:
  windows_event_log/system:
    channel: System
    max_reads: 100
  # UPGRADE(P3-05 v0.153): receiver.jaeger.DisableRemoteSampling feature gate removed.
  # Remote sampling is now permanently disabled in the jaeger receiver.
  # If this flag appears in startup args (systemd, Helm, Windows registry), remove it.
  # If you relied on remote sampling, use tail_sampling or probabilistic_sampler instead.
  # [OPTIONAL] Security event log — often handled by SIEM.  - This is a comment that should be ignored in the test
  # Coordinate with the Security team before enabling.
  # windowseventlog/security:
  #   channel: Security
  #   max_reads: 100
  # [OPTIONAL] Per-application channels — uncomment per server role.  - This is a comment that should be ignored in the test
  # windowseventlog/dns:
  #   channel: DNS Server
  # windowseventlog/dirsvc:
  #   channel: Directory Service
  # ----------------------------------------------------------------------------
  # [OPTIONAL — AGGRESSIVE] Windows Performance Counters
  # custom counters published by .NET apps, IIS-specific perf counters.
  # ----------------------------------------------------------------------------
  # windowsperfcounters:
  #   collection_interval: 30s
  #   metrics:
  #     bytes.committed:
  #       description: Bytes committed to memory
  #       unit: By
  #       gauge:
  #   perfcounters:
  #     - object: Memory
  #       counters:
  #         - name: Committed Bytes
  #           metric: bytes.committed
  # ----------------------------------------------------------------------------
  # [OPTIONAL — AGGRESSIVE] Per-role specialty receivers
  # ----------------------------------------------------------------------------
  # IIS — Enable on web servers.  - This is a comment that should be ignored in the test
  # iis:
  #   collection_interval: 60s
  # SQL Server — Enable on DB servers.  - This is a comment that should be ignored in the test
  # Requires a service account with VIEW SERVER STATE permission.
  # sqlserver:
  #   collection_interval: 30s
  #   instance_name: MSSQLSERVER
  #   computer_name: TR1ASP115
  # Active Directory Domain Services —  Enable on DCs.  - This is a comment that should be ignored in the test
  # active_directory_ds:
  #   collection_interval: 30s
  # ----------------------------------------------------------------------------
  jaeger:
    protocols:
      grpc:
        endpoint: "${SPLUNK_LISTEN_INTERFACE}:14250"
`
	found := detectedKeySet(engine.DetectCommentedComponents(raw))

	for _, key := range []string{
		"windowseventlog/security",
		"windowseventlog/dns",
		"windowseventlog/dirsvc",
		"windowsperfcounters",
		"iis",
		"sqlserver",
		"active_directory_ds",
	} {
		assert.Truef(t, found[key], "merged-blob: expected %q to be detected", key)
	}
	// The glued-on active annotation and the live jaeger receiver must not appear.
	assert.False(t, found["jaeger"], "active receiver must not be detected as commented")
	assert.False(t, found["bytes.committed"], "nested metric key must not be detected as a component")
}

// TestDetectCommentedComponents_FullyCommentedSection covers the other common
// shape: an entire section commented out including its top-level key. Here the
// section is inferred from the commented "# exporters:" header and components
// live one indent level deeper.
func TestDetectCommentedComponents_FullyCommentedSection(t *testing.T) {
	raw := `# exporters:
#   otlp:
#     endpoint: otelcol:4317
#   otlp/secondary:
#     endpoint: backup:4317
`
	components := engine.DetectCommentedComponents(raw)
	bySection := keysBySection(components)

	assert.ElementsMatch(t, []string{"otlp", "otlp/secondary"}, bySection["exporters"])
}

// TestDetectCommentedComponents_NoCommentedConfig ensures pure-prose comment
// blocks and fully active configs yield no components.
func TestDetectCommentedComponents_NoCommentedConfig(t *testing.T) {
	raw := `# This file configures the collector.
# Edit with care — see the runbook for details.
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
`
	components := engine.DetectCommentedComponents(raw)
	assert.Empty(t, components, "no commented-out components should be detected")
}
