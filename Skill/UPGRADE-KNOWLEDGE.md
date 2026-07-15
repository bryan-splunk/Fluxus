# Splunk OTel Collector Upgrade Knowledge Base — v0.120 → v0.153

## How to Use This Knowledge Base

This file is the reference used during both **Pre-Assessment** (scanning a config for applicable changes)
and **Apply Changes** (making those changes). Each entry contains:
- What to look for in the config
- The impact category
- Exactly what to do, with before/after YAML snippets

---

## Impact Categories

| Category | Meaning | Color |
|---|---|---|
| **P1 Breaking** | Startup failure or silent data loss if not fixed | 🔴 |
| **P2 Degrading** | Requires config changes, planning, or operational action | 🟡 |
| **P3 Advisory** | No immediate failure; cleanup or informational only | 🔵 |

---

## P1 Breaking Changes

These will cause collector startup failure or silent data loss if not addressed.

---

### P1-01 · routingprocessor Removed (0.134)

**Look for:** `routingprocessor:` in `processors:` block or `routing` in pipeline processor lists.

**Impact:** Collector fails to start.

**Action:** Migrate to the `routing` connector. The routing connector uses OTTL conditions to route
telemetry to named pipelines. It must be wired as an **exporter** in the input pipeline and as a
**receiver** in each output pipeline. Removing the processor without completing the pipeline rewiring
will still cause a startup failure.

```yaml
# BEFORE — processor (removed in 0.134):
processors:
  routing:
    from_attribute: X-Tenant
    table:
      - value: acme
        exporters: [otlp/acme]
      - value: beta
        exporters: [otlp/beta]

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, routing]  # ← routing was a processor
      exporters: [otlp/acme, otlp/beta]

# AFTER — connector (COMPLETE pipeline rewiring required):
# NOTE: match_once was removed in 0.120 (P1-21). Do NOT include it here.
connectors:
  routing:
    table:
      - condition: attributes["X-Tenant"] == "acme"
        pipelines: [traces/acme]
      - condition: attributes["X-Tenant"] == "beta"
        pipelines: [traces/beta]
    # default_pipelines routes unmatched spans:
    default_pipelines: [traces/default]

service:
  pipelines:
    traces/input:            # input pipeline — routing connector is the EXPORTER
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [routing]   # ← connector used as exporter here

    traces/acme:             # output pipeline — routing connector is the RECEIVER
      receivers: [routing]   # ← connector used as receiver here
      processors: []
      exporters: [otlp/acme]

    traces/beta:
      receivers: [routing]
      processors: []
      exporters: [otlp/beta]

    traces/default:
      receivers: [routing]
      processors: []
      exporters: [otlp/default]
```

> **Pipeline naming:** The routing connector routes to *pipeline names*, not exporter names.
> Old processor-style `exporters: [otlp/acme]` becomes `pipelines: [traces/acme]`.
> The output pipeline names must match exactly what is listed in the connector table.
>
> **Accuracy check:** After applying, verify the `service.pipelines` section contains:
> - The input pipeline with `routing` in its `exporters:` list
> - One output pipeline per destination with `routing` in its `receivers:` list
> - No remaining `routing:` entry under `processors:`

---

### P1-02 · sapm Receiver Removed (0.135)

**Look for:** `sapm:` in `receivers:` block.

**Impact:** Collector fails to start.

**Action:** Remove the `sapm` receiver definition and replace with the `otlp` receiver.

```yaml
# BEFORE:
receivers:
  sapm:
    endpoint: 0.0.0.0:7276

# AFTER:
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
```

---

### P1-03 · sapm Exporter Removed (0.147)

**Look for:** `sapm:` in `exporters:` block.

**Impact:** Collector fails to start.

**Action:** Remove the `sapm` exporter definition and replace with `otlphttp`.

```yaml
# BEFORE:
exporters:
  sapm:
    endpoint: https://ingest.us0.signalfx.com/v2/trace
    access_token: "${env:SPLUNK_ACCESS_TOKEN}"

# AFTER:
exporters:
  otlphttp:
    endpoint: https://ingest.us0.signalfx.com/v2/trace/otlp
    headers:
      X-SF-Token: "${env:SPLUNK_ACCESS_TOKEN}"
```

---

### P1-04 · signalfx Receiver Removed (deprecated 0.146, removed 0.153)

**Look for:** `signalfx:` in `receivers:` block, `signalfx` in any pipeline `receivers:` list.

**Impact:** Collector fails to start.

**Action:** Remove the `signalfx` receiver entirely. Replace with the `otlp` receiver.
Update any agents that were pushing to port 9943 — they must now emit OTLP.

```yaml
# BEFORE — signalfx receiver (REMOVED in 0.153):
receivers:
  signalfx:
    endpoint: 0.0.0.0:9943
    include_metadata: true

service:
  pipelines:
    metrics:
      receivers: [signalfx]
    logs:
      receivers: [signalfx]

# AFTER — use the OTLP receiver instead:
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

service:
  pipelines:
    metrics:
      receivers: [otlp]
    logs:
      receivers: [otlp]
```

**Also remove:** `access_token_passthrough:` if it was inside the signalfx receiver block (removed in 0.137).

Agents previously pushing to port 9943 must be reconfigured to send OTLP to port 4317 (gRPC) or
4318 (HTTP). The Splunk OTel agent and most supported SDKs support OTLP natively.

---

### P1-05 · access_token_passthrough Removed (0.137)

**Look for:** `access_token_passthrough:` in **any** of these locations:
- Inside a `signalfx:` receiver block
- Inside a `signalfx:` exporter block
- Inside a `splunk_hec:` exporter block

**Impact:** Collector fails to start if the field is present (removed as a recognized config key in 0.137).

**Action:** Remove the field entirely. Token passthrough is no longer a supported mechanism — use
`headers_setter` extension or inline `headers:` on the exporter for token propagation.

```yaml
# BEFORE — remove these fields wherever they appear:
receivers:
  signalfx:
    endpoint: 0.0.0.0:9943
    access_token_passthrough: true    # ← REMOVE

exporters:
  signalfx:
    access_token: "${env:SPLUNK_ACCESS_TOKEN}"
    access_token_passthrough: true    # ← REMOVE (also appears in signalfx exporter)

  splunk_hec:
    token: "${env:SPLUNK_HEC_TOKEN}"
    access_token_passthrough: true    # ← REMOVE (also appears in splunk_hec exporter)

# AFTER — no access_token_passthrough field anywhere:
exporters:
  signalfx:
    access_token: "${env:SPLUNK_ACCESS_TOKEN}"

  splunk_hec:
    token: "${env:SPLUNK_HEC_TOKEN}"
```

---

### P1-06 · OTLP Exporter — Deprecated batcher Block Removed (0.130)

**Look for:** `batcher:` key nested under any `otlp:` or `otlp/<name>:` exporter definition.

**Impact:** Collector fails to start.

**Action:** Remove the `batcher:` block. Use `sending_queue::batch` instead.

```yaml
# BEFORE (removed in 0.130):
exporters:
  otlp:
    batcher:
      enabled: true
      min_size_items: 100

# AFTER:
exporters:
  otlp:
    sending_queue:
      batch:
        min_size: 100
        flush_timeout: 200ms
```

---

### P1-07 · splunk_hec Exporter — batcher Block Removed (0.151)

**Look for:** `batcher:` key nested under any `splunk_hec:` exporter definition.

**Impact:** Collector fails to start.

**Action:** Remove the `batcher:` block. Use `sending_queue::batch` instead.

```yaml
# BEFORE (removed in 0.151):
exporters:
  splunk_hec:
    batcher:
      enabled: true
      min_size_items: 100

# AFTER:
exporters:
  splunk_hec:
    sending_queue:
      batch:
        min_size_items: 100
```

---

### P1-08 · kafka Exporter — Top-Level topic/encoding Removed (0.148)

**Look for:** `topic:` or `encoding:` as direct children of a `kafka:` exporter (not nested under `logs:`, `metrics:`, or `traces:`).

**Impact:** Collector fails to start.

**Action:** Remove the top-level `topic` and `encoding` fields. Use per-signal sub-keys instead.

```yaml
# BEFORE (deprecated since 0.124, removed 0.148):
exporters:
  kafka:
    topic: my-topic
    encoding: otlp_proto

# AFTER — per-signal fields:
exporters:
  kafka:
    logs:
      topic: my-logs-topic
      encoding: otlp_proto
    metrics:
      topic: my-metrics-topic
      encoding: otlp_proto
    traces:
      topic: my-traces-topic
      encoding: otlp_proto
```

---

### P1-09 · kafka Receiver — Removed Keys (0.141 / 0.144 / 0.147)

**Look for:** `topic:`, `exclude_topic:`, or `default_fetch_size:` as direct children of a `kafka:` receiver.

**Impact:** Collector fails to start.

**Action:** Remove these keys. Use `topics:` (list) and `exclude_topics:` (list) instead.

```yaml
# BEFORE (removed):
receivers:
  kafka:
    topic: my-topic           # REMOVED in 0.141/0.147 — use topics: [list]
    exclude_topic: bad-topic  # REMOVED in 0.147 — use exclude_topics: [list]
    default_fetch_size: 1048576  # REMOVED in 0.144 (Sarama-only)

# AFTER:
receivers:
  kafka:
    topics: [my-topic, other-topic]
    exclude_topics: [bad-topic]
    # Franz-go equivalent for fetch size:
    max_fetch_size: 50MiB
```

---

### P1-10 · sqlserver Receiver — Event Flag Renames (0.128)

