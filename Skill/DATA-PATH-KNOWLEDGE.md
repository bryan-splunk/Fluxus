# Data Path Verification Knowledge Base

Use this file during **Step 0f** (pre-upgrade data path assessment) and **Step 4a** (post-upgrade
data path assessment).

---

## Session Isolation (MANDATORY)

Data path assessment is **scoped to the active project in the current skill run only**.

### Allowed sources

| Source | When |
|--------|------|
| Config YAML files listed in Step 0a | Pre (Step 0g) and Post (Step 4a) — **re-read from disk each time** |
| `DATA-PATH-KNOWLEDGE.md` | Procedure, ports, anti-pattern IDs |
| `Data Path Assessment — Pre.md` | **Post step only** — for Pre vs Post diff, and **only** if written in this same run |

### Forbidden sources

Do **not** use any of the following to determine paths, verdicts, or issues:

- Conversation history or prior chat turns (e.g. an earlier trace-path analysis in the same chat)
- Agent transcripts from other sessions
- `Data Path Assessment — Pre.md` or `Post.md` from a **previous** skill invocation
- `PreAssessment.md`, operational assessments, or READMEs from previous runs (except Step 4a Pre vs Post diff as above)
- Config files not in the Step 0a file list (other directories, old copies, “known” production paths)
- Assumed hostnames, ports, or roles not present in the in-scope YAML

### Execution rules

1. **Fresh read:** Open and read each in-scope YAML file at the start of Step 0g and again at Step 4a.
2. **Evidence citation:** Every path step in flow tables must cite `filename.yaml` and line-level
   detail from that read — not from memory or chat.
3. **Overwrite:** Always write a new `Data Path Assessment — Pre.md` / `Post.md`; never append to
   or amend a document from a prior run without full re-analysis.
4. **Same file list:** Pre and Post headers must list identical file sets (same as Step 0a).
5. **No external topology:** If an agent exporter targets `gateway.example.internal:4317` (or any
   hostname) but no matching gateway config is in scope, flag PATH-09 as “gateway not in scope —
   cannot verify receive leg” rather than inventing gateway pipeline details from chat or memory.

---

## Purpose

Verify that telemetry flows end-to-end correctly across agent, gateway, and standalone configs —
independent of the version-upgrade change catalogue. A config can pass every upgrade check and
still drop, duplicate, or misroute data.

---

## Standard Port Reference (Splunk OTel Agent → Gateway)

| Port | Typical listener | Expected signal types | Notes |
|------|------------------|----------------------|-------|
| 4317 | `otlp` gRPC | traces, logs, metrics | Standard OTLP gRPC |
| 4318 | `otlp` HTTP | traces, logs, metrics | Standard OTLP HTTP |
| 4319 | `otlp/gateway` gRPC | **metrics** (agent fan-in) | Often dedicated metrics listener — traces on this port are a red flag |
| 6060 | `http_forwarder` ingress | signalfx protocol | Bypasses gateway OTLP trace pipeline |
| 9943 | signalfx receiver (removed 0.153) / ingest | signalfx protocol | Legacy; use OTLP after upgrade |
| 9411 | zipkin | traces | |
| 14250+ | jaeger | traces | |

Non-standard agent listen ports (e.g. 4315/4316) are valid but must be documented — apps default to 4317/4318.

---

## Per-Signal Verification Procedure

Repeat for **traces**, **metrics**, and **logs**. For each agent-role (or standalone) file:

### Step 1 — Agent ingest
- List `service.pipelines.<signal>.receivers`
- Note listen ports for OTLP/Jaeger/Zipkin/file receivers

### Step 2 — Agent export
- List **every** exporter in `service.pipelines.<signal>.exporters`
- Resolve: exporter name → endpoint host:port → protocol (gRPC/HTTP) → TLS (`insecure`, certs)
- Flag **PATH-01** if multiple exporters on the same signal pipeline

### Step 3 — Gateway receive (skip if standalone / direct-to-cloud)
- Match agent exporter endpoint to a gateway receiver listen address
- Record which receiver definition owns that port (`otlp`, `otlp/gateway`, etc.)

