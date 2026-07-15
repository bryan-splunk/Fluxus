# Output Templates — Splunk OTel Collector Upgrade

---

## PreAssessment Template

Create this as `PreAssessment.md` in the working directory during Step 0.
This is always created before any config changes are made.

```markdown
# Pre-Assessment — Splunk OTel Collector Upgrade v0.120 → v0.153

> Assessed: <YYYY-MM-DD>
> Files assessed:
> - `<config-filename>.yaml` (<role / description>)
> - ...
> Source version: v0.120.0  ·  Target version: v0.153.0

---

## Summary

### Current Changes — Eligible for Application (→ v<target, or "v0.153 (all)" if no target>)

| Category | Active Config | Commented-Out | Meaning |
|---|---|---|---|
| 🔴 P1 Breaking | N | N | Startup failure or silent data loss if not addressed |
| 🟡 P2 Degrading | N | N | Config or planning required before/after upgrade |
| 🔵 P3 Advisory | N | N | Cleanup / informational — no immediate failure |
| **Total** | **N** | **N** | |

### Future Changes — Scanned but NOT Applied (v<target+1> → v0.153)

> ℹ️ These changes are beyond your target version. They have been scanned against your config
> and are shown so you can decide whether to upgrade to a later version instead.
> **They will NOT be applied in this run regardless of your selection.**

| Category | Affects Your Config | Informational Only | Meaning if You Upgrade Further |
|---|---|---|---|
| 🔴 P1 Breaking | N | N | Would cause startup failure or data loss |
| 🟡 P2 Degrading | N | N | Would require config or planning |
| 🔵 P3 Advisory | N | N | Cleanup / informational |
| **Total** | **N** | **N** | |

> *(If no target version was specified, this section is omitted — all changes are current.)*

---

## Section 1 — Active Config Changes

How to apply these changes:
- `all` — apply all active changes
- `p1` / `p2` / `p3` — apply by category
- `1, 3, 5` — apply specific changes by number

### 🔴 P1 Breaking Changes (Active)

| # | Change | Affects | Description |
|---|---|---|---|
| 1 | P1-04 — signalfx Receiver Removed | `gateway.yaml` line 12 | signalfx: receiver permanently removed in 0.153; collector fails to start |
| 2 | P1-13 — filter/transform error_mode Default Changed | `gateway.yaml` line 45, `agent.yaml` line 31 | Default changed propagate→ignore; errors now silently pass |
| ... | ... | ... | ... |

### 🟡 P2 Degrading Changes (Active)

| # | Change | Affects | Description |
|---|---|---|---|
| N | P2-14 — signalfx Exporter URL Domain Change | `gateway.yaml` | Default endpoints moved to *.observability.splunkcloud.com; update firewalls |
| ... | ... | ... | ... |

### 🔵 P3 Advisory Changes (Active)

| # | Change | Affects | Description |
|---|---|---|---|
| N | P3-01 — Component Renames | `agent.yaml`, `gateway.yaml` | Aliases: windowseventlog, hostmetrics, fluentforward, resourcedetection (+ N others) |
| ... | ... | ... | ... |

---

## Section 2 — Commented-Out Config Changes (Current Upgrade)

> ### ⚠️ IMPORTANT — COMMENTED-OUT ITEMS
>
> The items below were found inside **commented-out lines or blocks** in your config files.
> This includes top-level commented blocks, and sub-settings that are commented out deep within
> active receivers, processors, or exporters.
>
> **These are NOT changed by default.** To include them in the upgrade:
> - Add `"include comments"` to your change request (e.g. `"all include comments"`)
> - Or reference specific CC numbers (e.g. `"1, CC2, CC4"`)
>
> When processed, commented-out lines remain commented — only the content is updated, and a
> warning comment is added to blocks that would fail on uncommenting without the fix.

### 🔴 P1 Breaking (Commented-Out) — Would Fail if Uncommented

| # | Change | File | Location | Description |
|---|---|---|---|---|
| CC1 | P1-09 — kafka receiver removed keys | `agent.yaml` | ~line 110 (commented block) | `topic:` and `default_fetch_size:` would cause startup failure if uncommented |
| CC2 | P1-06 — OTLP exporter batcher block | `gateway.yaml` | ~line 201 (commented inside active otlp exporter) | `batcher:` block would cause startup failure if uncommented |
| ... | ... | ... | ... | ... |

### 🟡 P2 Degrading (Commented-Out) — Would Need Planning if Uncommented

| # | Change | File | Location | Description |
|---|---|---|---|---|
| CC3 | P2-01 — Kafka client ID | `agent.yaml` | ~line 115 | `client_id: sarama` — old default value; ACLs may need updating |
| ... | ... | ... | ... | ... |

### 🔵 P3 Advisory (Commented-Out) — Cleanup if Uncommented

| # | Change | File | Location | Description |
|---|---|---|---|---|
| CC4 | P3-01 — Component rename: kafkametrics | `agent.yaml` | ~line 108 | `kafkametrics:` → `kafka_metrics:` (alias still works but will be removed) |
| ... | ... | ... | ... | ... |

---

## Section 3 — Future Changes (Beyond Target Version — NOT Applied)

> ℹ️ **These changes are beyond your target version of v\<target\>.**
> They have been fully scanned against your config and are presented here so you can decide
> whether to upgrade to a later version.
> **These changes will NOT be applied in this run regardless of your selection.**
> To apply them, re-run the skill with a higher target version (or no target version).
>
> *(This section is omitted when no target version is specified — all changes are current.)*

### 🔴 P1 Breaking Future — Active Config

| # | Change | Version | Affects | Description |
|---|---|---|---|---|
| F1 | P1-07 — splunk_hec exporter batcher removed | v0.151 | `gateway.yaml` line 201 | `batcher:` block must be removed |
| F2 | P1-14 — OTTL SetMap error handling | v0.150 | `gateway.yaml` line 88 | SetMap now returns error on type mismatch |
| ... | ... | ... | ... | ... |

### 🟡 P2 Degrading Future — Active Config

| # | Change | Version | Affects | Description |
|---|---|---|---|---|
| F3 | P2-14 — signalfx exporter URL domain change | v0.151 | `gateway.yaml` | Update firewall rules for new *.observability.splunkcloud.com endpoints |
| ... | ... | ... | ... | ... |

### 🔵 P3 Advisory Future — Active Config

| # | Change | Version | Affects | Description |
|---|---|---|---|---|
| F4 | P3-02 — Kafka deprecated no-op fields | v0.153 | NOT in config — informational | `rebalance_delay:`, `num_streams:` not found; no action needed |
| ... | ... | ... | ... | ... |

### Future Changes — Commented-Out Config

| # | Change | Version | File | Location | Description |
|---|---|---|---|---|---|
| FCC1 | P2-10 — windows_event_log event_data format | v0.148 | `agent.yaml` | ~line 55 | `event_data_as_map` field commented out — would need updating if uncommented |
| ... | ... | ... | ... | ... |

---

## Section 4 — Not Applicable — Checked and Excluded

| Change Code | Change | Reason Not Applicable |
|---|---|---|
| P1-01 | routingprocessor Removed | No routingprocessor in any config (active or commented) |
| P1-03 | sapm Exporter Removed | No sapm exporter in any config |
| ... | ... | ... |

---

## Section 5 — Pre-Existing Issues Found

> (Leave blank if none found)

| Issue | File | Location | Severity |
|---|---|---|---|
| Hardcoded credential in prometheus receiver | `agent.yaml` | line N | Security |
| Unused receiver definition (defined but not in any pipeline) | `agent.yaml` | receivers.unused_recv | Cleanup |
| ... | ... | ... | ... |

---

## Section 6 — SmartAgent Monitors Found

> (Leave blank if no SmartAgent monitors present — active or commented)

| Monitor | File | Active or Commented | Status | Replacement |
|---|---|---|---|---|
| `smartagent/ntp` | `agent.yaml` | Active | Removed in 0.149 | NTP receiver |
| `smartagent/postgresql` | `agent.yaml` | Commented (~line 55) | Removed in 0.149 | postgresql receiver |
| ... | ... | ... | ... | ... |

---

## Section 7 — Feature Gates Found

> (Leave blank if no feature gates present in startup args)

| Gate | File / Location | Status |
|---|---|---|
| `--feature-gates=receiver.kafkareceiver.UseFranzGo` | systemd unit / Windows registry | Removed in 0.144 — remove this flag |
| ... | ... | ... |

---

## Section 8 — Data Path Assessment

> End-to-end telemetry path analysis (traces, metrics, logs) is documented in a **separate**
> companion file — not in this document. That assessment uses **only** the config files listed
> in this PreAssessment header (re-read from disk); it does not use chat history or prior runs.

📄 **Review:** [`Data Path Assessment — Pre.md`](Data%20Path%20Assessment%20%E2%80%94%20Pre.md)

| Signal | Pre-upgrade verdict | Path issues found |
|--------|---------------------|-------------------|
| Traces | ✅ / ⚠️ / ❌ | N (PATH-*) |
| Metrics | ✅ / ⚠️ / ❌ | N |
| Logs | ✅ / ⚠️ / ❌ | N |

Path issues are independent of the upgrade change catalogue. Review the Pre data path document
before selecting which upgrade changes to apply.

---

## Recommended Order of Operations

Based on the changes above, recommended upgrade sequence:

1. Apply all P1 Breaking active changes first and validate with `otelcol validate`
2. Apply P2 Degrading active changes and test in staging
3. Apply P3 Advisory active renames and verify no remaining old names in active config
4. Update firewall allowlists and env vars (if P2-14 applies)
5. Update dashboards and alert thresholds (if P2-11, P2-12, P2-17 apply)
6. Remove feature gate flags from startup scripts
7. Review `Data Path Assessment — Pre.md` and remediate critical path issues if needed
8. *(Optional)* Apply commented-out fixes so base configs are upgrade-ready for deployment
```