**Look for:** `top_query_collection:` or `query_sample_collection:` nested under `sqlserver:` receiver.

**Impact:** Collector fails to start.

**Action:** Rename to the new `events:` sub-key syntax.

```yaml
# BEFORE (removed in 0.128):
receivers:
  sqlserver:
    top_query_collection:
      enabled: true
    query_sample_collection:
      enabled: true

# AFTER:
receivers:
  sqlserver:
    events:
      "db.server.top_query":
        enabled: true
      "db.server.query_sample":
        enabled: true
```

---

### P1-11 · postgresql Receiver — Query Collection Flags Removed (0.132)

**Look for:** `query_sample_collection:` or `top_query_collection:` under `postgresql:` receiver.

**Impact:** Collector fails to start.

**Action:** Remove these flags — both collections are now always enabled. If you need to disable
them, use a `filter` processor downstream.

```yaml
# REMOVED in 0.132:
receivers:
  postgresql:
    query_sample_collection:
      enabled: true/false   # REMOVED
    top_query_collection:
      enabled: true/false   # REMOVED
```

---

### P1-12 · kubeletstats / kubelet_stats — No-Op Config Sections Cause Startup Failure (0.136)

**Look for:** `kubeletstats:` or `kubelet_stats:` receiver with `metrics:` sub-block containing `enabled: false`
entries for metrics that are already disabled by default.

**Impact:** Collector fails to start. The Splunk distribution converter that silently dropped these no-op
sections was removed in 0.136.

**Action:** Remove any `enabled: false` entries for metrics that are already off by default.
Run `otelcol validate --config=config.yaml` to catch remaining issues.

```yaml
# REMOVE these no-op sections (causes startup failure in 0.136+):
receivers:
  kubeletstats:
    metrics:
      k8s.node.cpu.utilization:
        enabled: false   # already default-off — remove this
      k8s.pod.cpu.utilization:
        enabled: false   # already default-off — remove this
```

---

### P1-13 · filter / transform Processors — error_mode Default Changed (0.153)

**Look for:** Any `filter:` or `transform:` processor definition **without** an explicit `error_mode:` field.

**Impact:** Silent data loss. Errors that previously failed the batch now pass silently. Misconfigurations
in OTTL statements are no longer surfaced.

**Action:** Explicitly set `error_mode` on every filter and transform processor to make the behavior
intentional. Use `propagate` to restore pre-0.153 behavior.

```yaml
# To restore previous behavior:
processors:
  filter:
    error_mode: propagate   # was the default before 0.153
  transform:
    error_mode: propagate   # was the default before 0.153

# Or to explicitly adopt the new default:
processors:
  filter:
    error_mode: ignore
  transform:
    error_mode: ignore

# To revert via feature gate (temporary):
# --feature-gates=-processor.filter.defaultErrorModeIgnore
# --feature-gates=-processor.transform.defaultErrorModeIgnore
```

---

### P1-14 · OTTL — SetMap Error Handling (0.150)

**Look for:** Any `transform:` or `filter:` processor with OTTL `set()` calls targeting map or slice fields.

**Impact:** Silent data loss changed to active errors. Statements that were no-ops may now produce errors
that propagate or get ignored depending on `error_mode`.

**Action:** Review all `set()` calls on map or slice fields. Test in staging.

```yaml
# Review all set() calls on map or slice fields in
# your transform/filter pipelines.
# Statements that were previously no-ops may now
# produce errors that propagate (if error_mode: propagate)
# or be silently ignored (if error_mode: ignore).
```

---

### P1-15 · OTTL — Type-Strict Setters (0.150 + 0.153)

**Look for:** Any `transform:` processor with OTTL statements setting histogram-specific paths on
non-histogram data points, or any type-incorrect path assignments.

**Impact:** Errors on type-mismatched OTTL setters (previously silently ignored).

**Action:** Audit all OTTL transform/filter configs. Test in staging against real data.

```yaml
# Now returns an error (was silently ignored before 0.150):
set(explicit_bounds, [1.0])    # only valid on HistogramDataPoint
set(bucket_counts, [1, 2, 3])  # only valid on HistogramDataPoint

# Valid paths by data point type:
# NumberDataPoint:
#   value_double, value_int, exemplars
# HistogramDataPoint:
#   explicit_bounds, bucket_counts, count, sum, exemplars
# ExponentialHistogramDataPoint:
#   scale, zero_count, positive.*, negative.*, count, sum
# SummaryDataPoint:
#   quantile_values, count, sum
```

---

### P1-16 · prometheus Receiver — Removed Config Options (0.143 / 0.149)

**Look for:** `use_start_time_metric:`, `start_time_metric_regex:`, or `report_extra_scrape_metrics:`
under any `prometheus:` receiver.

**Impact:** Collector fails to start.

**Action:** Remove these keys. Use the `metricstarttime` processor for start time behavior.

```yaml
# Removed in 0.143:
receivers:
  prometheus:
    use_start_time_metric: true           # REMOVED — use metricstarttime processor
    start_time_metric_regex: ".*_start"   # REMOVED

# Removed in 0.149:
    report_extra_scrape_metrics: true     # REMOVED
    # Use PromConfig.ScrapeConfigs.ExtraScrapeMetrics

# Replacement for start time adjustment:
processors:
  metricstarttime: {}
```

---

### P1-17 · FluentD Permanently Removed from All Splunk Installers (0.144–0.145)

**Look for:** Any reference to `fluentforward` or `fluent_forward` receiver, or any note that FluentD
was being used for log collection.

**Impact:** FluentD is not installed; log collection pipeline breaks.

**Action:** Replace FluentD with the `file_log` receiver (formerly `filelog`).

```yaml
# FluentD removed from:
# - MSI (Windows)        0.144
# - Chocolatey           0.144
# - Standard installer   0.144
# - PowerShell script    0.145
# - RPM / DEB packages   0.145

# REPLACEMENT: use the file_log receiver
receivers:
  file_log:
    include: [/var/log/myapp/*.log]
    operators:
      - type: json_parser
```

---

### P1-18 · resourcedetection Processor — attributes Field Removed (0.142)

**Look for:** `attributes:` as a direct sub-key of any `resourcedetection:` or `resource_detection:` processor.

**Impact:** Collector fails to start.

**Action:** Remove the `attributes:` field. Use the standalone `resource` processor to filter/set attributes.

```yaml
# BEFORE (removed in 0.142):
processors:
  resourcedetection:
    detectors: [gcp, ec2]
    attributes: [cloud.region, cloud.account.id]

# AFTER — use 'resource' processor for attribute filtering:
processors:
  resource_detection:
    detectors: [gcp, ec2]
  resource:
    attributes:
      - key: cloud.region
        action: keep
```

---

### P1-19 · mysql / postgresql Receivers — Query Defaults Off (0.148)

**Look for:** `mysql:` or `postgresql:` receiver in use.

**Impact:** Silent data loss — query sample data stops being collected without config change.

**Action:** Explicitly re-enable if you need this data.

```yaml
# mysql: query_sample now off by default (0.148)
receivers:
  mysql:
    query_sample:
      enabled: true   # re-enable if needed

# postgresql: top_query and query_sample off by default (0.148)
receivers:
  postgresql:
    top_queries:
      enabled: true
    query_sample:
      enabled: true
```

---

### P1-20 · sending_queue::blocking Field Removed (0.129)

**Look for:** `blocking: true` or `blocking: false` inside a `sending_queue:` block of any exporter.

**Impact:** Collector fails to start. The `blocking` option was deprecated in 0.123 in favour of
`block_on_overflow` and removed in 0.129. Any exporter config that still has `blocking:` will fail
to parse.

**Action:** Rename the field.

```yaml
# BEFORE (causes startup failure in 0.129+):
exporters:
  otlp:
    sending_queue:
      blocking: true   # removed field

# AFTER:
exporters:
  otlp:
    sending_queue:
      block_on_overflow: true   # replacement field
```

---

## P2 Degrading Changes

These require config edits, planning, or operational action before or after upgrading.

---

### P2-01 · Kafka Receiver (Sarama) — Client ID Now Honoured (0.130)

**Look for:** Any `kafka:` receiver. Check if Kafka broker ACLs, monitoring, or audit logs
filter by client ID.

**Impact:** In the Sarama-based kafka RECEIVER, the `client_id` configuration was effectively
broken before 0.130 — it was always overridden to `"sarama"` regardless of what was set. In
0.130 the configuration is honoured and the default value is now `"otel-collector"`. This has
two consequences:

1. **Kafka ACL breakage (silent data loss):** If your Kafka brokers have ACLs granting access by
   client ID (e.g. `kafka-acls.sh --add --allow-principal ... --operation Write --cluster --client-id sarama`),
   those ACLs will no longer match after upgrade. The collector will authenticate successfully via
   SASL/mTLS, then be denied at the ACL layer. **No startup error** — the collector runs but produces
   and/or consumes nothing. This is the most common post-upgrade Kafka silent failure.
2. **Monitoring/audit breakage:** Dashboards and alerts filtering on `client.id = "sarama"` stop matching.

> **See also:** P2-34 covers the same client_id change for `kafkaexporter` and `kafkametricsreceiver`,
> which changed in the earlier 0.123 release.

**Action:** Choose one approach:

```yaml
# OPTION A — Pin the old client ID (no Kafka ACL changes needed):
receivers:
  kafka:
    client_id: sarama   # preserves legacy ACL match

exporters:
  kafka:
    client_id: sarama   # preserves legacy ACL match

# OPTION B — Update Kafka ACLs to allow "otel-collector" (recommended long-term):
# On your Kafka cluster admin:
#   kafka-acls.sh --bootstrap-server <host>:9092 \
#     --add --allow-principal User:<collector-user> \
#     --operation Write --topic <topic> \
#     --client-id otel-collector
# Then remove the old "sarama" ACL entry.
# Also update any dashboards/alerts that filter on client.id.
```

> **Per-file README note when P2-01 is applied or flagged:** Add a prominent warning:
> "⚠️ Kafka ACL ACTION REQUIRED — verify broker ACLs allow client.id 'otel-collector' (or pin
> client_id: sarama). Failure to do so causes silent Kafka connectivity loss after upgrade."

---

### P2-02 · Kafka Exporter — Batching Requires Explicit metadata_keys (0.148)

**Look for:** Any `kafka:` exporter with both `sending_queue::batch::enabled: true` and
`include_metadata_keys:` set.

**Impact:** Metadata-based partitioning silently stops working.

**Action:** Explicitly add `metadata_keys` inside the `partition:` block.

```yaml
# Before 0.148 (implicit wiring):
exporters:
  kafka:
    sending_queue:
      batch:
        enabled: true
    include_metadata_keys: [tenant_id]

# After 0.148 (explicit metadata_keys required):
exporters:
  kafka:
    sending_queue:
      batch:
        enabled: true
        partition:
          metadata_keys: [tenant_id]   # must match include_metadata_keys
    include_metadata_keys: [tenant_id]
```

---

### P2-03 · kafka_metrics Receiver — Sarama Removed (0.152)

**Look for:** `kafkametrics:` or `kafka_metrics:` receiver.

**Impact:** If the `+receiver.kafkametricsreceiver.UseFranzGo` gate was set, remove it — Franz-go is
now mandatory. Also rename `kafkametrics` → `kafka_metrics` (alias still works in 0.153 but will be removed).

**Action:** Remove the feature gate flag if present. Rename receiver if using old name.

```yaml
# Remove this flag from startup if present:
# --feature-gates=+receiver.kafkametricsreceiver.UseFranzGo

# Rename the receiver (alias still available in 0.153):
# kafkametrics  →  kafka_metrics
```

---

### P2-04 · cumulativetodelta Processor — max_staleness Default Changed (0.142)

**Look for:** `cumulativetodelta:` processor without an explicit `max_staleness:` field.

**Impact:** Series not seen for more than 1 hour are dropped from state (prevents unbounded memory
growth). Previously the default was infinite (0). This is intentional but may cause gaps in
long-running series.

**Action:** If you need infinite retention, explicitly set `max_staleness: 0`.

```yaml
processors:
  cumulativetodelta:
    max_staleness: 0   # restore infinite retention (pre-0.142 behavior)
    # New default if omitted: max_staleness: 1h
```

---

### P2-05 · resourcedetection Processor — Cloud Platform Values Changed (0.147)

**Look for:** Any `resource_detection` / `resourcedetection` processor with `azure` or `gcp` detectors.
Also look for downstream OTTL rules, dashboards, or filters that reference `cloud.platform`.

**Impact:** Azure cloud.platform values changed; `faas.id` attribute replaced. Dashboards or filters
matching the old string values break.

**Action:** Update any OTTL conditions, dashboard filters, or downstream logic referencing these values.

```yaml
# Before 0.147:
#   cloud.platform: azure_eks
#   cloud.platform: azure_vm
#   faas.id: <value>

# After 0.147 (aligns with OTel semconv v1.39):
#   cloud.platform: azure.eks
#   cloud.platform: azure.vm
#   faas.instance: <value>   (replaces faas.id)

# Feature gates now removed (were always-on):
# processor.resourcedetection.propagateerrors
# processor.resourcedetection.removeGCPFaasID
```

---

### P2-06 · tail_sampling Processor — Invert Decisions Permanently Disabled (0.144 / 0.152)

**Look for:** Any `tail_sampling:` processor with `invert_match: true` on any policy.

**Impact:** Invert decisions are permanently disabled. `invert_match: true` is a no-op as of 0.144
and the gate was stabilized in 0.152.

**Action:** Migrate invert-match logic to dedicated drop policies.

```yaml
# OLD (no longer works):
# invert_match: true  on any policy

# NEW: use a dedicated drop policy instead.
# See the tail_sampling processor documentation
# for drop policy syntax.

# Remove this flag if present:
# --feature-gates=processor.tailsamplingprocessor.disableinvertdecisions
```

---

### P2-07 · prometheus Receiver — Start Time No Longer Adjusted (0.140)

**Look for:** Any `prometheus:` receiver. Check if downstream systems depend on adjusted start
times for rate calculations.

**Impact:** Cumulative metrics may report incorrect rates if the downstream system expects the
Prometheus start-time adjustment behavior.

**Action:** Use the `metricstarttime` processor if start time adjustment is required.

```yaml
# Feature gates now permanently removed (remove from startup if set):
# receiver.prometheusreceiver.RemoveStartTimeAdjustment
# receiver.prometheusreceiver.UseCreatedMetric
# receiver.prometheusreceiver.EnableNativeHistograms
# receiver.prometheusreceiver.RemoveLegacyResourceAttributes
# receiver.prometheusreceiver.RemoveReportExtraScrapeMetricsConfig

# If you need start time adjustment, add:
processors:
  metricstarttime: {}
# ...and add it to your prometheus pipeline processors list.
```

---

### P2-08 · docker_observer / docker_stats — Docker API Version Upgraded (0.141 / 0.142)

**Look for:** `docker_observer:` extension or `docker_stats:` receiver.

**Impact:** If your Docker daemon is older than API 1.44, the receiver/observer will fail to connect.

**Action:** Pin the API version to your Docker daemon version if needed.

```yaml
# docker_observer (receiver_creator discovery) — changed in 0.141:
extensions:
  docker_observer:
    api_version: "1.24"   # pin to older version if needed

# docker_stats receiver — changed in 0.142:
receivers:
  docker_stats:
    api_version: "1.24"   # pin to older version if needed

# Both default to Docker API 1.44 from these versions on.
# Minimum supported API version has NOT changed; only the default.
```

---

### P2-09 · mongodb Receiver — Schema Change (0.147)

**Look for:** `mongodb:` receiver. Check dashboards that query by `database` resource attribute.

**Impact:** The `database` resource attribute is removed. Dashboards or OTTL rules joining on it break.

**Action:** Update dashboards and OTTL rules to use `db.namespace` metric attribute instead.

```yaml
# Schema change in 0.147:
# BEFORE: each database = separate resource with 'database' resource attribute
# AFTER:  single resource per MongoDB server
#
# 'database' resource attribute REMOVED.
# Database is now a metric-level attribute: db.namespace
#
# Added: service.instance.id resource attribute (UUID v5 from host:port)
#
# Update any dashboards that joined on the 'database' resource attribute.
```

---

### P2-10 · windows_event_log Receiver — event_data Format Change (0.148)

**Look for:** `windows_event_log:` or `windowseventlog:` receiver. Check downstream OTTL rules,
Splunk searches, or dashboards that parse `event_data`.

**Impact:** `event_data` field structure changed from array to flat map. Downstream parsing breaks.

**Action:** Audit all OTTL/filter rules, Splunk searches, and dashboards that inspect `event_data`.
Use `event_data_format: array` to restore the old format if needed.

```yaml
# Before 0.148:
# body["event_data"] = [{"ProcessId": "1234"}]

# After 0.148+:
# body["event_data"]["ProcessId"] = "1234"

# To restore previous array format:
receivers:
  windows_event_log:
    event_data_format: array

# New body keys also added in 0.148 (informational):
# body["rendering_info"]["culture/channel/provider/message"]
# body["user_data"][...] (parsed key-value map, alternative to event_data)
```

---

### P2-11 · http_check Receiver — Timing Metrics Now in Nanoseconds (0.153)

**Look for:** `http_check:` or `httpcheck:` receiver. Check dashboards with thresholds on timing metrics.

**Impact:** Timing metric values scale from milliseconds to nanoseconds. Dashboard thresholds become
wrong by a factor of 1,000,000.

**Action:** Update dashboard thresholds and alert conditions.

```yaml
# Affected metrics (update thresholds):
# httpcheck.dns.lookup.duration
# httpcheck.client.connection.duration
# httpcheck.tls.handshake.duration
# httpcheck.client.request.duration
# httpcheck.response.duration

# Old: 500µs = 0 (truncated to 0ms)
# New: 500µs = 500,000 (nanoseconds)
# Old: 1.5ms = 1 (ms)
# New: 1.5ms = 1,500,000 (ns)
```

---

### P2-12 · sqlserver Receiver — Event Name Changes (0.126)

**Look for:** `sqlserver:` receiver. Check downstream OTTL rules, Splunk searches, or dashboards
that reference the old event names or attributes.

**Impact:** Downstream processing that references old event names or `sqlserver.username` breaks.

**Action:** Update OTTL rules, searches, and dashboards.

```yaml
# Event name changes in 0.126:
# "top query"    →  "db.server.top_query"
# "query sample" →  "db.server.query_sample"

# Attribute renamed:
# sqlserver.username  →  user.name (in query sample events)

# Query sample body removed entirely.
# Data previously in the body is now in attributes.

# Also in 0.139: lookback_time requires 's' suffix
# lookback_time: 30   →  lookback_time: 30s
```

---

### P2-13 · prometheusremotewrite Exporter — Remote Write 2.0 (0.142)