### Step 4 — Gateway pipeline match
- Identify which `service.pipelines.*` references that receiver
- **Critical:** pipeline signal type must match (traces pipeline for trace data)
- Flag **PATH-02** if receiver is only in a pipeline for a different signal type

### Step 5 — Cloud export
- List gateway (or standalone) final exporter: `otlphttp`, `signalfx`, `splunk_hec`, etc.
- Resolve cloud endpoint (`${SPLUNK_INGEST_URL}/v2/trace/otlp`, realm-based signalfx, HEC URL)

### Step 6 — Bypass paths
- Check for `signalfx` exporter on agent sending to `http_forwarder` (:6060) while OTLP path also exists
- Flag **PATH-03** (traces-specific dual export) or **PATH-18** (any signal using forwarder bypass)

### Step 7 — Auth and token propagation
- Trace auth from agent export through gateway to cloud export:
  - Agent: `headers_setter` extension, `auth.authenticator`, inline `headers:` on exporter
  - Gateway: same on `otlphttp`, `signalfx`, `splunk_hec` cloud exporters
  - `batch.metadata_keys` referencing `X-SF-Token` / `X-SF-TOKEN`
- Flag **PATH-12** if agent attaches token but gateway cloud export does not
- Flag **PATH-13** if `metadata_keys` configured but exporter auth does not consume metadata

### Step 8 — Host metadata path
- Identify which pipeline exports host metadata (`signalfx` with `sync_host_metadata: true`, typically `metrics/internal`)
- Confirm path still works when app telemetry uses OTLP fan-in to gateway (not only direct signalfx)
- Flag **PATH-16** if no host-metadata export path exists in agent/gateway design
- Flag **PATH-17** if metadata path is split from app metrics path — document as intentional or issue

### Step 9 — Environment-variable cloud legs
- Note cloud export legs that depend on `${SPLUNK_INGEST_URL}`, `${SPLUNK_API_URL}`, `${SPLUNK_HEC_URL}`,
  `${SPLUNK_REALM}`, `${SPLUNK_HEC_TOKEN}`, `${SPLUNK_ACCESS_TOKEN}` without inline fallback
- Record “cannot verify outside YAML” — flag **PATH-24** if realm/URL-based signalfx export may be
  blocked by firewall (informational when URLs not in config)

### Step 10 — Logs cloud leg (when analysing logs signal)
- If logs reach gateway via OTLP/`file_log`/`windows_event_log`, confirm gateway `logs` pipeline exporter
- Verify `splunk_hec` has `token` and `endpoint` (or env var references) when HEC is the cloud leg
- Flag **PATH-20** if HEC exporter missing token/endpoint
- Flag **PATH-21** if `fluentforward` still in logs pipeline
- Flag **PATH-22** if `file_log` include paths are placeholders or OS-inappropriate

### Step 11 — Protocol match (agent export vs gateway receiver)
- For each OTLP leg: determine gRPC vs HTTP on agent exporter and gateway receiver
- Flag **PATH-14** for gRPC/HTTP mismatch on same logical hop
- Flag **PATH-15** specifically when agent targets `:4318` HTTP but gateway only listens gRPC `:4317`

### Step 12 — Multi-gateway and cross-agent consistency (when multiple files in scope)
- If multiple gateway configs exist, verify each agent exporter hostname matches an in-scope gateway
- Flag **PATH-25** if agents hardcode a gateway host not represented in scope or wrong region/DR file
- Flag **PATH-26** if multiple agent profiles use different export ports for the same signal without documented reason

---

## Anti-Pattern Catalogue

> Generic Splunk OTel deployment patterns — not tied to any customer environment. Flag only when
> the in-scope YAML exhibits the pattern; do not assume ports or hostnames from examples below.

