# Collector-upgRAde-Process

> **Note:** This Skill covers v0.120 → v0.153. For v0.154 and later, use the FLUXUS CLI (`fluxus assess` / `fluxus apply`) which is updated through v0.157.

A Cursor Agent Skill that upgrades Splunk OpenTelemetry Collector configuration files from **v0.120 → v0.153**.

Given one or more collector YAML configs, the skill first performs a **Pre-Assessment** ? scanning
each file for applicable changes in both **active** and **commented-out** config items, identifying
every change by category (🔴 P1 Breaking / 🟡 P2 Degrading / 🔵 P3 Advisory), writing a numbered list to
`PreAssessment.md`, and presenting a summary. Commented-out items get their own section and are
**never touched by default** ? the user must explicitly request `"include comments"` to process them.

The user then selects which changes to apply (`all`, `p1`, `p2`, `p3`, specific
numbers, or CC-prefixed numbers for commented-out items). The skill applies the selected changes,
writes per-file README change logs, performs an accuracy check, produces dedicated **pre- and
post-upgrade Data Path Assessment** documents (traces, metrics, logs end-to-end), and a multi-file
operational assessment that points to those documents for topology detail.

---

## What It Does

| Step | Description |
|------|-------------|
| 0a?0e. Pre-Assessment | Intake; scan active + commented-out configs against full change catalogue; split into current (? target version) and future (> target version) change buckets; write `PreAssessment.md` |
| 0f. Pre Data Path Assessment | **After** PreAssessment ? traces/metrics/logs on **original** configs re-read from disk; **active project only, no chat/transcript history** ? `Data Path Assessment ? Pre.md` |
| 0g. Present Summary | Present current vs future change counts + pre-upgrade data path verdicts to user; await selection |
| 1. Apply Changes | Applies only current changes the user selected; future (#F/#FCC) changes are never applied |
| 2. Write READMEs | Per-file README with applied changes, rationale, and pending actions |
| 3. Accuracy Check | Verifies upgrade changes; prompts user to run `otelcol validate`; updates READMEs |
| 4a. Post Data Path Assessment | Re-runs path analysis on **modified** configs ? `Data Path Assessment ? Post.md` (includes Pre vs Post comparison) |
| 4b. Operational Assessment | Upgrade summary, accuracy, pointers to both data path docs ? no duplicated topology |

---

## Files in This Skill

| File | Purpose |
|------|---------|
| `SKILL.md` | Main workflow ? step-by-step process the agent follows |
| `UPGRADE-KNOWLEDGE.md` | Full change catalogue with categories, YAML snippets, and decision guidance |
| `DATA-PATH-KNOWLEDGE.md` | Per-signal path verification (12 steps), golden paths, anti-patterns PATH-01?28, PROP1-01?05 |
| `TEMPLATES.md` | PreAssessment, Data Path Assessment, per-file README, Operational Assessment templates |
| `README.md` | This file |

---

## How to Use

### Method 1 ? Specify files directly

In any Cursor project, invoke the skill and list the config files explicitly:

```
Use the Collector-upgRAde-Process skill.

Config files:
- agent_config.yaml
- gateway-config.yaml (gateway / aggregator)

Target version: v0.153   # optional ? omit to apply all changes up to v0.153
```

To upgrade only partway (e.g., to v0.145), add the target version and the skill will apply changes
up to that version, then scan and present changes beyond it as **future changes** for awareness:

```
Use the Collector-upgRAde-Process skill.

Config files:
- agent_config.yaml
- gateway-config.yaml

Target version: v0.145
```

This is useful when your configs live across different locations, or when you only want to assess
a subset of the files in a larger project.

### Method 2 ? Open your configs folder as a Cursor project (recommended for multi-file upgrades)

If all your collector configs are in one folder, open that folder directly as a Cursor project
(`File > Open Folder`). Then simply invoke the skill without listing files:

```
Use the Collector-upgRAde-Process skill.

Target version: v0.153   # optional ? omit to apply all changes up to v0.153
```

The agent will scan the workspace, discover all `.yaml` / `.yml` files automatically, infer each
file's role (agent, gateway, standalone) from its content, and run the full Pre-Assessment across
all of them at once. This is the most efficient path when upgrading a complete deployment ? all
files are assessed together and the **Data Path Assessment** documents cover the full picture in one pass.

---

For either method, the agent will generate the PreAssessment and ask which changes to apply before
touching anything.

**Applying changes ? selection options:**

| Selection | What happens |
|---|---|
| `all` | Apply all active current changes |
| `p1` / `p2` / `p3` | Apply all current changes in that category |
| `1, 3, 5` | Apply specific active current changes by number |
| `all include comments` | Apply all active current changes + fix commented-out items |
| `p1 include comments` | Apply P1 active current + P1 commented-out |
| `1, CC2, CC4` | Mix active and commented-out current changes by number |
| `none` | Assessment only ? no config changes |

> **Future changes (#F1, #F2? / #FCC1, #FCC2?) are informational only and cannot be selected.**
> They are shown so you can decide whether to upgrade further than your target version.

---

## Outputs

**Always created (before any config changes):**
- **`PreAssessment.md`** ? upgrade catalogue scan; pointer to data path doc (Section 8)
- **`Data Path Assessment ? Pre.md`** ? traces/metrics/logs end-to-end paths on **original** configs

**Created for each selected change:**
- **Modified YAML(s)** ? active changes applied in-place

**After all files are processed:**
- **`Readme <filename>.md`** ? per-file upgrade change log
- **`Data Path Assessment ? Post.md`** ? paths on **modified** configs + Pre vs Post comparison
- **`Readme operational assessment.md`** ? upgrade accuracy summary; points to both data path documents

---

## What the Skill Knows (v0.120 → v0.157)

### Impact Categories

| Category | Meaning |
|---|---|
| 🔴 **P1 Breaking** | Collector startup failure or silent data loss if not fixed |
| 🟡 **P2 Degrading** | Config change, planning, or operational action required |
| 🔵 **P3 Advisory** | No immediate failure; cleanup / informational only |

### P1 Breaking Changes (32 total — P1-01 through P1-32)

| Code | Change | Version |
|------|--------|---------|
| P1-21 | routing connector `match_once` removed | 0.120 |
| P1-22 | signalfx exporter `translation_rules` removed ? use `transform` processor | 0.121 |
| P1-23 | `min_size_items`/`max_size_items` removed from batch config ? use `min_size`/`max_size` | 0.123 |
| P1-24 | `service::telemetry::address` silently ignored ? use `readers:` format in `metrics:` | 0.123 |
| P1-25 | `transform` processor Basic Config and Advanced Config cannot be mixed | 0.124 |
| P1-26 | `k8sattributes` `node_from_env_var` is now a hard startup error if env var unset | 0.125 |
| P1-01 | `routingprocessor` removed ? use routing connector | 0.134 |
| P1-02 | `sapm` receiver removed ? use `otlp` receiver | 0.135 |
| P1-03 | `sapm` exporter removed ? use `otlphttp` | 0.147 |
| P1-04 | `signalfx` receiver removed ? use `otlp` receiver | 0.153 |
| P1-05 | `access_token_passthrough` field removed (receiver + exporter) | 0.137 |
| P1-06 | OTLP exporter `batcher:` block removed | 0.130 |
| P1-07 | `splunk_hec` exporter `batcher:` block removed | 0.151 |
| P1-08 | kafka exporter top-level `topic`/`encoding` removed | 0.148 |
| P1-09 | kafka receiver `topic`/`exclude_topic`/`default_fetch_size` removed | 0.141?0.147 |
| P1-10 | sqlserver event flag renames (`top_query_collection` ? `events."db.server.top_query"`) | 0.128 |
| P1-11 | postgresql query collection flags removed | 0.132 |
| P1-12 | kubeletstats no-op config sections cause startup failure | 0.136 |
| P1-13 | `filter`/`transform` `error_mode` default: `propagate` ? `ignore` | 0.153 |
| P1-14 | OTTL `SetMap` error handling changed | 0.150 |
| P1-15 | OTTL type-strict setters (histogram paths on wrong data point types) | 0.150+0.153 |
| P1-16 | prometheus `use_start_time_metric`/`start_time_metric_regex`/`report_extra_scrape_metrics` removed | 0.143/0.149 |
| P1-17 | FluentD permanently removed from all Splunk installers | 0.144?0.145 |
| P1-18 | `resourcedetection` processor `attributes:` field removed | 0.142 |
| P1-19 | `mysql`/`postgresql` query collection defaults changed to off | 0.148 |
| P1-20 | `sending_queue::blocking` field removed → use `block_on_overflow` | 0.129 |
| P1-27 | Smart Agent extension `bundleDir` and `collectd:` block removed | 0.154 |
| P1-28 | Smart Agent `jmx` monitor removed — migrate to `jmx` receiver | 0.154 |
| P1-29 | Smart Agent `hana` monitor removed — migrate to `saphana`/`sqlquery` receivers | 0.154 |
| P1-30 | Smart Agent `signalfx-forwarder` and `trace-forwarder` monitors removed — migrate to OTLP receiver | 0.155 |
| P1-31 | `jmx` receiver removed from Splunk distribution — migrate to standalone `jmx-scraper` | 0.157 |
| P1-32 | `filter`/`transform` `defaultErrorModeIgnore` gates promoted to stable — revert flag fails startup | 0.157 |

### P2 Degrading Changes (43 total — P2-01 through P2-43)

| Code | Change | Version |
|------|--------|---------|
| P2-19 | prometheus receiver: Prometheus 3.0 metric name dots (internal metrics renamed) | 0.120 |
| P2-20 | `tail_sampling` decision timer unit changed: microseconds ? milliseconds | 0.120 |
| P2-21 | `activedirectoryds` attribute typo fixed: `distingushed_names` ? `distinguished_names` | 0.120 |
| P2-22 | `confighttp` HTTP client null fields ? use 0 for unlimited | 0.121 |
| P2-23 | `awss3` exporter `s3_partition` removed ? use `s3_partition_format` strftime | 0.121 |
| P2-24 | batch processor telemetry requires `level: normal` (was emitted at `level: basic`) | 0.122 |
| P2-25 | `sqlserver` receiver X.509 certificates must have positive serial number | 0.122 |
| P2-26 | `prometheusremotewrite` exporter `export_created_metric` removed | 0.123 |
| P2-27 | `splunkenterprise` receiver: all metrics now opt-in except `splunk.health` | 0.124 |
| P2-28 | `sqlserver` `db.lock_timeout` attribute unit changed: ms ? seconds | 0.124 |
| P2-29 | Kafka `auth::tls` deprecated ? use top-level `tls:` block | 0.124 |
| P2-30 | `otelcol.component.kind` attribute values now lowercase | 0.125 |
| P2-31 | `telemetry.newPipelineTelemetry` gate on by default for logs/traces (scope attrs) | 0.125 |
| P2-32 | `kubeletstats` `enableCPUUsageMetrics` gate to beta (cpu.utilization ? cpu.usage) | 0.125 |
| P2-33 | `sqlserver` `host.name`/`computer.name`/`instance.name` moved to resource attributes | 0.125 |
| P2-34 | kafka exporter + kafkametrics: `client_id` default `sarama` ? `otel-collector`; `auth.plain_text` deprecated; `refresh_frequency` renamed | 0.123 |
| P2-01 | Kafka receiver (Sarama): `client_id` now honoured ? default changes to `otel-collector` | 0.130 |
| P2-02 | kafka exporter batching requires explicit `metadata_keys` | 0.148 |
| P2-03 | `kafka_metrics` receiver Sarama removed | 0.152 |
| P2-04 | `cumulativetodelta` `max_staleness` default changed (0 ? 1h) | 0.142 |
| P2-05 | `resourcedetection` cloud platform values changed (`azure_eks` ? `azure.eks`) | 0.147 |
| P2-06 | `tail_sampling` invert decisions permanently disabled | 0.144/0.152 |
| P2-07 | prometheus start time no longer adjusted | 0.140 |
| P2-08 | `docker_observer`/`docker_stats` Docker API version upgraded to 1.44 | 0.141/0.142 |
| P2-09 | mongodb schema change (`database` resource attr removed; `db.namespace` added) | 0.147 |
| P2-10 | `windows_event_log` `event_data` format changed (array ? flat map) | 0.148 |
| P2-11 | `http_check` timing metrics changed to nanoseconds | 0.153 |
| P2-12 | sqlserver event names and attribute renamed | 0.126 |
| P2-13 | `prometheusremotewrite` updated to Remote Write 2.0 (requires Prometheus 3.8+) | 0.142 |
| P2-14 | signalfx exporter default URL domain: `signalfx.com` ? `observability.splunkcloud.com` | 0.151 |
| P2-15 | IIS application pool metrics enabled by default | 0.131 |
| P2-16 | Windows MSI download URL changed | 0.151 |
| P2-17 | Internal metrics `service_name`/`service_instance_id`/`service_version` labels removed | 0.149 |
| P2-18 | prometheus receiver resource attributes renamed: `net.host.name` → `server.address` etc. | 0.126 |
| P2-36 | kafka receiver `group_rebalance_strategy` deprecated — use `group_rebalance_strategies:` | 0.154 |
| P2-37 | `memory_limiter` internal metrics renamed with `otelcol_processor_memory_limiter_` prefix | 0.155 |
| P2-38 | `oracledb` receiver `db.namespace` attribute semantic changed | 0.155 |
| P2-39 | `signalfx` exporter per-core CPU metrics removed from default translations | 0.155 |
| P2-40 | `oracledb` receiver requires new `V$SQL_PLAN_STATISTICS_ALL` privilege | 0.156 |
| P2-41 | `prometheus` receiver `IgnoreScopeInfoMetric` gate to beta — `otel_scope_info` metrics suppressed by default | 0.156 |
| P2-42 | `host_metrics` cpu scraper: per-core data points now opt-in; `system.cpu.logical.count` added by default | 0.157 |
| P2-43 | routing connector default `error_mode` changed to `ignore` | 0.157 |

### P3 Advisory Changes (18 total — P3-01 through P3-18)

| Code | Change | Version |
|------|--------|---------|
| P3-11 | `hostmetrics` `normalizeProcessCPUUtilization` feature gate removed | 0.120 |
| P3-12 | `k8sattributes` `fieldExtractConfigRegex.disallow` gate promoted to stable | 0.121 |
| P3-13 | `k8sattributes` `k8sattr.rfc3339` gate removed | 0.123 |
| P3-14 | `k8sobjects` API availability check moved from config validation to runtime | 0.125 |
| P3-01 | Component renames ? 25+ components to snake_case (aliases still work) | 0.145?0.153 |
| P3-02 | Kafka deprecated no-op fields (`resolve_canonical_bootstrap_servers_only`, `auth.sasl.version`) | 0.153 |
| P3-03 | `k8s_attributes` new semconv feature gates | 0.146 |
| P3-04 | `kubelet_stats` deprecated resource attributes disabled by default | 0.150 |
| P3-05 | jaeger `DisableRemoteSampling` gate removed | 0.153 |
| P3-06 | clickhouse `json` feature gate removed | 0.153 |
| P3-07 | Config debug endpoint (`localhost:55554`) removed | 0.142 |
| P3-08 | `splunk_otlp_histograms` converter removed | 0.148 |
| P3-09 | New OTTL functions available (Coalesce, Base64Encode, IsInCIDR, etc.) | 0.126+ |
| P3-10 | `truncate_all` OTTL function now UTF-8 safe by default | 0.148 |
| P3-15 | `resourcedetection` `k8snode` detector renamed to `k8s_api` | 0.154 |
| P3-16 | `signalfx` exporter trace functionality deprecated (removal December 2026) | 0.154 |
| P3-17 | Splunk `timestamp` processor deprecated — migrate to OTTL `transform` | 0.156 |
| P3-18 | routing connector `request` context and `request["key"]` syntax deprecated | 0.156 |

---

## Deploying Locally

Skills are stored in a folder named `Skill` inside your Cursor user directory. The **name of the
subfolder becomes the name of the skill** ? so the folder structure must be exact for Cursor to
recognize and load it.

### Personal install (available in all your Cursor projects)

```
~/.cursor/skills/Collector-upgRAde-Process/
  ??? SKILL.md
  ??? UPGRADE-KNOWLEDGE.md
  ??? DATA-PATH-KNOWLEDGE.md
  ??? TEMPLATES.md
  ??? README.md
```

On Windows this expands to:
```
C:\Users\<you>\.cursor\skills\Collector-upgRAde-Process\
```

On macOS/Linux:
```
/Users/<you>/.cursor/skills/Collector-upgRAde-Process/
```

Once the files are in place, the skill is immediately available. Invoke it in any chat by name:
```
Use the Collector-upgRAde-Process skill.
```
The skill name Cursor uses is exactly the subfolder name (`Collector-upgRAde-Process`).

### Project-scoped install (available only within one project)

Place the folder inside the project's `.cursor/skills/` directory instead:
```
<your-project>/.cursor/skills/Collector-upgRAde-Process/
  ??? SKILL.md
  ??? UPGRADE-KNOWLEDGE.md
  ??? DATA-PATH-KNOWLEDGE.md
  ??? TEMPLATES.md
  ??? README.md
```

Committing this to your repository lets teammates clone the repo and have the skill available
automatically in that project without any manual copy step.

---

## Extending the Skill

To adapt for a different upgrade range (e.g., v0.153 ? v0.170):

1. Update `UPGRADE-KNOWLEDGE.md` with the new change catalogue for that version window
2. Update the version references in `SKILL.md` frontmatter description and section headers
3. Update `TEMPLATES.md` post-upgrade checklist if new validation steps are needed

The pre-assessment workflow in `SKILL.md` is version-agnostic ? only the knowledge base needs updating
for a new upgrade range.