**Look for:** `prometheusremotewrite:` exporter.

**Impact:** The exporter now uses Remote Write 2.0 rc.4, which is wire-incompatible with Prometheus
older than 3.8.0.

**Action:** Upgrade your Prometheus endpoint to 3.8.0+ before upgrading the collector.

```yaml
# Updated to Remote Write 2.0 rc.4 in 0.142.
# Requires Prometheus 3.8.0+ as receiving endpoint.
# Wire-protocol incompatible with older Prometheus.
# Upgrade Prometheus BEFORE upgrading the collector.

# Feature gate removed in 0.151 — remove if set:
# pkg.translator.prometheus.NormalizeName
```

---

### P2-14 · signalfx Exporter — Default URL Domain Change (0.151)

**Look for:** `signalfx:` exporter using `realm:` without explicit `api_url`/`ingest_url`. Also check
environment variables `SPLUNK_API_URL` / `SPLUNK_INGEST_URL`.

**Impact:** Default endpoints change. Firewall allowlists that restrict to `*.signalfx.com` will block
the collector.

**Action:** Update firewall allowlists and environment variables.

```yaml
# Before 0.151 (realm: us0):
# api_url:    https://api.us0.signalfx.com
# ingest_url: https://ingest.us0.signalfx.com

# After 0.151:
# api_url:    https://api.us0.observability.splunkcloud.com
# ingest_url: https://ingest.us0.observability.splunkcloud.com

# ACTION:
# 1. Add *.observability.splunkcloud.com to firewall allowlists
# 2. Update SPLUNK_API_URL and SPLUNK_INGEST_URL env vars
# 3. Update header comments referencing signalfx.com
```

---

### P2-15 · IIS Receiver — Application Pool Metrics Enabled by Default (0.131)

**Look for:** `iis:` receiver in any config running on a Windows host with many IIS application pools.

**Impact:** Two new metrics enabled by default increase metric volume and billing on environments with
many application pools.

**Action:** Explicitly disable if the additional volume is not desired.

```yaml
# Now enabled by default in 0.131:
# iis.application_pool.state
# iis.application_pool.uptime

# To disable:
receivers:
  iis:
    metrics:
      iis.application_pool.state:
        enabled: false
      iis.application_pool.uptime:
        enabled: false
```

---

### P2-16 · Windows MSI Download URL Changed (0.151)

**Look for:** Any automation scripts, CI/CD pipelines, or infrastructure-as-code that download the
Splunk OTel Collector MSI.

**Impact:** Old download URL returns 404 or redirects. Automated installs fail.

**Action:** Update all references.

```yaml
# OLD URL in installer scripts:
# dl.signalfx.com

# NEW URL:
# dl.observability.splunkcloud.com
```

---

### P2-17 · Internal Metrics — service_name / service_instance_id / service_version Labels Removed (0.149)

**Look for:** Any dashboard queries, alerts, or OTTL rules that filter individual metrics by
`service_name`, `service_instance_id`, or `service_version`.

**Impact:** These labels no longer appear on individual metric series. Queries that join on them return
no results.

**Action:** Update queries to use `target_info` for service label lookups.

```yaml
# Before 0.149:
# service_name, service_instance_id, service_version
# stamped on every individual internal metric.

# After 0.149:
# These attributes only appear in target_info metric.
# Update dashboard queries that join on these labels.
```

---

### P2-18 · prometheus Receiver — Legacy Resource Attributes Renamed (0.126)

**Look for:** Any `prometheus:` receiver block. Also check dashboards, alerts, or OTTL rules that
reference `net.host.name`, `net.host.port`, or `http.scheme` as resource attributes from Prometheus
scrape targets.

**Impact:** The `receiver.prometheusreceiver.RemoveLegacyResourceAttributes` feature gate became
**beta (enabled by default)** in 0.126. The following resource attributes are renamed on all metrics
from Prometheus receivers:

| Old attribute | New attribute |
|---|---|
| `net.host.name` | `server.address` |
| `net.host.port` | `server.port` |
| `http.scheme` | `url.scheme` |

Dashboards and alerts filtering on the old names return no data after upgrade. The gate became
**stable** in 0.129, making the rename permanent and the gate flag non-functional from 0.129+.

**Action:** Update dashboards, alerts, and OTTL rules to use the new attribute names. If you must
temporarily preserve old names during the transition window (0.126–0.128 only), disable the gate:

```yaml
# Temporary rollback (0.126–0.128 ONLY — gate is stable/locked from 0.129+):
# --feature-gates=-receiver.prometheusreceiver.RemoveLegacyResourceAttributes
```

---

## P3 Advisory Changes

These do not cause immediate failures. Migration is recommended before aliases are removed in a future release.

---

### P3-01 · Component Renames — snake_case Migration

**Look for:** Any component name in the old format (see table below).

**Impact:** Deprecated aliases still work in 0.153 but will be removed in a future release. No immediate failure.

**Action:** Rename all occurrences in definitions AND all pipeline references AND commented-out blocks.

| Type | Old Name | New Canonical Name | Since |
|---|---|---|---|
| receiver | `filelog` | `file_log` | 0.149 |
| receiver | `fluentforward` | `fluent_forward` | 0.151 |
| receiver | `hostmetrics` | `host_metrics` | 0.151 |
| receiver | `k8sobjects` | `k8s_objects` | 0.151 |
| receiver | `kubeletstats` | `kubelet_stats` | 0.152 |
| receiver | `kafkametrics` | `kafka_metrics` | 0.152 |
| receiver | `httpcheck` | `http_check` | 0.150 |
| receiver | `tcpcheck` | `tcp_check` | 0.152 |
| receiver | `tcplog` | `tcp_log` | 0.150 |
| receiver | `udplog` | `udp_log` | 0.150 |
| receiver | `windowseventlog` | `windows_event_log` | 0.150 |
| receiver | `sshcheck` | `ssh_check` | 0.151 |
| receiver | `tlscheck` | `tls_check` | 0.150 |
| receiver | `filestats` | `file_stats` | 0.151 |
| receiver | `namedpipe` | `named_pipe` | 0.150 |
| receiver | `cloudfoundry` | `cloud_foundry` | 0.152 |
| receiver | `azureeventhub` | `azure_event_hub` | 0.145 |
| receiver | `mongodbatlas` | `mongodb_atlas` | 0.145 |
| processor | `k8sattributes` | `k8s_attributes` | 0.146 |
| processor | `metricstransform` | `metrics_transform` | 0.152 |
| processor | `logdedup` | `log_dedup` | 0.151 |
| processor | `resourcedetection` | `resource_detection` | 0.153 |
| connector | `spanmetrics` | `span_metrics` | 0.151 |
| connector | `servicegraph` | `service_graph` | 0.151 |
| exporter | `loadbalancing` | `load_balancing` | 0.153 |

**Not renamed** (confirmed clean through v0.153): `windowsservices`, `vcenter`, `jaeger`, `otlp`,
`zipkin`, `prometheus`, `smartagent`, `nop`, `splunk_hec`, `otlphttp`, `signalfx` (exporter),
`batch`, `memory_limiter`, `resource`, `attributes`, `filter`.

---

### P3-02 · Kafka Deprecated No-Op Fields (0.153)

**Look for:** `resolve_canonical_bootstrap_servers_only:` or `auth.sasl.version:` under kafka receiver or exporter.

**Impact:** Fields are no-ops (have no effect). Will be removed in a future release.

**Action:** Remove them now to avoid a future breaking change.

```yaml
# DEPRECATED in 0.153 (no-ops — remove to avoid future failure):
kafka:
  resolve_canonical_bootstrap_servers_only: true   # Franz-go has no equivalent
  auth:
    sasl:
      version: 1   # Franz-go negotiates SASL handshake version automatically
```

---

### P3-03 · k8s_attributes Processor — New Semconv Feature Gates (0.146)

**Look for:** `k8sattributes:` or `k8s_attributes:` processor.

**Impact:** Informational. New alpha feature gates available. `otelcol.k8s.pod.association` metric
disabled by default in 0.151 until pod_identifier is properly calculated.

**Action:** No config change required unless you want to opt into new semantic conventions.

```yaml
# New gates in 0.146 (alpha, opt-in):
# processor.k8sattributes.EmitV1K8sConventions
#   → enables k8s.<type>.label.<name> (singular)
# processor.k8sattributes.DontEmitV0K8sConventions
#   → disables k8s.<type>.labels.<name> (plural)
#
# allowLabelsAnnotationsSingular gate (0.135) deprecated in 0.146.
# otelcol.k8s.pod.association metric disabled by default (0.151).
```

---

### P3-04 · kubelet_stats Receiver — Deprecated Attributes Disabled by Default (0.150)

**Look for:** `kubeletstats:` or `kubelet_stats:` receiver. Check if any downstream dashboards or
OTTL rules reference the listed attributes.

**Impact:** Attributes stop being emitted. Dashboards that joined on them show empty results.

**Action:** Re-enable in config if still needed; plan to migrate to replacement attributes.

```yaml
# Disabled by default in 0.150+:
#   aws.volume.id
#   fs.type
#   gce.pd.name
#   glusterfs.endpoints.name
#   glusterfs.path
#   partition
# These will be removed in a future release.
# Re-enable them explicitly in config if still needed.
```

---

### P3-05 · jaeger Receiver — DisableRemoteSampling Gate Removed (0.153)

**Look for:** `--feature-gates=receiver.jaeger.DisableRemoteSampling` in service startup flags.

