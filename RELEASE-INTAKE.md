# Release Intake Runbook

**Purpose:** Step-by-step process for ingesting a new Splunk OTel Collector release and adding the corresponding upgrade rules to FLUXUS.

**Average effort per release:** ~9 P1 rules + ~8 P2/P3 rules = ~17 entries, typically 2–4 hours.

---

## Step 0 — Determine the New Version and Locate Release Notes

1. Check the [Splunk OTel Collector releases](https://github.com/signalfx/splunk-otel-collector/releases) page.
2. Note the new version number (e.g. `0.154`). The previous version in the rule set is the `introduced` ceiling.
3. Open the GitHub release notes. Also review:
   - The upstream [opentelemetry-collector-contrib CHANGELOG](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CHANGELOG.md)
   - The upstream [opentelemetry-collector CHANGELOG](https://github.com/open-telemetry/opentelemetry-collector/blob/main/CHANGELOG.md)
4. Identify every **breaking change**, **deprecation**, and **behavior change** mentioned. Use the criteria in Step 1 to classify each.

---

## Step 1 — Classify Each Change as P1 / P2 / P3

| Priority | Criteria |
|---|---|
| **P1 Breaking** | Startup failure OR silent data loss if not fixed. Field removals, required renames, protocol/schema changes that crash the collector. |
| **P2 Degrading** | Requires config change, planning, or operational action. Behavior changes that cause incorrect data, ACL mismatches, dashboard breakage, or external system updates. Deprecated fields that still work but will break in a future release go here. |
| **P3 Advisory** | No immediate failure. Informational: feature gate removals that need cleanup, renamed aliases that still work, new opt-in capabilities. |

**When uncertain:** prefer P1 (safer to over-flag than miss a startup failure).

---

## Step 2 — Identify the YAML Config Path(s) Affected

For each change, determine:

1. Which component type is affected (receiver, processor, exporter, connector, extension)?
2. Which specific component name (e.g. `kafka`, `signalfx`, `resourcedetection`)?
3. What is the exact YAML path to the removed/renamed/added field?
   - Use JSONPath notation: `$.receivers.kafka.topic`
4. Is this detectable by key existence, value pattern, or regex on raw YAML?
5. Does it also apply to commented-out config sections (`in_comments: true`)?

---

## Step 3 — Write the Rule YAML File

Create a new file in `rules/` (or `rules/security/` for credential-related rules).

**File naming:** `p1-NN.yaml`, `p2-NN.yaml`, `p3-NN.yaml`, `sec-p1-NN.yaml`  
Use the next available sequence number in the appropriate priority group.

### Rule YAML Template

```yaml
id: P1-NN            # P1-NN, P2-NN, P3-NN, or SEC-P1-NN — increment from highest existing
category: p1         # p1, p2, or p3
introduced: "0.154"  # the version this change was introduced
title: "ComponentName — What Changed (0.154)"
look_for:
  # Primary detection: existence of the removed/changed field
  - path: "$.receivers.component.removed_field"
    match: exists
  # Optional: also match in commented-out sections
  - path: "$.receivers.component.removed_field"
    match: exists
    in_comments: true
  # Alternative: raw regex when JSONPath is insufficient
  # - raw_pattern: "(?m)some_pattern_here"
migration:
  # strategy options:
  #   key_move   — engine renames the key automatically
  #   key_delete — engine removes the key automatically
  #   guided     — requires manual migration; engine reports but does not auto-apply
  #   inform_only — annotation injected into comments; no YAML change
  strategy: guided     # or: key_move, key_delete, inform_only
  key_moves:
    # For automated renames (strategy: key_move or as supplemental comment injection):
    - from: $.receivers.component.old_field
      to:   $.receivers.component.new_field
      # OR for comment-only annotation (any strategy):
    - comment_path: $.receivers.component
      comment_text: |
        # UPGRADE(P1-NN v0.154): <short summary of what changed and what to do>
        # ACTION: <one-line action>
  before: |
    # BEFORE — the problematic config:
    receivers:
      component:
        old_field: value
  after: |
    # AFTER — the corrected config:
    receivers:
      component:
        new_field: value
description: >-
  One to three sentences describing what the engine detects and what the automated
  action does (or why manual action is required). Written for a user reading the
  pre-assessment report card. Should answer: "What is this finding and what happens
  if I ignore it?"
guidance: |
  Extended operational guidance for users who need more context. Include:
  - The "Action:" prose from UPGRADE-KNOWLEDGE.md
  - Accuracy checks (how to verify the change was applied correctly)
  - Transition-period notes (feature gates, temporary workarounds)
  - Actions outside the config file (firewall updates, ACL changes, certificate
    reissuance, dashboard updates, environment variable changes)
  Do NOT duplicate the before/after YAML snippets here — those are already in migration:.
see_also:
  - P1-NN   # related rules, if any
```

### Detection Strategies Quick Reference

| Scenario | look_for approach |
|---|---|
| Field exists and must be removed | `match: exists` on the field path |
| Field must be renamed | `match: exists` on the old field path; strategy: `key_move` |
| Value must match a pattern | `match: regex`, `pattern: "..."` on the field path |
| Cannot express as JSONPath | `raw_pattern: "(?m)regex"` |
| Only flag in commented sections | Add same look_for entry with `in_comments: true` |

---

## Step 4 — Write the Test Fixture

Create a pair of fixture files in `testdata/rules/`:

**Naming convention:**
- Input file: `testdata/rules/<rule-id>-before.yaml` (e.g. `p1-27-before.yaml`)
- Expected output: `testdata/rules/<rule-id>-after.yaml` (only needed for `key_move`/`key_delete` strategies)

### Test Fixture Template

**`testdata/rules/p1-NN-before.yaml`** — minimal config that triggers the rule:
```yaml
# Test fixture for P1-NN — <title>
receivers:
  component:
    old_field: some_value   # this should be detected

service:
  pipelines:
    traces:
      receivers: [component]
      processors: []
      exporters: [logging]
```

**`testdata/rules/p1-NN-after.yaml`** — expected config after automated apply (key_move / key_delete only):
```yaml
receivers:
  component:
    new_field: some_value   # renamed by the engine

service:
  pipelines:
    traces:
      receivers: [component]
      processors: []
      exporters: [logging]
```

For `guided` and `inform_only` strategies, the after fixture should match the before fixture (plus any injected UPGRADE comment).

### Adding the Test Case

Open `engine/engine_test.go` and add a test case to the appropriate test table:

```go
// In TestDryRun or the relevant test function:
{
    name:     "P1-NN detects component.old_field",
    input:    readFixture(t, "p1-NN-before.yaml"),
    ruleFile: "p1-NN",
    wantIDs:  []string{"P1-NN"},
},
```

---

## Step 5 — Add the Entry to `Skill/UPGRADE-KNOWLEDGE.md`

Add a new section to the appropriate priority group (P1, P2, or P3) in `Skill/UPGRADE-KNOWLEDGE.md`.

Use this heading format:
```markdown
### P1-NN · ComponentName — What Changed (0.154)

**Look for:** Description of what to look for in the config.

**Impact:** One sentence describing the failure mode or data loss.

**Action:** Prose description of what to do — this text becomes the rule's `guidance:` field.

```yaml
# BEFORE:
receivers:
  component:
    old_field: value

# AFTER:
receivers:
  component:
    new_field: value
```

> Optional blockquote for accuracy checks, transition-period notes, or actions outside
> the config file (firewall changes, ACL updates, dashboard updates, etc.)
```

---

## Step 6 — Run Tests

```bash
go test ./engine/...
```

All tests must pass before proceeding. If a new test fails:
1. Check the `look_for` path — use a YAML path tester or add a debug print to verify the JSONPath expression.
2. Verify the fixture file matches exactly what the rule expects to detect.
3. For `key_move` rules, verify the `from` and `to` paths are correct.

---

## Step 7 — Update Version Range References

Update the version range in all places that display the supported release window:

1. **`Skill/UPGRADE-KNOWLEDGE.md` header** — update the version range in the first line:
   ```
   # Splunk OTel Collector Upgrade Knowledge Base — v0.120 → v0.154
   ```

2. **`splunk-otel-upgrade-guide.html`** (if it exists) — update any version range displayed in the UI.

3. **`splunk-otel-upgrade-guide.canvas.tsx`** (if it exists) — update the version range in the canvas header.

---

## Step 8 — Update `Skill/UPGRADE-KNOWLEDGE.md` Header Version Range

Confirm the header line of `Skill/UPGRADE-KNOWLEDGE.md` was updated in Step 7.

The format is:
```
# Splunk OTel Collector Upgrade Knowledge Base — v0.120 → v0.154
```

Replace `v0.154` with the new release version.

---

## Step 9 — Commit

Use the following commit message convention:

```
feat(rules): add P1/P2/P3 rules for v0.154

- P1-NN: ComponentName removed (0.154)
- P2-NN: ComponentName behavior change (0.154)
- P3-NN: ComponentName alias deprecated (0.154)
- Updated UPGRADE-KNOWLEDGE.md version range to v0.120 → v0.154
```

---

## Checklist

Copy this checklist into your tracking ticket or PR description:

```
Release Intake Checklist — v0.XXX

[ ] Step 0: Located release notes and identified all breaking/degrading/advisory changes
[ ] Step 1: Classified each change as P1 / P2 / P3
[ ] Step 2: Identified YAML path(s) for each change
[ ] Step 3: Rule YAML files written (one per change)
            - id, category, introduced fields correct
            - look_for paths verified
            - migration strategy correct (guided/key_move/key_delete/inform_only)
            - comment_text included for inform_only and guided rules
            - before/after YAML snippets present
            - description written (engine-facing, 1-3 sentences)
            - guidance written (operator-facing, action prose + accuracy checks)
[ ] Step 4: Test fixtures written in testdata/rules/
            - before.yaml triggers the rule
            - after.yaml matches expected output (key_move/key_delete only)
            - Test case added to engine_test.go
[ ] Step 5: UPGRADE-KNOWLEDGE.md entries added in correct P1/P2/P3 section
[ ] Step 6: go test ./engine/... passes (all tests green)
[ ] Step 7: Version range references updated
[ ] Step 8: UPGRADE-KNOWLEDGE.md header version range updated
[ ] Step 9: Committed with correct message format
```

---

## Tips and Gotchas

### JSONPath Detection Tips
- Use `$.receivers.kafka` (not `$.receivers["kafka"]`) — the engine uses standard dot notation.
- For named component instances (e.g. `kafka/myname`), the path `$.receivers.kafka` also matches `$.receivers["kafka/myname"]` via the engine's partial-key matching.
- When a field can be deeply nested, prefer `raw_pattern` with a multiline regex over a deeply nested JSONPath.

### Strategy Selection
- Use `key_move` only when the rename is purely mechanical and the engine can reliably find both old and new paths.
- Use `guided` for any change requiring the user to make decisions (new pipeline topology, choosing between options, external system changes).
- Use `inform_only` when no YAML change is required but you need to inject an annotation and surface the finding in the report.
- Use `key_delete` when a field must simply be removed with no replacement.

### Guidance vs Description
- **description:** What the engine detected and what it does automatically. Written for the assessment report card. 1-3 sentences.
- **guidance:** What the operator should do after seeing the finding. Includes action prose, accuracy checks, transition notes, external actions. Displayed only when `--include-guidance` is used.

### Security Rules
- Security rules live in `rules/security/` with IDs like `SEC-P1-NN`.
- They use `introduced: "0.0.0"` (no version dependency — always applicable).
- They always use `strategy: inform_only`.
- Add the entry to `Skill/UPGRADE-KNOWLEDGE.md` under a `## Security` section if one does not exist.