---

## Data Path Assessment Template

Create **`Data Path Assessment — Pre.md`** during Step 0f (original configs) and
**`Data Path Assessment — Post.md`** during Step 4a (modified configs).

Use the same template structure for both. Set `phase` to **Pre** or **Post** in the title and header.

```markdown
# Data Path Assessment — <Pre | Post>

> Assessed: <YYYY-MM-DD>
> Phase: <Pre-upgrade (original configs) | Post-upgrade (after applied changes)>
> Scope: Active project only — configs re-read from disk this run; no prior chat or transcript history
> Files assessed (must match Step 0a intake exactly):
> - `<config-filename>.yaml` (<role — e.g. agent, gateway, standalone>)
> - ...

---

## Scope and Method

| Rule | This assessment |
|------|-----------------|
| Config source | YAML files listed above, read fresh from workspace |
| Chat / transcript history | Not used |
| Prior Data Path Assessment files | Not used (Pre vs Post compares only Pre.md from **this** run) |
| Files outside workspace scope | Not analysed |

---

## Executive Summary

| Signal | Verdict | Issues | Notes |
|--------|---------|--------|-------|
| Traces | ✅ HEALTHY / ⚠️ ISSUES / ❌ BROKEN | N PATH-* | <one-line summary> |
| Metrics | ✅ / ⚠️ / ❌ | N | |
| Logs | ✅ / ⚠️ / ❌ | N | |

**Overall:** ✅ HEALTHY / ⚠️ ISSUES / ❌ BROKEN

---

## Architecture Diagram

<Text-based or mermaid diagram showing how all assessed files connect for each signal type.>

---

## 1. Traces — End-to-End Path

### 1.1 Flow Table

| Step | Component | File | Detail | Status |
|------|-----------|------|--------|--------|
| Ingest | otlp receiver | `<agent>.yaml` | listen :4317 gRPC | ✅ |
| Pipeline | traces | `<agent>.yaml` | memory_limiter → batch → resource_detection | ✅ |
| Export | otlp/gateway | `<agent>.yaml` | → `<gateway-host>`:4317 gRPC | ✅ |
| Receive | otlp receiver | `<gateway>.yaml` | listen :4317 gRPC | ✅ |
| Pipeline | traces | `<gateway>.yaml` | memory_limiter → batch | ✅ |
| Cloud export | otlphttp | `<gateway>.yaml` | → ${SPLUNK_INGEST_URL}/v2/trace/otlp | ✅ |

### 1.2 Per-Agent Trace Paths

> Repeat when multiple agent configs exist.

| Agent file | Export target | Gateway match | Pipeline match | Verdict |
|------------|---------------|---------------|----------------|---------|
| `<agent-primary>.yaml` | :4317 | ✅ otlp :4317 | ✅ traces pipeline | ✅ |
| `<agent-alt>.yaml` | :4319 | ✅ otlp/gateway :4319 | ❌ metrics pipeline only | ❌ PATH-04 |

### 1.3 Trace Path Issues

| ID | Issue | Files | Severity | Likely symptom | Recommended fix |
|----|-------|-------|----------|----------------|-----------------|
| PATH-03 | Dual export otlp/gateway + signalfx on traces | `<agent>.yaml` ~line N | High | Duplicate spans | Remove signalfx from traces exporters |
| PATH-12 | Gateway cloud export missing X-SF-TOKEN | `<gateway>.yaml` ~line N | P1 Breaking | 401 at ingest | Add headers_setter / auth on otlphttp or signalfx |
| PATH-14 | gRPC/HTTP mismatch agent → gateway | `<agent>.yaml` ~line N | P1 Breaking | No data | Align protocol and port (4317 gRPC vs 4318 HTTP) |

---

## 2. Metrics — End-to-End Path

(Same structure as Section 1.)

---

## 3. Logs — End-to-End Path

(Same structure as Section 1.)

---

## 4. Cross-Agent Consistency

| Check | `<agent-primary>` | `<agent-alt>` | Consistent |
|-------|-------------------|---------------|------------|
| Traces export port | :4317 | :4319 | ❌ |
| Metrics export port | :4319 | :4319 | ✅ |
| Logs export port | :4317 | :4319 | ❌ |

---

## 5. Dead Ends and Unused Components

| File | Component | Type | Issue | Severity |
|------|-----------|------|-------|----------|
| `<gateway>.yaml` | `<receiver-name>` | Receiver | Defined, not in any pipeline | 🔵 PATH-07 |

---

## 6. Processor Order Analysis

| File | Pipeline | Current order | Issue | Recommended |
|------|----------|---------------|-------|-------------|
| `<agent>.yaml` | metrics | memory_limiter → batch → filter | PROP1-01 batch before filter | memory_limiter → filter → batch |

---

## 7. TLS / Protocol Coherence

| Agent exporter | TLS | Gateway receiver | TLS | Match |
|----------------|-----|------------------|-----|-------|
| otlp/gateway | insecure: true | otlp | none | ✅ |

---

## 7b. Auth / Token Propagation

| Hop | Token mechanism | Present | Status |
|-----|-----------------|---------|--------|
| Agent otlp/gateway export | headers_setter + X-SF-TOKEN | ✅ | |
| Gateway otlphttp/signalfx export | headers / auth.authenticator | ✅ / ❌ | PATH-12 if missing |
| batch.metadata_keys | X-SF-Token | ✅ | PATH-13 if exporter auth missing |

---

## 7c. Host Metadata Path

| Component | Pipeline | Exporter | sync_host_metadata | Status |
|-----------|----------|----------|-------------------|--------|
| metrics/internal | `<agent>.yaml` | signalfx | true | ✅ / PATH-16 if missing |

---

## 8. Pre vs Post Comparison

> **Include this section only in `Data Path Assessment — Post.md`.**
> Omit from the Pre document.

| Signal | Pre verdict | Post verdict | Delta |
|--------|-------------|--------------|-------|
| Traces | ❌ BROKEN | ✅ HEALTHY | Improved — PATH-NN resolved |
| Metrics | ✅ HEALTHY | ✅ HEALTHY | Unchanged |
| Logs | ⚠️ ISSUES | ⚠️ ISSUES | Unchanged — PATH-07 still open |

### Issues resolved by upgrade

| ID | Issue | Resolved by change |
|----|-------|-------------------|
| PATH-NN | Traces to wrong gateway port/pipeline | Config fix / not in upgrade catalogue |

### Issues still open

| ID | Issue | Action needed |
|----|-------|---------------|
| PATH-NN | Dual trace export | Remove redundant exporter from traces pipeline |

### Newly introduced issues

| ID | Issue | Introduced by |
|----|-------|---------------|
| — | None | — |

---

## 9. Prioritized Path Actions

| Priority | ID | Signal | Action | Status |
|----------|-----|--------|--------|--------|
| 🔴 | PATH-NN | Traces | Align agent traces exporter with gateway traces receiver/pipeline | ⬜ Open |
```