**Impact:** Startup warning if flag is present. Remote sampling is permanently disabled.

**Action:** Remove the flag from startup arguments. If you relied on remote sampling, configure
an alternative (e.g., tail_sampling processor or probabilistic sampler).

```yaml
# Remove this flag if present:
# --feature-gates=receiver.jaeger.DisableRemoteSampling
```

---

### P3-06 · clickhouse Exporter — json Feature Gate Removed (0.153)

**Look for:** `--feature-gates=+clickhouse.json` in service startup flags.

**Impact:** Startup warning if flag is present.

**Action:** Remove the flag; set `json: true` directly in the clickhouse exporter config.

```yaml
# Remove from startup:
# --feature-gates=+clickhouse.json

# Set directly in config:
exporters:
  clickhouse:
    json: true
```

---

### P3-07 · Config Debug Endpoint Removed (0.142)

**Look for:** Any scripts, tooling, or documentation referencing `http://localhost:55554/debug/configz`.

**Impact:** That endpoint returns 404.

**Action:** Use the zpages extension instead.

```yaml
# Replacement via zpages extension:
# http://localhost:55679/debug/expvarz

# Get effective config (bash):
curl http://localhost:55679/debug/expvarz --silent \
  | jq -r '.["splunk.config.effective"]'

# Get initial config (bash):
curl http://localhost:55679/debug/expvarz --silent \
  | jq -r '.["splunk.config.initial"]'

# PowerShell:
(Invoke-WebRequest http://localhost:55679/debug/expvarz).Content \
  | ConvertFrom-Json \
  | Select-Object -ExpandProperty "splunk.config.effective"
```

---

### P3-08 · splunk_otlp_histograms Converter Removed (0.148)

**Look for:** Any config that relies on the automatic `splunk_otlp_histograms` resource attribute
being injected by a Splunk config converter.

**Impact:** Attribute is no longer auto-injected. Queries filtering on this attribute return no results.

**Action:** Add the attribute manually if needed.

```yaml
# If you still need this attribute:
processors:
  resource:
    attributes:
      - key: splunk_otlp_histograms
        value: "true"
        action: upsert
```

---

### P3-09 · New OTTL Functions Available Since 0.126

**Impact:** Informational — new capabilities available.

```yaml
# Coalesce (0.150): first non-nil value from list
set(attributes["user"],
  Coalesce([attributes["user.id"],
             attributes["enduser.id"],
             "unknown"]))

# Base64Encode (0.146)
set(attributes["encoded"], Base64Encode(body))

# IsInCIDR (0.146): check if IP in CIDR range
where IsInCIDR(attributes["client.ip"], "10.0.0.0/8")

# Bool (0.143): convert to boolean
set(attributes["flag"], Bool(attributes["flag_str"]))

# delete_index: remove item from array (0.145)
delete_index(attributes["items"], 0)

# TrimPrefix / TrimSuffix (0.139)
set(name, TrimPrefix(name, "prefix_"))
```

---

### P3-10 · truncate_all OTTL Function — UTF-8 Safe by Default (0.148)

**Look for:** Any OTTL `truncate_all()` call in transform/filter processors.

**Impact:** Results may be slightly shorter than the limit to avoid splitting multi-byte characters.
Generally a safe behavior change.

**Action:** If you need the old byte-level truncation behavior, pass `false` as the third argument.

```yaml
# New default (utf8_safe: true):
truncate_all(attributes, 100)
# → may produce strings shorter than 100 bytes

# To restore old byte-level truncation:
truncate_all(attributes, 100, false)
```

---

## Feature Gate Removals

Remove these flags from all service startup arguments (launch scripts, systemd unit files, Windows
service registry, Helm chart values, etc.):

| Gate | Removed In | Notes |
|---|---|---|
| `exporter.kafkaexporter.UseFranzGo` | 0.144 | Franz-go is now mandatory |
| `receiver.kafkareceiver.UseFranzGo` | 0.144 | Franz-go is now mandatory |
| `telemetry.disableHighCardinalityMetrics` | 0.144 | Removed |
| `service.noopTracerProvider` | 0.144 | Removed |
| `processor.resourcedetection.removeGCPFaasID` | 0.147 | Now always on |
| `processor.resourcedetection.propagateerrors` | 0.147 | Now always on |
| `processor.transform.ConvertBetweenSumAndGaugeMetricContext` | 0.150 | Removed |
| `pkg.translator.prometheus.NormalizeName` | 0.151 | Removed |
| `receiver.jaeger.DisableRemoteSampling` | 0.153 | Remote sampling now permanently off |
| `+clickhouse.json` | 0.153 | Use `json: true` in config instead |
| `processor.tailsamplingprocessor.disableinvertdecisions` | 0.152 | Stabilized — invert now always off |
| `receiver.prometheusreceiver.EnableNativeHistograms` | 0.151 | Now always on |
| `receiver.prometheusreceiver.RemoveStartTimeAdjustment` | 0.151 | Now always on |
| `receiver.prometheusreceiver.UseCreatedMetric` | 0.151 | Now always on |
| `receiver.prometheusreceiver.RemoveLegacyResourceAttributes` | 0.129 | Stabilized (gate flag non-functional; permanently removed in 0.151) |
| `receiver.prometheusreceiver.RemoveReportExtraScrapeMetricsConfig` | 0.151 | Now always on |
| `+receiver.kafkametricsreceiver.UseFranzGo` | 0.152 | Stabilized; gate removed in 0.154 |

---

## Removed SmartAgent Monitors

Any config referencing these SmartAgent monitors must migrate to the replacement.

| Removed Monitor / Plugin | Recommended Replacement | Removed In |
|---|---|---|
| `sapm` exporter | `otlphttp` exporter | 0.147 |
| `sapm` receiver | `otlp` receiver | 0.135 |
| `routingprocessor` | `routing` connector | 0.134 |
| `collectd/apache` | Apache receiver | 0.149 |
| `collectd/cpufreq` | `host_metrics` receiver | 0.149 |
| `collectd/memory` | `host_metrics` receiver (memory scraper) | 0.149 |
| `collectd/opcache` | PHP SDK for OpenTelemetry | 0.149 |
| `collectd/php-fpm` | PHP SDK for OpenTelemetry | 0.149 |
| `collectd/processes` | `host_metrics` receiver (process scraper) | 0.149 |
| `collectd/systemd` | systemd receiver | 0.149 |
| `collectd/uptime` | `host_metrics` receiver (system scraper) | 0.149 |
| `collectd/zookeeper` | zookeeper receiver | 0.149 |
| `smartagent/ntp` | NTP receiver | 0.149 |
| `smartagent/postgresql` | postgresql receiver | 0.149 |
| `collectd/protocols` | `host_metrics` receiver (network scraper) | 0.149 |
| `cadvisor` | prometheus receiver | 0.144 |
| `collectd/chrony` | chrony receiver | 0.144 |
| `collectd/cpu` | cpu monitor | 0.144 |
| `collectd/couchbase` | prometheus receiver | 0.144 |
| `haproxy` plugin | haproxy receiver | 0.144 |
| `heroku` plugin | resource_detection processor (heroku) | 0.144 |
| `kubelet-stats` / `kubelet-metrics` | `kubelet_stats` receiver | 0.144 |
| `mongodbatlas` monitor | `mongodb_atlas` receiver | 0.144 |
| `nagios` | No replacement | 0.144 |
| `collectd/nginx` | nginx receiver | 0.144 |
| `collectd/rabbitmq` | RabbitMQ receiver | 0.144 |
| `collectd/redis` | Redis receiver | 0.144 |
| `windows-legacy` | `host_metrics` + `windowsperfcounters` receivers | 0.144 |
| `collectd/spark` | Apache Spark receiver | 0.144 |
| `collectd/activemq` | `jmxreceiver` (target: activemq) | 0.131 |
| `collectd/cassandra` | `jmxreceiver` (target: cassandra) | 0.131 |
| `collectd/hadoop` | `jmxreceiver` (target: hadoop) | 0.131 |
| `collectd/kafka` | `jmxreceiver` (target: kafka) | 0.131 |
| `collectd/kafka-consumer` | `jmxreceiver` (target: kafka) | 0.131 |
| `collectd/kafka-producer` | `jmxreceiver` (target: kafka) | 0.131 |
| `collectd/solr` | `jmxreceiver` (target: solr) | 0.131 |
| `collectd/tomcat` | `jmxreceiver` (target: tomcat) | 0.131 |
| `smartagent/jaeger-grpc` | jaeger receiver (grpc protocol) | 0.131 |
| `collectd/jenkins` | Jenkins OpenTelemetry plugin | 0.144 |
| FluentD (all installers) | `file_log` receiver | 0.144–0.145 |
| `ecs_task_observer` extension | None (removed upstream) | 0.140 |
| `migratecheckpoint` command | None (FluentD sidecar gone) | 0.139 |

---

## Common Pre-Existing Issues to Flag

Report these if found, regardless of whether they are caused by the upgrade:

| Issue | Where to look | Severity |
|---|---|---|
| Hardcoded credentials | Any receiver `basic_auth.password` field | Security |
| Unused receiver definitions (defined but not in any pipeline) | Compare `receivers:` block vs `service.pipelines` | Cleanup |
| Unused processor definitions (defined but not in any pipeline) | Compare `processors:` block vs `service.pipelines` | Cleanup |
| SmartAgent monitor with no replacement listed | `smartagent/<name>` not in the removed monitors table | Investigate |
| `signalfx` receiver still referenced in any pipeline | Gateway or older agent configs | P1 Breaking |
| Feature gate flag referencing a removed gate | Service startup arguments | Cleanup |

