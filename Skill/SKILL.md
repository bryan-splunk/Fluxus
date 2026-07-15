---
name: splunk-otel-upgrade
description: >-
  Upgrades Splunk OpenTelemetry Collector configuration files covering the v0.120–v0.153 change
  catalogue. Performs a Pre-Assessment scan of both active and commented-out config items against
  ALL known changes, then splits results into two buckets — changes within the user's target
  version (presented for selection and application) and future changes beyond the target (scanned
  against the config but never applied, shown so the user can decide whether to upgrade further).
  Commented-out items are identified separately and are NOT changed by default. Applies selected
  changes, writes per-file READMEs, performs an accuracy check, writes dedicated pre- and post-upgrade
  Data Path Assessment documents, and writes a multi-file operational assessment. Use when the user
  provides one or more OTel Collector YAML config files and asks to upgrade, assess, or prepare them
  for a Splunk collector version upgrade.
disable-model-invocation: true
---

# Splunk OTel Collector Upgrade — v0.120 → v0.153 Change Catalogue

## Before You Start

Read the full knowledge base and templates before touching any files:
- **[UPGRADE-KNOWLEDGE.md](UPGRADE-KNOWLEDGE.md)** — every change, version, category (P1 Breaking/P2 Degrading/P3 Advisory), rationale, and YAML snippets
- **[DATA-PATH-KNOWLEDGE.md](DATA-PATH-KNOWLEDGE.md)** — per-signal path verification procedure, port reference, anti-patterns
- **[TEMPLATES.md](TEMPLATES.md)** — PreAssessment, Data Path Assessment, README, and Operational Assessment output formats