| ID | Pattern | Severity | Likely symptoms |
|----|---------|----------|-----------------|
| PATH-01 | Multiple exporters on one agent signal pipeline | High | Duplicates, broken correlation, inflated volume |
| PATH-02 | Signal arrives at gateway receiver wired into wrong pipeline type | Critical | Missing or garbled data in Splunk UI |
| PATH-03 | `signalfx` exporter on traces pipeline alongside OTLP gateway path | High | Bypasses `otlphttp`; duplicate or inconsistent traces |
| PATH-04 | Agent exports traces/logs to a port whose gateway receiver is only in a metrics pipeline (e.g. dedicated `otlp/gateway` fan-in port) | Critical | Traces or logs land in wrong pipeline or are dropped |
| PATH-05 | Agent OTLP listen ports non-standard vs app instrumentation defaults | Medium | Partial or no data from instrumented apps |
| PATH-06 | Gateway pipeline missing enrichment agents already apply | Low | Inconsistent resource attributes |
| PATH-07 | Receiver or exporter defined but not referenced in any pipeline | Medium | Dead end / unused component |
| PATH-08 | Pipeline with no exporter configured | Critical | Data sink — enters but goes nowhere |
| PATH-09 | Agent exporter port has no matching gateway receiver | Critical | Connection failure, no data |
| PATH-10 | TLS mismatch between agent exporter and gateway receiver | Critical | Silent connection failure |
| PATH-11 | Duplicate path: same signal via two routes to same cloud destination | High | Duplicates in Observability / Cloud |
| PATH-12 | Agent export attaches `X-SF-TOKEN` but gateway cloud export (`otlphttp`/`signalfx`) missing matching auth headers or `headers_setter` | Critical | 401 / silent export failure at cloud |
| PATH-13 | `batch.metadata_keys` includes token key but exporter does not use `auth.authenticator` / headers | Medium | Token not propagated on batched export |
| PATH-14 | OTLP gRPC/HTTP protocol mismatch between agent exporter and gateway receiver | Critical | Connection failures or partial signal loss |
| PATH-15 | Agent OTLP HTTP to `:4318` but gateway exposes only gRPC on `:4317` (or vice versa) | Critical | No data despite reachable host |
| PATH-16 | No pipeline exports host metadata (`signalfx` + `sync_host_metadata`) while app data uses OTLP gateway fan-in | High | Hosts missing in Infrastructure / broken service map |
| PATH-17 | Host metadata via `metrics/internal` → `signalfx` but app metrics only via OTLP — split paths | Low | Confusing ops; document if intentional |
| PATH-18 | Any signal uses agent `signalfx` exporter → gateway `http_forwarder` (:6060) bypassing OTLP pipeline | High | Bypasses gateway processing; inconsistent attributes |
| PATH-19 | `http_forwarder` `egress` URL wrong (API vs ingest) while agents depend on forwarder for metadata/correlation | Medium | Partial metadata or wrong cloud destination |
| PATH-20 | Gateway logs pipeline uses `splunk_hec` but `${SPLUNK_HEC_TOKEN}` / `${SPLUNK_HEC_URL}` missing or empty | Critical | Traces/metrics work; logs do not |
| PATH-21 | `fluentforward` receiver still in active `logs` pipeline (FluentD removed from installers) | Critical | Log collection dead after install |
| PATH-22 | `file_log` include paths are placeholders or wrong OS (e.g. Linux paths on Windows agent) | Medium | Path wired but collects zero logs |
| PATH-23 | Metrics split across ports/hosts (e.g. host metrics `:4319`, app metrics `:4317`) without matching gateway pipelines | High | Partial metrics in Observability |
| PATH-24 | Gateway `signalfx` uses `realm:` / env URLs that may require firewall updates (post-0.151 domains) | Medium | Path correct in YAML; cloud leg blocked operationally |
| PATH-25 | Multiple gateway configs in scope but agents target hostname not matching intended gateway file | High | Data to wrong region or black hole |
| PATH-26 | Multiple agent profiles export same signal to different ports without documented design reason | Medium | Inconsistent behaviour across host types |
| PATH-27 | Gateway `filter` aggressively drops all data from a receiver agents actively send | High | Healthy connection, zero useful telemetry |
| PATH-28 | `tail_sampling` only on gateway; agent sends unsampled 100% traces | Low | High volume/cost; path still functional |

---

## Expected Paths (agent → gateway → cloud)

### Traces → Splunk Observability