---

## Validation Commands

```bash
# Validate config before deploying:
otelcol validate --config=/etc/otelcol/config.yaml

# Check for deprecated component name warnings:
otelcol --config=/etc/otelcol/config.yaml 2>&1 | grep -i "deprecated"

# Linux: verify collector service health post-upgrade:
systemctl status splunk-otel-collector

# Docker: test config validity:
docker run --rm -v $(pwd)/config.yaml:/etc/otelcol/config.yaml \
  quay.io/signalfx/splunk-otel-collector:0.153.0 \
  otelcol validate --config=/etc/otelcol/config.yaml
```

```powershell
# Windows: validate config before starting:
& "C:\Program Files\Splunk\OpenTelemetry Collector\otelcol.exe" validate --config="<path-to-config>"

# Check service status:
Get-Service -Name splunk-otel-collector | Select-Object Status, StartType

# View effective config via zpages:
(Invoke-WebRequest http://localhost:55679/debug/expvarz).Content | ConvertFrom-Json | Select-Object -ExpandProperty "splunk.config.effective"

# Watch for deprecated component warnings:
Get-Content "C:\ProgramData\Splunk\OpenTelemetry Collector\*.log" -Wait | Select-String "deprecated|error|warn"

# Check service startup flags for removed feature gates:
Get-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Services\splunk-otel-collector" | Select-Object ImagePath
```

---

## NEW CHANGES — v0.120–v0.125

These entries extend coverage to collectors running v0.120.0 and above.
All existing C-/I-/CA- entries remain valid for the full v0.120–v0.153 range.

---

## P1 Breaking Changes — v0.120–v0.125

---

### P1-21 · routing connector match_once Removed (0.120)

**Look for:** `match_once:` key inside a `routing:` block under `connectors:`.

**Impact:** Collector fails to start. The `match_once` parameter was removed from the routing
connector in 0.120. If your config was already migrated from `routingprocessor` to the
`routing` connector (see P1-01) and includes `match_once:`, it will cause a startup failure
when upgrading to 0.120+.

**Action:** Remove the `match_once:` field entirely. The routing connector now always evaluates
all table entries (previous `match_once: false` behavior is the only supported mode).

```yaml
# BEFORE — causes startup failure in 0.120+:
connectors:
  routing:
    match_once: true    # REMOVE this field
    table:
      - condition: attributes["env"] == "prod"
        pipelines: [traces/prod]
      - condition: attributes["env"] == "dev"
        pipelines: [traces/dev]

# AFTER — remove match_once entirely:
connectors:
  routing:
    table:
      - condition: attributes["env"] == "prod"
        pipelines: [traces/prod]
      - condition: attributes["env"] == "dev"
        pipelines: [traces/dev]
    default_pipelines: [traces/default]
```

---

### P1-22 · signalfx Exporter translation_rules Removed (0.121)

**Look for:** `translation_rules:` key inside a `signalfx:` exporter block.

**Impact:** Collector fails to start. The `translation_rules` config option was removed from
the `signalfx` exporter in 0.121. Metric transformations must now use the `transform` processor.

**Action:** Remove the `translation_rules:` block from the signalfx exporter and replace with
a `transform` processor. See the
[translation rules migration guide](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/signalfxexporter/docs/translation_rules_migration_guide.md).

```yaml
# BEFORE — causes startup failure in 0.121+:
exporters:
  signalfx:
    access_token: "${env:SPLUNK_ACCESS_TOKEN}"
    translation_rules:        # REMOVE this block
      - action: rename_metrics
        mapping:
          old_metric: new_metric

# AFTER — use transform processor instead:
processors:
  transform:
    metric_statements:
      - context: metric
        statements:
          - set(name, "new_metric") where name == "old_metric"

exporters:
  signalfx:
    access_token: "${env:SPLUNK_ACCESS_TOKEN}"

service:
  pipelines:
    metrics:
      processors: [..., transform]
      exporters: [signalfx]
```

---

### P1-23 · min_size_items / max_size_items Removed from Batch Config (0.123)

**Look for:** `min_size_items:` or `max_size_items:` in any `batch:` block under `sending_queue:`,
or in top-level `processors: batch:` config.

**Impact:** Collector fails to start. These fields were deprecated in 0.121 and removed in 0.123.

**Action:** Replace with `min_size:` and `max_size:` respectively.

```yaml
# BEFORE — causes startup failure in 0.123+:
exporters:
  otlp:
    sending_queue:
      batch:
        min_size_items: 100    # REMOVED
        max_size_items: 1000   # REMOVED

processors:
  batch:
    send_batch_size: 1000
    # Note: min_size_items/max_size_items in batch processor config
    # are also removed — use send_batch_size / send_batch_max_size instead

# AFTER:
exporters:
  otlp:
    sending_queue:
      batch:
        min_size: 100      # replacement for min_size_items
        max_size: 1000     # replacement for max_size_items
```

---

### P1-24 · service::telemetry::address Silently Ignored (0.123)

**Look for:** `address:` key directly under `service: telemetry:` (the legacy flat telemetry
address format, NOT inside `metrics:` or `traces:`).

**Impact:** Collector starts without error but emits **no internal telemetry metrics** — a
silent operational failure. The `telemetry.disableAddressFieldForInternalTelemetry` feature
gate was promoted to beta in 0.123, causing the deprecated `service::telemetry::address`
field to be ignored.

**Action:** Migrate to the new telemetry configuration format.

```yaml
# BEFORE — silently ignored in 0.123+; no internal metrics emitted:
service:
  telemetry:
    address: 0.0.0.0:8888   # IGNORED — causes silent telemetry loss

# AFTER — use the full readers format (official migration per 0.123 release notes):
service:
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 0.0.0.0
                port: 8888
                without_scope_info: true
                without_type_suffix: true
                without_units: true

# SIMPLER ALTERNATIVE — metrics: address: is also accepted (but itself deprecated):
# service:
#   telemetry:
#     metrics:
#       address: 0.0.0.0:8888

# NOTE: The telemetry.disableAddressFieldForInternalTelemetry gate became STABLE
# in 0.128, meaning --feature-gates=-telemetry.disableAddressFieldForInternalTelemetry
# FAILS collector startup from 0.128+. There is no gate-based workaround — migrate the config.
```

---

### P1-25 · transform Processor Mixed Config Style (0.124)

**Look for:** A `transform:` processor block that contains both a `metric_statements:/
log_statements:/trace_statements:` list (Basic Config) AND a `statements:` block at the
top processor level (Advanced Config).

**Impact:** Collector fails to start. The `transform` processor now requires exactly one
config style per processor instance — Basic Config and Advanced Config cannot be used together.

**Action:** Choose one style and migrate the other. Advanced Config uses top-level `statements:`
with context hints; Basic Config uses signal-specific keys (`metric_statements:`, etc.).

```yaml
# BEFORE — mixing styles causes startup failure in 0.124+:
processors:
  transform:
    statements:                     # Advanced Config style
      - set(attributes["env"], "prod")
    metric_statements:              # Basic Config style (CANNOT mix with above)
      - context: metric
        statements:
          - set(name, Concat([name, "_v2"], ""))

# AFTER OPTION A — Pure Basic Config (recommended):
processors:
  transform:
    metric_statements:
      - context: metric
        statements:
          - set(name, Concat([name, "_v2"], ""))
    log_statements:
      - context: log
        statements:
          - set(attributes["env"], "prod")

# AFTER OPTION B — Pure Advanced Config:
processors:
  transform:
    statements:
      - context: metric
        statements:
          - set(name, Concat([name, "_v2"], ""))
      - context: log
        statements:
          - set(attributes["env"], "prod")
```

---

### P1-26 · k8sattributes node_from_env_var Startup Error (0.125)

**Look for:** `node_from_env_var:` under `processors: k8s_attributes:` (or `k8sattributes:`).

**Impact:** Collector fails to start if the referenced environment variable is not set. Before
0.125, an unset env var silently caused the processor to monitor the **entire cluster** instead
of a single node — a dangerous silent misconfiguration that is now a hard startup error.

**Action:** Either ensure the environment variable is always set (e.g., via the Kubernetes
Downward API), or remove `node_from_env_var:` to intentionally monitor the whole cluster.

```yaml
# BEFORE — silently monitored whole cluster if MY_NODE_NAME was unset:
processors:
  k8s_attributes:
    node_from_env_var: MY_NODE_NAME   # startup error in 0.125+ if var is unset

# AFTER OPTION A — Inject node name via Kubernetes Downward API (recommended):
# In your Kubernetes pod spec:
#   env:
#     - name: MY_NODE_NAME
#       valueFrom:
#         fieldRef:
#           fieldPath: spec.nodeName
# Config stays the same — env var will always be set

# AFTER OPTION B — Remove to monitor whole cluster intentionally:
processors:
  k8s_attributes: {}   # no node_from_env_var = monitors full cluster
```

---

## P2 Degrading Changes — v0.120–v0.125

---

### P2-19 · prometheus Receiver — Prometheus 3.0 Metric Name Changes (0.120)

**Look for:** Any `prometheus:` receiver block, AND any dashboards, OTTL rules, or downstream
systems that reference internal collector metrics by name.

**Impact:** Prometheus 3.0 scraper (adopted in 0.120) no longer escapes dots in metric names.
Metrics that previously had underscores in place of dots now use dots. Dashboards and alert
rules using the old underscore-based names will break.