> **Important — session isolation (applies to ALL steps, including data path):**
> Base every finding solely on the **active project workspace** in this session:
> - The YAML config files discovered or provided in Step 0a **only**
> - Read those files fresh from disk at the time of each assessment step
>
> **Do NOT use as evidence:**
> - Prior chat messages or conversation history (including trace-path discussions from earlier turns)
> - Agent transcripts from previous sessions
> - `Data Path Assessment — Pre.md` / `Post.md` from a previous skill run (unless re-read as part of *this* run's Pre vs Post comparison in Step 4a only)
> - `PreAssessment.md`, READMEs, or operational assessments from previous runs
> - Assumptions about deployment topology not stated in the config YAML at hand
>
> Each invocation is a clean, independent assessment. If a file is not in the workspace scope
> defined in Step 0a, it does not exist for path analysis.

---

## Step 0 — Pre-Assessment (REQUIRED before making any changes)

> This step is always performed first. No config changes are made until the user explicitly
> selects which changes to apply.

### 0a. Intake

Ask the user for (or infer from context):
1. **File path(s)** — absolute or relative paths to all YAML config files, OR if the user has
   opened a folder as a Cursor project without listing specific files, scan the workspace for all
   `.yaml` / `.yml` files and use those (exclude non-collector files like Kubernetes manifests,
   Helm values, or CI configs by checking for the presence of `receivers:`, `processors:`,
   `exporters:`, or `service:` top-level keys)
2. **Role** (optional) — for each file: what it does (agent, gateway, standalone, etc.); if not
   provided, infer from content (e.g. files with `otlp` exporters pointing to a remote host are
   likely agents; files with both receivers and exporters sending to Splunk directly are likely
   gateways or standalone)
3. **Target version** (optional) — the version the user wants to upgrade TO.
   - If not specified: **default = v0.153** — all changes in the catalogue are treated as current
     and eligible for application.
   - If specified (e.g. `v0.145`): changes with `introduced_version` ≤ v0.145 are **current
     changes** (eligible for application); changes with `introduced_version` > v0.145 are
     **future changes** (scanned and reported but never applied).
   - No source version is needed — the skill always scans the full catalogue from the beginning.
     If a change is already applied or not present in the config, the scan marks it Not Applicable.

Read all config files in full before proceeding.

### 0b. Scan Active Config for Applicable Changes

Scan **all** changes in `UPGRADE-KNOWLEDGE.md` (P1-01 through P1-19, P2-01 through P2-17, P3-01
through P3-10) against the active config, **regardless of target version**. Version filtering
happens in Step 0d — scan everything first.

For each change, check whether it is **applicable** to any **active (uncommented)** config items:

**Applicable** = the change is relevant to this config (the config contains the component, field, or
pattern described in "Look for"). Mark it with:
- ✅ **Applicable** — config contains the affected pattern in active code
- ➖ **Not Applicable** — component/pattern not present in this config at all
- ⚠️ **Possible** — component is present but applicability requires human confirmation

Also scan for:
- SmartAgent monitors from the Removed SmartAgent Monitors table
- Feature gate flags from the Feature Gate Removals table
- Pre-existing issues from the Common Pre-Existing Issues section (excluding data-path / topology
  issues — those are covered in Step 0f)

> **Data path / topology verification is NOT done here.** It runs in Step 0f after PreAssessment
> is written, and again in Step 4a after changes are applied. See `DATA-PATH-KNOWLEDGE.md`.

### 0c. Scan Commented-Out Config for Applicable Changes

**Perform a full second scan pass — this time targeting only commented-out lines.**
Scan ALL changes against commented-out content, regardless of target version. Version filtering
happens in Step 0d.

Commented-out config patterns appear in many forms. Treat any of the following as commented-out
and include them in this scan:
- Lines starting with `#` (with or without leading whitespace)
- Multi-line blocks where every line is commented (a full commented-out receiver, processor, or exporter block)
- Sub-settings deep within a larger block that are commented out individually — e.g. a receiver
  definition where the receiver key itself is active but some nested options like `metrics:`,
  `enabled:`, `topic:`, `batcher:`, etc. are commented lines inside it

**Depth requirement:** Look for affected patterns at every nesting level. Do not limit the scan to
top-level blocks only. For example:
- A `kafka:` receiver that is fully commented out
- A `kafka:` receiver that is active but has `default_fetch_size: 1048576` commented out inside it
- A `filter/exclude_dev:` processor that is active but has `error_mode:` commented out inside it
- A `splunk_hec:` exporter with a `batcher:` block that is commented out inside an active definition

For each change in `UPGRADE-KNOWLEDGE.md`, identify every occurrence in commented-out lines and
classify it using the same impact categories. Assign each a **CC-prefixed number** (CC1, CC2, CC3…)
so they can be referenced independently from active-config changes.

### 0d. Split Changes by Target Version and Build Numbered Lists

Using the target version established in Step 0a, split all applicable findings into two buckets:

**Current changes** (`introduced_version` ≤ target version, OR no target specified):
- These are eligible for application
- Assign sequential numbers: #1, #2, #3… for active config changes
- Assign CC-prefixed numbers: #CC1, #CC2… for commented-out changes
- Group within each prefix by: P1 Breaking first, then P2 Degrading, then P3 Advisory

**Future changes** (`introduced_version` > target version):
- These are scanned and reported for awareness only — **never applied**
- Assign F-prefixed numbers: #F1, #F2, #F3… for active config, #FCC1, #FCC2… for commented-out
- Group by: P1 Breaking first, then P2 Degrading, then P3 Advisory
- Include full scan results (which files affected, which patterns found) — same depth as current changes

Example (target: v0.145):
```
CURRENT CHANGES — eligible for application (→ v0.145):
  P1 Breaking (🔴):
    #1   P1-04 — signalfx receiver removed [affects: gateway.yaml line 12]
    #2   P1-13 — filter/transform error_mode default changed [affects: gateway.yaml line 45]

  P2 Degrading (🟡):
    #3   P2-08 — docker_observer Docker API upgrade [affects: agent.yaml]

  P3 Advisory (🔵):
    #4   P3-01 — Component renames: windowseventlog, hostmetrics [affects: agent.yaml]

COMMENTED-OUT CURRENT CHANGES:
    #CC1  P1-06 — OTLP exporter batcher block [commented in gateway.yaml ~line 201]

FUTURE CHANGES — scanned but NOT applied (v0.146 → v0.153):
  P1 Breaking (🔴):
    #F1   P1-07 — splunk_hec exporter batcher block removed (v0.151) [affects: gateway.yaml]
    #F2   P1-14 — OTTL SetMap error handling changed (v0.150) [affects: gateway.yaml]

  P2 Degrading (🟡):
    #F3   P2-14 — signalfx exporter URL domain change (v0.151) [affects: gateway.yaml]

  P3 Advisory (🔵):
    #F4   P3-02 — Kafka deprecated no-op fields (v0.153) [NOT in config — informational]

COMMENTED-OUT FUTURE CHANGES:
    #FCC1  P2-10 — windows_event_log event_data format (v0.148) [commented in agent.yaml ~line 55]
```

### 0e. Write the PreAssessment Document

Create `PreAssessment.md` in the working directory using the template in
`TEMPLATES.md § PreAssessment Template`.

The PreAssessment must include:
- Header: files assessed, date, version range
- Summary table: active counts (X P1 Breaking, Y P2 Degrading, Z P3 Advisory) and commented-out counts separately
- The full numbered change list for active config items (#1, #2…)
- A clearly separated section for commented-out config items (#CC1, #CC2…) with a prominent notice
  that these items are NOT changed by default
- A "Future Changes" section (Section 3) if a target version was specified, containing the full
  scan results for all #F and #FCC entries — changes beyond the target version, clearly marked
  as NOT applied
- A "Not Applicable" table listing checked-but-excluded changes and why
- A "Pre-Existing Issues" section (if any found — upgrade-related and config hygiene only, not data paths)
- A pointer to the Data Path Assessment document (see Step 0f)

### 0f. Pre Data Path Assessment (REQUIRED after Step 0e, before Step 0g)

> Runs **after** `PreAssessment.md` is written, **before** presenting the summary to the user.
> Read-only — no config changes. Uses the **original unmodified** config files.

**Scope lock — read before analysing paths:**

1. **Re-read** every config file listed in Step 0a from disk. Do not rely on memory from earlier
   in the conversation or from a previous skill run.
2. Analyse **only** those files. Do not pull in configs from other folders, prior projects, or
   chat history unless the user adds them to scope in Step 0a.
3. Do **not** read or cite an existing `Data Path Assessment — Pre.md` on disk from a prior run.
   Overwrite it with a fresh assessment every time.
4. Gateway hostnames, ports, and pipeline wiring must be quoted from the YAML — not inferred from
   what was discussed in chat.

Read **[DATA-PATH-KNOWLEDGE.md](DATA-PATH-KNOWLEDGE.md) § Session Isolation** and perform the full
per-signal verification for **traces**, **metrics**, and **logs**:

1. For each signal type, trace the complete active path across in-scope files only (agent → gateway → cloud,
   or standalone internal path)
2. Flag every anti-pattern (PATH-01 through PATH-28) and processor-order issue (PROP1-01 through PROP1-06)
3. Run verification Steps 7–12 from DATA-PATH-KNOWLEDGE.md (auth, host metadata, env legs, logs leg, protocol, multi-gateway)
4. Build end-to-end flow tables and ASCII or mermaid diagrams per signal type
5. Record cross-agent consistency when multiple in-scope agent configs are present
6. Assign an overall verdict per signal: ✅ HEALTHY / ⚠️ ISSUES / ❌ BROKEN

Create **`Data Path Assessment — Pre.md`** in the working directory using
`TEMPLATES.md § Data Path Assessment Template` (phase = Pre).

The document header must list **exactly** the same files as Step 0a — no others.

Do **not** duplicate this content inside `PreAssessment.md`. PreAssessment includes only a short
pointer to this file (see template Section 8).

### 0g. Present the Summary to the User

After writing `PreAssessment.md` and `Data Path Assessment — Pre.md`, present the summary in chat:

```
Pre-Assessment complete.

CURRENT CHANGES — eligible for application (→ v<target>):

  Active config:
    🔴 P1 Breaking: N  (startup failure or data loss if not addressed)
    🟡 P2 Degrading: N  (config or planning required)
    🔵 P3 Advisory:  N  (cleanup / informational)

  Commented-out config:
    🔴 P1 Breaking: N  (would cause failure if uncommented without fixing)
    🟡 P2 Degrading: N  (config or planning required if uncommented)
    🔵 P3 Advisory:  N  (cleanup / informational)
    ⚠️  Commented-out items are NOT changed unless you explicitly request it.

FUTURE CHANGES — scanned against your config but NOT applied (v<target+1> → v0.153):
    🔴 P1 Breaking: N  ← [N] affect your config — consider upgrading to v0.153 directly
    🟡 P2 Degrading: N  ← [N] affect your config
    🔵 P3 Advisory:  N  ← [N] affect your config
    ℹ️  Future changes are shown so you can decide whether to upgrade further.
       They will NEVER be applied in this run regardless of selection.

Full upgrade scan written to: PreAssessment.md
Data path (pre-upgrade) written to: Data Path Assessment — Pre.md

  Traces:  ✅ / ⚠️ / ❌  (N path issues — see Pre data path doc)
  Metrics: ✅ / ⚠️ / ❌
  Logs:    ✅ / ⚠️ / ❌

Review Data Path Assessment — Pre.md before selecting changes — path issues may explain
odd behaviour in Splunk today and are independent of the upgrade catalogue.

⚠️  BEFORE SELECTING: Please commit or back up your config files now.
    The skill will begin modifying YAML files as soon as you make a selection.
    To revert: restore your backed-up files and restart the collector service.

Which CURRENT changes would you like to apply?

  Active config changes:
    • "all"       — apply all current active changes
    • "p1"        — apply only P1 Breaking current active changes
    • "p2"        — apply only P2 Degrading current active changes
    • "p3"        — apply only P3 Advisory current active changes
    • "1, 3, 5"   — apply specific current changes by number

  To also process commented-out items, add "include comments":
    • "all include comments"   — apply all current active + all current commented-out
    • "p1 include comments"    — apply P1 Breaking active + P1 Breaking commented-out
    • "1, CC2, CC4"               — mix specific active and commented-out by number
    • "CC1, CC3"                  — only specific commented-out changes

  • "none" — skip all changes (assessment only)

  Note: #F and #FCC numbers are future changes and cannot be selected for this run.
```

**Wait for the user's response before proceeding.**

---

## Step 1 — Apply Selected Changes

Based on the user's response from Step 0g, determine the set of changes to apply.

### 1a. Interpret the User's Selection

**Future changes (#F and #FCC numbers) are NEVER applied regardless of selection.**
If the user references an F-prefixed number, inform them it is a future change and cannot be
applied in this run.

| User Response | Active Config | Commented-Out Config |
|---|---|---|
| `all` | Apply all **current** active changes | ❌ Not touched |
| `p1` | Apply only P1 Breaking **current** active | ❌ Not touched |
| `p2` | Apply only P2 Degrading **current** active | ❌ Not touched |
| `p3` | Apply only P3 Advisory **current** active | ❌ Not touched |
| `1, 3, 5` | Apply specific numbered **current** active changes | ❌ Not touched |
| `all include comments` | Apply all current active changes | Apply all **current** commented-out changes |
| `p1 include comments` | Apply only P1 Breaking current active | Apply only P1 Breaking **current** commented-out |
| `p2 include comments` | Apply only P2 Degrading current active | Apply only P2 Degrading **current** commented-out |
| `p3 include comments` | Apply only P3 Advisory current active | Apply only P3 Advisory **current** commented-out |
| `1, CC2, CC4` | Apply current active changes #1 | Apply current commented-out #CC2 and #CC4 |
| `CC1, CC3` | ❌ No active changes | Apply only specific current commented-out |
| `F1` or `FCC1` | ❌ Future change — not applicable | ❌ Future change — not applicable |
| `none` | ❌ No changes applied | ❌ Not touched — skip Step 1; still run Steps 4a–4b (Post data path = same as Pre) |

**Default rule: commented-out lines are never modified unless the user's response includes
"include comments" or references CC-prefixed numbers.**

### 1b. Work Through Each Selected Active Change

For each selected active change, in order (P1 Breaking → P2 Degrading → P3 Advisory, then by number):

1. Open the relevant config file(s)
2. Make the exact change described in `UPGRADE-KNOWLEDGE.md` for that change code
3. Apply it to **every active occurrence** — definitions and pipeline references
4. Leave a brief comment in the YAML where a significant removal occurred (e.g., removed receiver)

**Cross-file dependency check — required after any receiver removal (P1-01, P1-02, P1-03, P1-04, P1-05):**

When a receiver or exporter is removed from one file, scan **all other in-scope files** for
components that reference that removed receiver's port or protocol:

- If you remove a `signalfx:` receiver (P1-04) from a gateway config, scan every in-scope agent
  config for `signalfx:` exporters pointing to that gateway's port 9943 — those agents must be
  updated to send OTLP to port 4317/4318 instead.
- If you remove a `sapm:` receiver (P1-02) from a gateway, scan agent configs for `sapm:` exporters
  sending to port 7276 on that gateway.
- If you remove a `routingprocessor` (P1-01) and add a routing connector, verify that all other
  pipelines referencing the connector exist in the `service.pipelines` section.

For each cross-file dependency found:
1. Apply the fix to the dependent file in the same pass
2. Note the cross-file impact in that file's per-file README
3. Flag it in the Operational Assessment under "Cross-File Changes"

### 1c. Work Through Each Selected Commented-Out Change

For each selected CC-prefixed change, in order (P1 Breaking → P2 Degrading → P3 Advisory, then by CC number):

1. Locate the commented-out line(s) identified in the Pre-Assessment
2. Update the commented-out code to reflect the correct post-upgrade form
3. Leave the lines **still commented out** — do not uncomment them; only fix the content
4. Add a short inline comment indicating what was changed, e.g.:
   `# [upgraded from: topic: my-topic — removed in 0.141, use topics: [my-topic]]`
5. If the commented-out block would cause a startup failure if uncommented without this fix,
   add a warning comment above it:
   `# ⚠️ UPGRADE NOTE: This block contains a breaking change from v0.141. Fix below before uncommenting.`

### 1d. Renames (P3-01)

If renames are selected:
- **Active:** Update every occurrence in definitions, `service.pipelines` references
- **Active + include comments:** Also update the old names inside commented-out blocks
- Use the full renames table in `UPGRADE-KNOWLEDGE.md § P3-01`

### 1e. Feature Gate Removals

If feature gate cleanup is selected (or `all`):
- Search for flags in service startup scripts, systemd unit files, Windows service registry,
  Helm chart values, or Kubernetes manifests alongside the config files
- Remove any flags listed in `UPGRADE-KNOWLEDGE.md § Feature Gate Removals`
- Note that startup configuration outside the YAML may need to be updated separately

---

## Step 2 — Write Per-File README

After all selected changes are applied to a file, create or update its README using the template in
`TEMPLATES.md § Per-File README Template`.

The README must include:
- Applied Changes Log table (what changed, ✅ Applied / ⚠️ Pending / N/A / Not selected)
- One section per applied change with: what changed, why (version + rationale), and what was done
- "Things That Were Checked But Not Applicable" table
- Summary table of all selected changes and their status

---

## Step 3 — Accuracy Check

Re-read the modified config file(s) and verify the applied changes are correct.
For each change that was applied, verify:

**For P1 Breaking changes (active):**
- [ ] Removed component definitions are fully absent (or replaced with a comment block)
- [ ] All pipeline references to removed components are updated
- [ ] No startup-failure patterns remain (old batcher keys, removed config fields, old receiver names)

**For P2 Degrading changes (active):**
- [ ] Config edits are syntactically valid YAML
- [ ] Operational actions are noted in the README (firewall updates, env var changes, etc.)

**For P3 Advisory/Renames (active):**
- [ ] Old names are absent from active definitions AND pipeline lists
- [ ] No old names remain in active code (scan: hostmetrics, windowseventlog, fluentforward, resourcedetection, etc.)

**For Commented-Out changes (if "include comments" was selected):**
- [ ] Each updated commented-out block still has every line commented out (no accidental uncommenting)
- [ ] The corrected content inside the comment is syntactically valid if it were to be uncommented
- [ ] Upgrade note comments were added where a breaking change would occur on uncommenting
- [ ] Old component names inside comments are updated (if renames were included)

**OTTL changes (if applied):**
- [ ] filter/transform processors have explicit `error_mode`
- [ ] OTTL `set()` calls on map/slice fields have been reviewed

Update the README Applied Changes Log to mark anything found during this pass.

**After the accuracy check is complete, prompt the user:**

```
Accuracy check complete. All selected changes verified in the modified config files.

Before proceeding to the Post Data Path Assessment, please validate the config syntax:

  Linux / Docker:
    otelcol validate --config=<path-to-config.yaml>
    # or via Docker:
    docker run --rm -v $(pwd)/<config>.yaml:/etc/otelcol/config.yaml \
      quay.io/signalfx/splunk-otel-collector:0.153.0 \
      otelcol validate --config=/etc/otelcol/config.yaml

  Windows:
    & "C:\Program Files\Splunk\OpenTelemetry Collector\otelcol.exe" `
      validate --config="<path-to-config.yaml>"

If otelcol validate reports errors, paste them here and I will fix them before continuing.
If it passes (or if you cannot run it now), reply "continue" and I will proceed to the
Post Data Path Assessment.
```

**Wait for the user's response before proceeding to Step 4a.**

---

## Step 4 — Post-Assessment

### 4a. Post Data Path Assessment (REQUIRED)

After Step 3 (accuracy check), re-run the **same** data path verification from Step 0f against the
**modified** config files currently on disk in the active project.

**Scope lock — same rules as Step 0f:**

1. **Re-read** every in-scope config file from disk (post-modification). Do not reuse Step 0f
   conclusions from memory — derive Post findings fresh from the modified YAML.
2. Analyse **only** the files listed in Step 0a. No other configs, no chat history.
3. For the **Pre vs Post comparison** section only: you may read `Data Path Assessment — Pre.md`
   written **in this same skill run** to diff verdicts and PATH-* IDs. Do **not** read a Pre.md
   left over from a previous run if this run did not create it in Step 0f.
4. If Step 0f was skipped or Pre.md is missing from this run, state that in Post.md and skip the
   comparison section — do not reconstruct Pre from memory or old files.

Read `DATA-PATH-KNOWLEDGE.md` § Session Isolation again.

Create **`Data Path Assessment — Post.md`** using `TEMPLATES.md § Data Path Assessment Template`
(phase = Post).

The Post document must include:
- Full per-signal path analysis (same depth as Pre), derived from modified YAML on disk — including
  Steps 7–12 and PATH-01 through PATH-28 / PROP1-01 through PROP1-06
- A **Pre vs Post comparison** section (when Pre.md from this run exists): which PATH-* issues
  resolved, which remain, any newly introduced
- Overall verdict delta (e.g. Traces: ❌ BROKEN → ✅ HEALTHY)

If the user selected `none`, re-read unchanged configs from disk; Post should match Pre unless
configs were edited outside the skill.

### 4b. Operational Assessment

After Step 4a, write `Readme operational assessment.md` using
`TEMPLATES.md § Operational Assessment Template`.

The assessment must cover:
1. **Pre-Assessment summary** — counts and which changes were selected to apply
2. **Per-file accuracy summary** — one table row per file: ✅ / ⚠️ / ❌ and summary
3. **Full change log** — every applied change with file, status, and any pending actions
4. **Data path summary** — short pointer to Pre and Post data path documents with verdict delta
5. **P1 Breaking issues** — startup failure or data loss (upgrade-related; path issues → data path docs)
6. **Issues requiring decisions** — items needing a human choice
7. **P3 Advisory observations** — cleanup, security (not duplicated from data path docs)
8. **Dashboard and alert impact** — concrete list of metric/attribute changes that break existing dashboards (P2-11, P2-12, P2-17); N/A if none apply
9. **Infrastructure and environment actions** — consolidated table of all work required outside config YAML (firewall, env vars, Kafka ACLs, Prometheus version, Docker version, MSI URLs, feature gate non-YAML locations)
10. **Feature gate locations** — where to remove removed gates in non-YAML startup configuration
11. **Rollback instructions** — how to revert if the upgrade causes issues
12. **Pre/post-upgrade checklist** — validation steps
13. **Prioritized action table** — upgrade and operational items ranked by severity
14. **Validation commands** — ready-to-run commands for otelcol validate and service checks

> **Do not duplicate** architecture diagrams, end-to-end flow tables, connectivity checks, dead-end
> detection, or processor-order analysis in the Operational Assessment. Those live exclusively in
> `Data Path Assessment — Pre.md` and `Data Path Assessment — Post.md`.
>
> **Do not duplicate** the Infrastructure/Environment Actions table, Feature Gate Locations table,
> Dashboard/Alert Impact table, or Rollback Instructions from the Operational Assessment in any
> per-file README. Per-file READMEs reference these sections by number instead.

---

## Output Summary

| Artifact | When created |
|---|---|
| `PreAssessment.md` | Step 0e — upgrade catalogue scan; pointers to data path doc |
| `Data Path Assessment — Pre.md` | Step 0f — after PreAssessment, before user selection; original configs |
| Modified config YAML(s) | Step 1 — only for selected changes; commented-out lines only modified if "include comments" was specified |
| `Readme <filename>.md` per config | Step 2, updated in Step 3 |
| `Data Path Assessment — Post.md` | Step 4a — after accuracy check; modified configs; includes Pre vs Post comparison |
| `Readme operational assessment.md` | Step 4b — upgrade accuracy + pointers to both data path documents |