---

## Per-File README Template

Use this structure for each config file's README. File it as `Readme <config-filename>.md` alongside
the config. Populate each section only with items **relevant to this specific file**.

```markdown
## Required Changes for <config-filename>.yaml

---

### Applied Changes Log

> Last updated: <YYYY-MM-DD>

| # | Change | Category | Status |
|---|---|---|---|
| 1 | P1-04 — signalfx receiver removed | 🔴 P1 Breaking | ✅ Applied |
| 2 | P1-13 — filter/transform error_mode | 🔴 P1 Breaking | ✅ Applied |
| 3 | P2-14 — signalfx URL domain change | 🟡 P2 Degrading | ⚠️ Operational — update firewalls before upgrade |
| 4 | P3-01 — Component renames (N items) | 🔵 P3 Advisory | ✅ Applied |
| 5 | <Not selected change> | 🟡 P2 Degrading | ➖ Not selected — user chose P1 only |

---

### 1. <P1 Breaking change title> — [APPLIED] ✅

**Version:** 0.XXX  
**Category:** 🔴 P1 Breaking

**What changed:** <Brief description of what broke and why.>

**What was done in this file:**
- Removed `<old component>` from line N
- Replaced with `<new component>` at line N
- Updated pipeline reference at line N

**Before:**
```yaml
<old yaml snippet>
```

**After:**
```yaml
<new yaml snippet>
```

---

### N. <P2 Degrading change title> — [PENDING OPERATIONAL ACTION] ⚠️

**Version:** 0.XXX  
**Category:** 🟡 P2 Degrading

**What changed:** <Description.>

**Config change status:** ✅ Applied / N/A

**Pending operational action required:**
- [ ] Update firewall allowlist to allow `*.observability.splunkcloud.com`
- [ ] Update `SPLUNK_INGEST_URL` environment variable on all affected hosts
- [ ] (etc.)

---

### Things That Were Checked But Not Applicable

| Change Code | Change | Status in this file |
|---|---|---|
| P1-01 | routingprocessor Removed | Not applicable — no routing processor in this config |
| P1-09 | kafka receiver keys removed | Not applicable — no Kafka components |
| P1-06 | OTLP batcher removed | Clean — no batcher block present |
| ... | ... | ... |

---

### Summary — Changes Applied vs. Still Pending

| Category | Status |
|---|---|
| P1 Breaking — N changes | ✅ All applied |
| P2 Degrading — N changes | ✅ Applied / ⚠️ N pending operational actions |
| P3 Advisory — N renames | ✅ Applied |
| Pre-existing issues | ⚠️ N issues flagged — see Data Path Assessment — Post.md |

---

### Commented-Out Configuration Sections

For each commented-out receiver or processor block, state whether it will work correctly if
uncommented, and what changes (if any) are needed.

#### <receiver-name> — <Clean / One Required Change / N Issues>

<Brief explanation.>
```