**Key renames:**
- `processor_filter_datapoints_filtered` → `processor_filter_datapoints.filtered`
- `processor_filter_logs_filtered` → `processor_filter_logs.filtered`
- `processor_filter_spans_filtered` → `processor_filter_spans.filtered`
- `deltatocumulative_streams_tracked` → `deltatocumulative.streams.tracked`
- `deltatocumulative_streams_tracked_linear` → `deltatocumulative.streams.tracked.linear`
- `deltatocumulative_streams_limit` → `deltatocumulative.streams.limit`
- `deltatocumulative_streams_evicted` → `deltatocumulative.streams.evicted`
- `deltatocumulative_streams_max_stale` → `deltatocumulative.streams.max_stale`
- `deltatocumulative_datapoints_processed` → `deltatocumulative.datapoints.processed`
- `deltatocumulative_datapoints_dropped` → `deltatocumulative.datapoints.dropped`
- `deltatocumulative_datapoints_linear` → `deltatocumulative.datapoints.linear`
- `deltatocumulative_gaps_length` → `deltatocumulative.gaps.length`
- `receiver_googlecloudpubsub_stream_restarts` → `receiver.googlecloudpubsub.stream_restarts`

**Resource attribute renames (scraping self-monitoring):**
- `service_name` → `service.name`
- `service_instance_id` → `service.instance.id`
- `service_version` → `service.version`

**Action:** Update all dashboard queries, alert rules, and OTTL `filter` or `transform`
statements that reference the old metric names. No YAML config change is required in the
collector configuration itself.

---

### P2-20 · tail_sampling Decision Timer Metric Unit Change (0.120)

**Look for:** Any dashboards or alert rules that query the `tail_sampling` processor's
decision timer metric (`processor_tail_sampling_sampling_decision_timer_firing_time`
or similar).

**Impact:** The unit of the decision timer metric changed from **microseconds** to
**milliseconds** in 0.120. Alert thresholds set in microseconds will be off by a factor
of 1000 after upgrade.

**Action:** Divide all alert thresholds referencing the tail_sampling decision timer metric
by 1000 after upgrading. No collector config change is required.

---

### P2-21 · activedirectoryds Receiver Attribute Typo Fixed (0.120)

**Look for:** Any OTTL rules, `filter` processor conditions, dashboards, or downstream
systems that reference the attribute `distingushed_names` (note the misspelling — missing
the 'i' in 'distinguished').

**Impact:** The typo in the attribute name was fixed: `distingushed_names` → `distinguished_names`.
Any OTTL statements or queries using the old misspelled name will silently stop matching.

**Action:** Search across all configs and dashboards for `distingushed_names` and replace
with `distinguished_names`.

```yaml
# BEFORE — typo attribute name (0.119 and earlier):
# activedirectoryds emitted: distingushed_names

# AFTER — corrected attribute name (0.120+):
# activedirectoryds emits: distinguished_names

# Example OTTL fix:
# OLD:  where attributes["distingushed_names"] == "CN=Admin"
# NEW:  where attributes["distinguished_names"] == "CN=Admin"
```

---

### P2-22 · confighttp HTTP Client Options Type Change (0.121)

**Look for:** Any component config block (receiver, exporter, extension) that uses HTTP
client settings with `max_idle_conns:`, `max_idle_conns_per_host:`, `max_conns_per_host:`,
or `idle_conn_timeout:` set to `null` or omitted with the intent of unlimited connections.

**Impact:** These four options changed from nullable to integer type in 0.121. Setting them
to `null` will now cause a YAML parse error. Setting to `0` preserves the "unlimited" behavior.

**Action:** Replace any `null` values with `0`, or remove the field entirely (default is `0`).

```yaml
# BEFORE — null caused parse issues in 0.121+:
exporters:
  otlphttp:
    http_client_config:
      max_idle_conns: null         # INVALID in 0.121+
      max_idle_conns_per_host: null
      idle_conn_timeout: null

# AFTER — use 0 for unlimited, or omit entirely:
exporters:
  otlphttp:
    http_client_config:
      max_idle_conns: 0         # 0 = unlimited (same as previous null behavior)
      # or simply omit — 0 is the default
```

---

### P2-23 · awss3 Exporter s3_partition → s3_partition_format (0.121)

**Look for:** `s3_partition:` key in any `awss3:` exporter block.

**Impact:** The `s3_partition` option was replaced by `s3_partition_format` in 0.121. The
old key will cause a startup error.

**Action:** Replace `s3_partition: <preset>` with the equivalent `s3_partition_format:`
strftime string.

```yaml
# BEFORE — removed in 0.121:
exporters:
  awss3:
    s3uploader:
      s3_partition: minute    # REMOVED

# AFTER — equivalent strftime format:
exporters:
  awss3:
    s3uploader:
      s3_partition_format: "year=%Y/month=%m/day=%d/hour=%H/minute=%M"
      # This is the default value — same as the old s3_partition: minute

# Other common mappings:
# s3_partition: hour  → s3_partition_format: "year=%Y/month=%m/day=%d/hour=%H"
# s3_partition: day   → s3_partition_format: "year=%Y/month=%m/day=%d"
```

---

### P2-24 · Batch Processor Telemetry Requires level: normal (0.122)

**Look for:** `service: telemetry: metrics: level: basic` in your config.

**Impact:** Batch processor metrics (e.g., `processor_batch_*`) are no longer emitted at
`level: basic` verbosity. If your telemetry level is `basic` and you monitor batch processor
behavior, those metrics will disappear after upgrade to 0.122+.

**Action:** Change telemetry level from `basic` to `normal` to restore batch processor metrics.

```yaml
# BEFORE — batch processor metrics visible at basic level (pre-0.122):
service:
  telemetry:
    metrics:
      level: basic

# AFTER — must use normal to keep batch processor metrics:
service:
  telemetry:
    metrics:
      level: normal   # was: basic
      # 'normal' is also the default if not specified
```

---

### P2-25 · sqlserver Receiver X.509 Certificate Requirement (0.122)

**Look for:** `sqlserver:` receiver blocks where TLS is configured and a custom X.509
certificate is used for authentication.

**Impact:** The SQL Server receiver now requires X.509 certificates to have a positive serial
number. Certificates with a zero or negative serial number will cause the receiver to fail to
start or fail to authenticate.

**Action:** Regenerate any affected certificates using standard CA tooling to ensure a positive
serial number. This is a certificate management action — no YAML config change is required.

---

### P2-26 · prometheusremotewrite Exporter export_created_metric Removed (0.123)

**Look for:** `export_created_metric:` in any `prometheusremotewrite:` exporter block.

**Impact:** The `export_created_metric` config option was removed in 0.123. Collector fails
to start if this field is present.

**Action:** Remove the `export_created_metric:` field entirely.

```yaml
# BEFORE — causes startup failure in 0.123+:
exporters:
  prometheusremotewrite:
    endpoint: https://prometheus.example.com/api/v1/write
    export_created_metric: true    # REMOVED

# AFTER:
exporters:
  prometheusremotewrite:
    endpoint: https://prometheus.example.com/api/v1/write
```

---

### P2-27 · splunkenterprise Receiver Default Metrics Now Opt-In (0.124)

**Look for:** Any `splunkenterprise:` receiver block where you relied on the default set of
scraped metrics without explicit `metrics:` config.

**Impact:** In 0.124, all `splunkenterprise` receiver metrics are **disabled by default**
except for `splunk.health`. If you were relying on the default metric set, those metrics will
silently stop being collected after upgrade.

**Action:** Explicitly enable each metric you need in the receiver config.

```yaml
# BEFORE — all metrics enabled by default (pre-0.124):
receivers:
  splunkenterprise:
    basicauth/bfauth:
      client_auth:
        username: admin
        password: "changeme"
    endpoint: https://splunk:8089

# AFTER — explicitly re-enable desired metrics:
receivers:
  splunkenterprise:
    basicauth/bfauth:
      client_auth:
        username: admin
        password: "changeme"
    endpoint: https://splunk:8089
    metrics:
      splunk.data.indexes.extended.bucket.count:
        enabled: true
      splunk.index.size.extended:
        enabled: true
      splunk.license.index.usage:
        enabled: true
      # Add each metric you need — only splunk.health is on by default
```

---

### P2-28 · sqlserver Receiver db.lock_timeout Unit Change (0.124)

**Look for:** Any dashboards, alert rules, or OTTL statements that reference the
`db.lock_timeout` attribute in SQL Server query sample collection logs.

**Impact:** The unit of the `db.lock_timeout` attribute changed from **milliseconds** to
**seconds** in 0.124. Alert thresholds and dashboard visualizations will be incorrect by
a factor of 1000 after upgrade. No collector config change is required.

**Action:** Divide all `db.lock_timeout` threshold values in dashboards and alert rules by
1000 after upgrading.

---

### P2-29 · Kafka auth::tls Deprecated — New Top-Level tls Config (0.124)

**Look for:** `auth: tls:` inside any `kafka:` receiver, exporter, or `kafka_metrics:`
receiver block.

**Impact:** The `auth::tls` nested path is deprecated in 0.124 in favor of a new top-level
`tls:` field. The old path still works in 0.124 but will be removed in a future release.

**Action:** Migrate to the top-level `tls:` block to avoid a future startup failure.