```
App / instrumentation
  → Agent receiver (otlp / jaeger / zipkin)
  → Agent traces pipeline
  → Agent otlp/gateway exporter → Gateway host:4317 (gRPC)
  → Gateway otlp receiver (:4317)
  → Gateway traces pipeline
  → Gateway otlphttp exporter → ${SPLUNK_INGEST_URL}/v2/trace/otlp
  → Splunk Observability Cloud
```

Valid alternates (document explicitly):
- Agent direct `otlphttp` to cloud (no gateway)
- Standalone collector with local receivers → cloud exporters

Invalid when "OTLP via gateway" is the stated design:
- Agent `signalfx` exporter → gateway `http_forwarder` :6060 → cloud API (bypasses gateway `otlphttp`)

### Metrics → Splunk Observability

```
host_metrics / otlp receiver (agent)
  → Agent metrics pipeline
  → Agent otlp/gateway exporter → Gateway host:4319 (gRPC)  [or :4317]
  → Gateway otlp/gateway or otlp receiver
  → Gateway metrics pipeline
  → Gateway signalfx exporter → cloud (realm or explicit ingest URL)

Parallel host-metadata path (typical):
  prometheus/internal → metrics/internal pipeline → signalfx exporter (sync_host_metadata: true)
    → gateway http_forwarder :6060/:9943 OR direct cloud signalfx
```

Flag **PATH-16** if the parallel metadata path is missing. Flag **PATH-23** if host and app metrics use different export legs without gateway coverage for both.

### Logs → Splunk Cloud / Platform

```
file_log / windows_event_log / otlp receiver (agent)
  → Agent logs pipeline
  → Agent otlp/gateway exporter → Gateway host:4317 (gRPC)  [or :4319]
  → Gateway otlp receiver
  → Gateway logs pipeline
  → Gateway splunk_hec exporter → ${SPLUNK_HEC_URL}
  → Splunk Cloud Platform
```

Valid alternate: agent direct `splunk_hec` (standalone, no gateway).

Flag **PATH-20** if HEC token/endpoint missing. Flag **PATH-21** if `fluentforward` still used.

### Standalone / direct-to-cloud

```
receivers (local instrumentation + scrapers)
  → pipelines (traces / metrics / logs)
  → otlphttp / signalfx / splunk_hec exporters
  → Splunk Observability Cloud and/or Splunk Cloud Platform
```

No gateway in scope — verify each signal's internal path and cloud exporter auth (PATH-12).

---

## Processor Order Checks (per pipeline)

Known-good order:

| Position | Processor | Reason |
|----------|-----------|--------|
| First | `memory_limiter` | Gate intake before processing |
| Early | `k8s_attributes` / `resource_detection` | Enrich before filter/sample |
| Middle | `filter` | Drop before expensive transforms |
| Middle | `transform` | Transform surviving data |
| Before batch | `tail_sampling` | Sample before batching |
| Last | `batch` | Batch as late as possible |

| ID | Anti-pattern | Severity |
|----|--------------|----------|
| PROC-01 | `batch` before `filter` or `transform` | Medium |
| PROC-02 | `k8s_attributes` / `resource_detection` after `filter` | Medium |
| PROC-03 | `memory_limiter` not first in pipeline | High |
| PROC-04 | `tail_sampling` after `batch` | High |
| PROC-05 | `truncate` / `truncate_all` before `filter` | Medium |

---

## Standalone / Single-File Mode

When only one config file is provided:
- Trace each signal's internal path: receiver → pipeline → exporter → destination
- Skip cross-file matching (Steps 3–4, 12)
- Still run Steps 7–11 per signal
- Check PATH-01, PATH-07, PATH-08, PATH-12, PATH-20, PATH-21, PATH-22, and all PROC-* patterns

---

## Post-Assessment Comparison

When writing the Post data path document, compare against the Pre document:

| Compare | What to report |
|---------|----------------|
| Per-signal verdict | Improved / unchanged / regressed |
| PATH-* issues | Resolved / still open / newly introduced |
| End-to-end flow tables | Side-by-side or delta column |
| Overall verdict | Pre vs Post: HEALTHY / ISSUES / BROKEN |

If the user chose `none` (no config changes), Pre and Post should match unless configs were edited outside the skill run.