---

## Operational Assessment Template

Create this as `Readme operational assessment.md` in the project directory after all config files
are processed.

```markdown
# Operational Assessment — Splunk OTel Collector Upgrade v0.120 → v0.153

> Assessed: <YYYY-MM-DD>
> Files assessed:
> - `<config-filename>.yaml` (<role — description>)
> - ...
> Changes applied: <all / p1 / p2 / p3 / numbers N, N, N>

---

## 1. Data Path Assessment

> Architecture diagrams, end-to-end flow tables, connectivity checks, dead-end detection,
> and processor-order analysis are **not duplicated here**. Review the dedicated documents:

| Document | Contents |
|----------|----------|
| [`Data Path Assessment — Pre.md`](Data%20Path%20Assessment%20%E2%80%94%20Pre.md) | Paths **before** upgrade changes (original configs) |
| [`Data Path Assessment — Post.md`](Data%20Path%20Assessment%20%E2%80%94%20Post.md) | Paths **after** upgrade changes + Pre vs Post comparison |

### Data path verdict summary

| Signal | Pre | Post | Delta |
|--------|-----|------|-------|
| Traces | ✅ / ⚠️ / ❌ | ✅ / ⚠️ / ❌ | Improved / Unchanged / Regressed |
| Metrics | ✅ / ⚠️ / ❌ | ✅ / ⚠️ / ❌ | |
| Logs | ✅ / ⚠️ / ❌ | ✅ / ⚠️ / ❌ | |

Open path issues after upgrade: **N** (list PATH-* IDs or "none")

---

## 2. Pre-Assessment Summary

| Metric | Value |
|---|---|
| 🔴 P1 Breaking changes found | N |
| 🟡 P2 Degrading changes found | N |
| 🔵 P3 Advisory changes found | N |
| Changes applied | N (per user selection) |
| Changes pending | N |
| Pre-existing issues flagged | N |

---

## 3. Per-File Accuracy Assessment

| File | Role | P1 Breaking ✅ | P2 Degrading ✅ | P3 Advisory ✅ | Pending ⚠️ | Verdict |
|---|---|---|---|---|---|---|
| `agent.yaml` | Agent | N/N | N/N | N/N | N | ✅ ACCURATE / ⚠️ ISSUES |
| `gateway.yaml` | Gateway | N/N | N/N | N/N | N | ✅ ACCURATE / ⚠️ ISSUES |

### 3.N <config-filename>.yaml

| Check | Result |
|---|---|
| P1-04 signalfx receiver removed | ✅ Removed |
| P1-13 filter error_mode | ✅ Set explicitly |
| P3-01 Component renames | ✅ N renames applied |
| P2-14 signalfx URL domain | ⚠️ Operational action pending |
| ... | ... |

**Verdict: ACCURATE** / **ISSUES FOUND** — <summary>

---

## 4. Full Change Log

| # | Change | File | Category | Status | Notes |
|---|---|---|---|---|---|
| 1 | P1-04 — signalfx receiver | `gateway.yaml` | 🔴 P1 Breaking | ✅ Applied | Comment block added |
| 2 | P1-13 — error_mode | `gateway.yaml`, `agent.yaml` | 🔴 P1 Breaking | ✅ Applied | error_mode: propagate set |
| 3 | P2-14 — signalfx URL | `gateway.yaml` | 🟡 P2 Degrading | ⚠️ Pending | Firewall + env var update needed |
| ... | ... | ... | ... | ... | ... |

---

## 5. P1 Breaking Issues

### 🔴 P1 BREAKING ISSUE #N — <Title>

**Status:** ✅ Resolved in config / ⬜ Pending / ⚠️ Requires action outside config

**Description:** <What is broken and why.>

**Impact:** <What fails if this is not fixed.>

**Recommended Fix:**
```yaml
<corrected yaml snippet>
```

---

## 6. Issues Requiring a Decision

### 🟡 ISSUE #N — <Title>

**Description:** <What needs a decision and what the options are.>

**Recommendation:** <Which option and why.>

---

## 7. P3 Advisory Observations

### 🔵 OBS #N — <Title>

**Affected files:** `<filename>`

<Description, impact, suggested action.>

---

## 8. Dashboard and Alert Impact

> Populate this section only when P2-11, P2-12, or P2-17 are applicable to this config.
> If none apply, write "None — no metric/attribute changes in selected upgrade scope."

| Change | Signal type | What changed | Dashboards / alerts at risk |
|--------|-------------|-------------|----------------------------|
| P2-11 — http_check timing | Metrics | `http.client.duration` now in **nanoseconds** (was milliseconds) | Any panel/alert with thresholds in ms (e.g. "response > 500") will never fire — values are now ~500,000,000 |
| P2-12 — sqlserver event renames | Metrics/Logs | Event names and attributes renamed | Alerts filtering on old event names break silently |
| P2-17 — internal metrics labels | Metrics | `service_name`, `service_instance_id`, `service_version` labels removed from collector internal metrics | Dashboards built on these labels go blank; SLOs that filter on them stop evaluating |

**Recommended action for each applicable change:**
- **P2-11:** Divide threshold values in all `http_check` dashboards/alerts by 1,000,000 (ms → ns).
  Or add a `transform` processor that converts: `multiply_sum_data_points(metric.name == "http.client.duration", 0.000001)`.
- **P2-12:** Search Splunk for alerts and dashboards using the old sqlserver event names and update them to the new names from UPGRADE-KNOWLEDGE.md.
- **P2-17:** Update internal collector dashboards to use the replacement attributes (`service.name`, `service.instance.id`, `service.version` from resource attributes, not metric labels).

---

## 9. Infrastructure and Environment Actions Required

> List all changes that require work outside the config YAML files.
> Populate only rows that apply; omit or write "N/A" for the rest.

| Action | Required by | Target | Priority |
|--------|------------|--------|----------|
| Update firewall rules to allow `*.observability.splunkcloud.com` | P2-14 | Network/ops team | 🟡 Before upgrade |
| Update `SPLUNK_INGEST_URL` env var on all affected hosts | P2-14 | Deployment automation | 🟡 Before upgrade |
| Update `SPLUNK_API_URL` env var if used | P2-14 | Deployment automation | 🟡 Before upgrade |
| Update Windows MSI download URL in deployment scripts | P2-16 | Automation/SCCM/Intune | 🟡 Before next install |
| Upgrade Prometheus to ≥ 3.8.0 | P2-13 | Platform team | 🟡 Before upgrade |
| Upgrade Docker Engine to API ≥ 1.44 | P2-08 | Platform team | 🟡 Before upgrade |
| Update Kafka ACLs for client ID `otel-collector` | P2-01 | Kafka admin | 🔴 Before upgrade (silent failure risk) |
| Remove feature gate flags from startup scripts (see Section 10) | Feature gates | Ops/deployment | 🟡 Before upgrade |

---

## 10. Feature Gate Locations

> List removed feature gates found (from PreAssessment Section 7), plus where to remove them.
> If no feature gates were found in workspace YAML, note the non-YAML locations to check manually.

**Gates found in workspace:** `<gate-name>` in `<file>` at `<location>` — Status: ✅ Removed / ⬜ Pending

**Non-YAML locations to check manually if gates were not found in workspace files:**

| Deployment type | Where to look | Example |
|----------------|---------------|---------|
| Linux systemd | `/etc/systemd/system/splunk-otel-collector.service` — `ExecStart=` line | `--feature-gates=-exporter.kafkaexporter.UseFranzGo` |
| Linux init.d / env file | `/etc/default/splunk-otel-collector` or `/etc/sysconfig/splunk-otel-collector` | `OTELCOL_OPTIONS=--feature-gates=...` |
| Windows service | Registry: `HKLM:\SYSTEM\CurrentControlSet\Services\splunk-otel-collector` → `ImagePath` | Run: `Get-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Services\splunk-otel-collector" \| Select-Object ImagePath` |
| Kubernetes / Helm | `values.yaml` → `featureGates:` or `agent.featureGates:` / `gateway.featureGates:` | Check Helm release: `helm get values <release-name>` |
| Docker run command | Inspect running container args | `docker inspect <container> --format='{{.Args}}'` |