```yaml
# BEFORE — deprecated in 0.124 (will be removed in a future release):
receivers:
  kafka:
    auth:
      tls:
        ca_file: /etc/ssl/ca.pem
        cert_file: /etc/ssl/cert.pem
        key_file: /etc/ssl/key.pem

exporters:
  kafka:
    auth:
      tls:
        insecure_skip_verify: false

# AFTER — use top-level tls: block:
receivers:
  kafka:
    tls:
      ca_file: /etc/ssl/ca.pem
      cert_file: /etc/ssl/cert.pem
      key_file: /etc/ssl/key.pem

exporters:
  kafka:
    tls:
      insecure_skip_verify: false
```

---

### P2-30 · otelcol.component.kind Attribute Values Lowercase (0.125)

**Look for:** Any OTTL statements, `filter` processor conditions, dashboards, or downstream
alert rules that check the value of the `otelcol.component.kind` attribute in internal
collector telemetry.

**Impact:** Attribute values changed from title case to lowercase in 0.125. For example,
`Receiver` → `receiver`, `Exporter` → `exporter`, `Processor` → `processor`. Any exact-match
comparisons against the old capitalized values will silently stop matching.

**Action:** Update all OTTL conditions, filter rules, and dashboard queries to use lowercase
values.

```yaml
# Example OTTL fix:
# OLD:  where attributes["otelcol.component.kind"] == "Receiver"
# NEW:  where attributes["otelcol.component.kind"] == "receiver"

# OLD:  where attributes["otelcol.component.kind"] == "Exporter"
# NEW:  where attributes["otelcol.component.kind"] == "exporter"
```

---

### P2-31 · telemetry.newPipelineTelemetry Gate On for Logs and Traces (0.125)

**Look for:** Any pipeline that exports internal OTLP collector logs to a downstream system
and parses log attributes to identify the source component (e.g., looks for `otelcol.component.id`
as a log body attribute vs. a scope attribute).

**Impact:** With `telemetry.newPipelineTelemetry` now on by default for logs and traces,
internal collector logs exported over OTLP now use **instrumentation scope attributes** to
identify the source component, rather than regular log record attributes. Systems consuming
these logs that parse log attributes for component identity will stop seeing those attributes.

**Note:** This does not affect stderr/stdout log output. This gate remains **off by default
for metrics** (Prometheus exporter does not yet support scope attributes).

**Action:** Update OTLP log consumers to read component identity from the instrumentation
scope attributes instead of log record attributes. If you need to temporarily revert:

```yaml
# To temporarily disable for logs/traces (not recommended long-term):
# --feature-gates=-telemetry.newPipelineTelemetry
# Note: the off-state has a known regression where component attributes
# are missing from internal logs entirely. Migrate consumers instead.
```

---

### P2-32 · kubeletstats CPU Usage Metrics Migration (0.125)

**Look for:** Dashboards, alert rules, or OTTL statements that reference:
- `container.cpu.utilization`
- `k8s.pod.cpu.utilization`
- `k8s.node.cpu.utilization`

**Impact:** The `enableCPUUsageMetrics` feature gate was promoted to **beta** in 0.125
(becomes stable in 0.130). These deprecated metrics are being replaced by:
- `container.cpu.usage`
- `k8s.pod.cpu.usage`
- `k8s.node.cpu.usage`

By default in 0.125, both old and new metrics are emitted. In 0.130+ the old deprecated
metrics are off by default.

**Action:** Begin migrating dashboards and alerts to the `.usage` variant metrics. Explicitly
re-enable deprecated metrics in config if needed during transition:

```yaml
# To keep deprecated utilization metrics enabled during transition (0.125–0.129):
receivers:
  kubelet_stats:
    metrics:
      container.cpu.utilization:
        enabled: true   # deprecated — re-enable during migration
      k8s.pod.cpu.utilization:
        enabled: true   # deprecated
      k8s.node.cpu.utilization:
        enabled: true   # deprecated

# To disable the feature gate entirely and stay on old metrics:
# --feature-gates=-receiver.kubeletstats.enableCPUUsageMetrics
# WARNING: gate became STABLE in 0.130 — this flag FAILS collector startup from 0.130+.
```

---

### P2-33 · sqlserver Receiver Attributes Moved to Resource (0.125)

**Look for:** Any dashboards, OTTL rules, or downstream log parsing that access
`computer_name` or `instance_name` as **log attributes** in SQL Server top query collection
logs. Also look for code reading these values from the log body/attributes.

**Impact:** In 0.125, three attributes were promoted from log attributes to **resource
attributes** in both top query collection and query sample collection:
- `host.name` (new resource attribute)
- `sqlserver.computer.name` (replaces `computer_name` log attribute)
- `sqlserver.instance.name` (replaces `instance_name` log attribute)

The old `computer_name` and `instance_name` log attributes are now deprecated.

**Action:** Update log parsing, OTTL rules, and dashboards to read these values from the
resource attributes instead of log record attributes.

```yaml
# Example OTTL fix (reading from correct location after 0.125):
# OLD: attributes["computer_name"]         (log attribute, deprecated)
# NEW: resource.attributes["sqlserver.computer.name"]

# OLD: attributes["instance_name"]         (log attribute, deprecated)
# NEW: resource.attributes["sqlserver.instance.name"]

# OLD: attributes["host.name"]             (was log attribute)
# NEW: resource.attributes["host.name"]    (now resource attribute)
```

---

### P2-34 · kafka Exporter + kafkametricsreceiver — Client ID and Auth Changes (0.123)

**Look for:** Any `kafka:` exporter or `kafkametrics:` receiver block. Check if Kafka broker ACLs,
monitoring, or audit logs filter by client ID. Also look for `auth.plain_text:` or
`refresh_frequency:` keys.

**Impact (three sub-changes from 0.123):**

1. **kafka exporter** and **kafkametricsreceiver** default `client_id` changed from `"sarama"` to
   `"otel-collector"`. Kafka ACLs that grant access by client ID `"sarama"` will silently break
   (collector runs but sends/receives nothing).
2. `auth.plain_text:` is **deprecated** on all Kafka components (exporter, receiver, kafkametrics).
   Use `auth.sasl:` with `mechanism: PLAIN` instead.
3. `kafkametricsreceiver` field `refresh_frequency:` is **deprecated**. Use
   `metadata.refresh_interval:` instead.

> **See also:** P2-01 covers the same client_id change for the kafka RECEIVER (Sarama), which did
> not change until 0.130 because the config field was ineffective (always used "sarama").

**Action:**

```yaml
# 1. Pin client_id to preserve ACL compatibility, OR update Kafka ACLs:
exporters:
  kafka:
    client_id: sarama   # preserves ACL match until ACLs are updated

receivers:
  kafkametrics:
    client_id: sarama   # preserves ACL match until ACLs are updated

# 2. Migrate auth.plain_text → auth.sasl (PLAIN):
# BEFORE:
receivers:
  kafka:
    auth:
      plain_text:
        username: myuser
        password: mypass

# AFTER:
receivers:
  kafka:
    auth:
      sasl:
        mechanism: PLAIN
        username: myuser
        password: mypass

# 3. Rename kafkametricsreceiver refresh_frequency:
# BEFORE:
receivers:
  kafkametrics:
    refresh_frequency: 30s

# AFTER:
receivers:
  kafkametrics:
    metadata:
      refresh_interval: 30s
```

---

## P3 Advisory Changes — v0.120–v0.125

---

### P3-11 · hostmetrics normalizeProcessCPUUtilization Gate Removed (0.120)

**Look for:** `--feature-gates=receiver.hostmetrics.normalizeProcessCPUUtilization` or
`+receiver.hostmetrics.normalizeProcessCPUUtilization` in startup scripts, systemd unit
files, Helm values, or Windows registry.

**Impact:** The `receiver.hostmetrics.normalizeProcessCPUUtilization` feature gate was
stabilized and then removed in 0.120. The behavior (normalizing CPU utilization by number
of logical CPUs) is now always enabled. Specifying the gate causes an "unknown feature gate"
error on startup.

**Action:** Remove the feature gate flag from startup configuration.

---

### P3-12 · k8sattributes fieldExtractConfigRegex.disallow Gate to Stable (0.121)

**Look for:** `--feature-gates=processor.k8sattributes.fieldExtractConfigRegex.disallow`
in startup scripts, systemd unit files, Helm values, or Windows registry.

**Impact:** The gate was moved to stable in 0.121. Regex-based field extraction config
validation is now always enforced. The gate itself will be removed in a future release.

**Action:** Remove the feature gate flag from startup configuration. Verify your
`k8sattributes` extract config does not rely on patterns that were only allowed under the
old permissive validation.

---

### P3-13 · k8sattributes k8sattr.rfc3339 Gate Removed (0.123)

**Look for:** `--feature-gates=processor.k8sattributes.k8sattr.rfc3339` in startup scripts,
systemd unit files, Helm values, or Windows registry.

**Impact:** The `k8sattr.rfc3339` gate was stabilized and removed in 0.123. RFC 3339
timestamp formatting for k8s attributes is now always used. Specifying the gate causes
an "unknown feature gate" error on startup.

**Action:** Remove the feature gate flag from startup configuration.

---

### P3-14 · k8sobjects Receiver API Check Moved to Startup (0.125)

**Look for:** Any `k8s_objects:` (or `k8sobjects:`) receiver block.

**Impact:** Prior to 0.125, the receiver validated that the referenced K8s API objects
exist during config validation (at startup before the service begins). In 0.125, this
check is now performed during receiver startup, not config validation. This means config
validation (`otelcol validate`) will no longer catch missing K8s API objects — those errors
only appear at runtime.

**Action:** No config change required. Be aware that `otelcol validate` is no longer
sufficient to detect K8s API availability issues; test with a live cluster instead.