**Removed gates to search for across all locations:**
See `UPGRADE-KNOWLEDGE.md § Feature Gate Removals` for the full list.
Key gates to check: `exporter.kafkaexporter.UseFranzGo`, `receiver.kafkareceiver.UseFranzGo`,
`telemetry.disableHighCardinalityMetrics`, `processor.tailsamplingprocessor.disableinvertdecisions`,
`receiver.jaeger.DisableRemoteSampling`.

---

## 11. Rollback Instructions

> Retain this section. If the upgrade causes unexpected issues, use these steps to revert.

**Pre-condition:** You backed up (or git-committed) your config files before the upgrade.

### To revert this upgrade:

1. **Stop the collector service:**
   ```bash
   # Linux:
   systemctl stop splunk-otel-collector

   # Windows:
   Stop-Service -Name splunk-otel-collector
   ```

2. **Restore original config files from backup:**
   ```bash
   # Linux (example — adjust path to your backup location):
   cp /backup/agent_config.yaml.bak /etc/otelcol/agent_config.yaml
   cp /backup/gateway_config.yaml.bak /etc/otelcol/gateway_config.yaml

   # Or from git:
   git checkout HEAD~1 -- agent_config.yaml gateway_config.yaml
   ```

3. **Downgrade the collector binary** (if it was already upgraded):
   - Linux: reinstall the previous package version via package manager
   - Windows: reinstall the previous MSI from the Splunk download archive
   - Docker: update the image tag in your compose/K8s manifest to the previous version

4. **Restart the service:**
   ```bash
   # Linux:
   systemctl start splunk-otel-collector && systemctl status splunk-otel-collector

   # Windows:
   Start-Service -Name splunk-otel-collector
   ```

5. **Verify telemetry is flowing** by checking the pre-upgrade data path (see `Data Path Assessment — Pre.md`).

> **Note:** If the original configs no longer start on the older binary (e.g., if the binary was
> downgraded beyond a previously applied change), restore one version at a time and validate between steps.

---

## 12. Pre/Post-Upgrade Checklist

### Before Upgrading

- [ ] **Back up or git-commit your config files NOW** — rollback instructions in Section 11
- [ ] Run `otelcol validate --config=<file>` on all modified configs (prompted after Step 3)
- [ ] Review PreAssessment.md and confirm all Infrastructure/Environment actions (Section 9) are scheduled
- [ ] Review **Data Path Assessment — Pre.md** — understand current telemetry paths and open issues
- [ ] Complete all 🔴 P1 Breaking infrastructure actions before upgrading (Kafka ACLs if P2-01, firewall if P2-14)
- [ ] Update firewall allowlists (if P2-14 applies) — see Section 9
- [ ] Update env vars `SPLUNK_INGEST_URL` / `SPLUNK_API_URL` (if P2-14 applies) — see Section 9
- [ ] Update download URLs in automation scripts (if P2-16 applies) — see Section 9
- [ ] Update Prometheus to 3.8.0+ before upgrade (if P2-13 applies) — see Section 9
- [ ] Update Docker Engine to API 1.44+ before upgrade (if P2-08 applies) — see Section 9
- [ ] Verify Kafka ACLs allow client ID `otel-collector` (if P2-01 applies) — see Section 9
- [ ] Remove removed feature gate flags from all startup locations (see Section 10) — Linux systemd, Windows registry, Helm values
- [ ] Test in staging environment for at least 1 pipeline cycle
- [ ] Fix critical PATH-* issues from Data Path Assessment — Pre.md before upgrading (if any)

### After Upgrading

- [ ] Verify service starts without errors on all collectors (agent and gateway)
- [ ] Confirm all expected pipelines are running
- [ ] Check collector logs for deprecated component warnings (expected for renamed components)
- [ ] Review **Data Path Assessment — Post.md** — compare Pre vs Post verdicts
- [ ] Validate traces, metrics, and logs flowing end-to-end per Post data path document
- [ ] Verify gateway is receiving from all expected agents (check gateway receiver stats)
- [ ] Confirm gateway is exporting successfully (check exporter success/failure metrics)
- [ ] Test any Kafka integrations end-to-end (if kafka changes applied) — watch for silent ACL failure
- [ ] Check OTTL pipeline logs for newly surfaced type-mismatch errors (if P1-14 / P1-15 applied)
- [ ] Update dashboards/alerts affected by unit or label changes (see Section 8: P2-11, P2-12, P2-17)
- [ ] Verify feature gate flags removed from all non-YAML locations (see Section 10)

---

## 13. Prioritized Action List

| Priority | # | Change | File(s) | Action | Status |
|---|---|---|---|---|---|
| 🔴 P1 Breaking | 1 | P1-04 signalfx receiver | `gateway.yaml` | Remove receiver + update pipelines | ✅ Applied |
| 🔴 P1 Breaking | 2 | P1-13 error_mode | all | Add explicit error_mode to filter/transform | ✅ Applied |
| 🟡 P2 Degrading | 3 | P2-14 signalfx URL | `gateway.yaml` | Update firewalls + SPLUNK_INGEST_URL | ⬜ Pending |
| 🔵 P3 Advisory | 4 | P3-01 renames | all | Rename N components to snake_case | ✅ Applied |
| 🔵 Pre-existing | — | Hardcoded credential | `agent.yaml` | Move password to env var | ⬜ Pending |

---

## 14. Validation Commands

```bash
# Validate all configs before deploying:
otelcol validate --config=/etc/otelcol/config.yaml

# Check for deprecated name warnings:
otelcol --config=/etc/otelcol/config.yaml 2>&1 | grep -i "deprecated"

# Linux: check service after upgrade:
systemctl status splunk-otel-collector
journalctl -u splunk-otel-collector -n 50 --no-pager

# View effective config via zpages:
curl http://localhost:55679/debug/expvarz | jq -r '.["splunk.config.effective"]'
```

```powershell
# Windows: validate config:
& "C:\Program Files\Splunk\OpenTelemetry Collector\otelcol.exe" validate --config="<path>"

# Check service status:
Get-Service -Name splunk-otel-collector | Select-Object Status, StartType

# Watch startup logs for errors:
Get-Content "C:\ProgramData\Splunk\OpenTelemetry Collector\*.log" -Wait | Select-String "deprecated|error|warn"
```
```

---

## Accuracy Check — Internal Checklist

This checklist is for the agent's own verification pass during Step 3. Do NOT include it verbatim
in any output README — it is the internal verification guide only.

```
P1 BREAKING REMOVALS (for each that was applied to active config):
- [ ] signalfx: receiver definition is GONE (comment block present if removed)
- [ ] signalfx removed from ALL pipeline receivers: lists
- [ ] sapm: exporter / receiver is GONE
- [ ] routingprocessor is GONE; routing connector added AND pipelines rewired (input pipeline exports [routing], each output pipeline receives [routing])
- [ ] No batcher: blocks on otlp or splunk_hec exporters
- [ ] No attributes: field under resource_detection / resourcedetection
- [ ] No top-level topic: / encoding: under kafka exporter
- [ ] No topic: / exclude_topic: / default_fetch_size: under kafka receiver
- [ ] No top_query_collection: / query_sample_collection: under sqlserver
- [ ] No query_sample_collection: / top_query_collection: under postgresql
- [ ] No use_start_time_metric: / start_time_metric_regex: / report_extra_scrape_metrics: under prometheus
- [ ] No kubeletstats no-op enabled: false sections
- [ ] No access_token_passthrough: in signalfx receiver, signalfx exporter, or splunk_hec exporter (P1-05)

CROSS-FILE DEPENDENCY CHECK (for receiver removals):
- [ ] After removing signalfx: receiver → scanned all other in-scope agent configs for signalfx: exporters to port 9943; updated to OTLP
- [ ] After removing sapm: receiver → scanned all other in-scope agent configs for sapm: exporters to port 7276; updated
- [ ] After P1-01 routing connector → verified service.pipelines has input pipeline with exporters: [routing] and output pipelines with receivers: [routing]

BEHAVIORAL CHANGES (for each that was applied to active config):
- [ ] Every filter: processor has explicit error_mode:
- [ ] Every transform: processor has explicit error_mode:
- [ ] OTTL set() on map/slice fields reviewed and noted

RENAMES (for each that was applied to active config):
- [ ] All old names absent from active definitions
- [ ] All old names absent from ALL active service.pipelines lists
- [ ] Scan: no remaining active "hostmetrics", "windowseventlog", "fluentforward", "resourcedetection",
        "k8sattributes", "kafkametrics", "filelog", "httpcheck", "spanmetrics" etc.

COMMENTED-OUT CHANGES (only if "include comments" was selected):
- [ ] Each updated commented block is still fully commented (no accidental uncommenting)
- [ ] Content inside fixed comment blocks is syntactically correct YAML
- [ ] ⚠️ UPGRADE NOTE comment added above any commented block with a P1 Breaking issue
- [ ] Inline [upgraded from: ...] comment added to each changed commented line
- [ ] Renames in commented blocks are updated (if renames were included in the selection)
- [ ] Old component names NOT updated in comments that were NOT selected (only touch what was selected)

FUTURE CHANGES (if a target version was specified):
- [ ] No #F or #FCC prefixed change was applied to any config file
- [ ] Future changes section in PreAssessment.md is complete with scan results for each #F entry
- [ ] Future changes clearly labelled as "NOT applied" in the PreAssessment document
- [ ] FCC entries (future commented-out) are also listed and not applied

FEATURE GATES:
- [ ] No removed feature gate flags remain in any startup script / manifest / registry

DATA PATH ASSESSMENT (Step 0f and Step 4a):
- [ ] Scope = Step 0a file list only; configs re-read from disk (not from chat memory)
- [ ] No prior-run Pre/Post docs, transcripts, or conversation history used as evidence
- [ ] Data Path Assessment — Pre.md written before user selection (original configs)
- [ ] Data Path Assessment — Post.md written after accuracy check (modified configs)
- [ ] Post document includes Pre vs Post comparison section (Pre.md from this run only)
- [ ] Traces, metrics, logs each have end-to-end flow tables in both documents
- [ ] Flow table rows cite filename + config evidence from disk read
- [ ] Steps 7–12 run: auth (PATH-12/13), host metadata (PATH-16/17), env legs, logs leg (PATH-20–22), protocol (PATH-14/15), multi-gateway (PATH-25/26)
- [ ] PATH-01 through PATH-28 flagged where applicable with severity and recommended fix
- [ ] PROP1-01 through PROP1-06 flagged where applicable
- [ ] Auth/token propagation table (§7b) and host metadata table (§7c) populated when relevant
- [ ] PreAssessment.md Section 8 points to Pre data path doc (no duplicated topology)
- [ ] Operational Assessment Section 1 points to both data path docs (no duplicated topology)
- [ ] Post path verdict compared to Pre — regressions called out explicitly
```
