# How to Update the Standalone HTML Guide

**File:** `splunk-otel-upgrade-guide.html`  
**Audience:** Anyone (human or AI agent) adding a new Splunk OTel Collector release to the project.

This file is a hand-crafted, self-contained Alpine.js SPA (~2000 lines). It is never auto-generated.
Update it after all rule YAML files, test fixtures, and `Skill/UPGRADE-KNOWLEDGE.md` entries are
already complete and tested — treat this as the final documentation step for a new release.

---

## When to Update

Update this file when a new collector version is ingested (following `RELEASE-INTAKE.md`).
Do **not** modify it for:
- Engine code changes or refactors
- Rule schema changes
- CLI flag additions
- Any change that does not alter the set of documented breaking changes

---

## Step 1 — Update the Version Range (3 locations)

### 1a. `<title>` tag (line 6)
```html
<title>Splunk OTel Collector Upgrade Guide — v0.120.0 → v0.153.0</title>
```
Change the upper bound to the new version.

### 1b. Header pill badge (line 72)
```html
<span class="pill" style="...">v0.120.0 → v0.153.0</span>
```
Change the upper bound to the new version.

### 1c. Header description paragraph (line 74)
```html
<p ...>Comprehensive reference ... across 33 releases.</p>
```
Update the release count (`33 releases` → the new total number of releases covered).

---

## Step 2 — Add Rows to the "Breaking Changes by Release" Overview Table

The overview table is in the `overview` section (starting around line 140). Add one row per new
version at the **bottom** of the `<tbody>`. Use the appropriate colour class based on severity:

| Severity | Class |
|---|---|
| High-impact (P1 startup failures / data loss) | `class="text-red-400 font-semibold"` on `<td>` for version; `class="text-red-300 font-medium"` on changes `<td>` |
| Medium-impact (P2 config/planning changes) | `class="text-yellow-400"` on version `<td>` |
| Low-impact (P3 advisory only) | no class (default neutral style) |

Row template:
```html
<tr><td class="font-mono text-blue-400">0.154</td><td>Short summary of key changes for this version</td></tr>
```

For high-impact releases:
```html
<tr><td class="font-mono text-red-400 font-semibold">0.157</td><td class="text-red-300 font-medium">Summary of breaking changes</td></tr>
```

---

## Step 3 — Add Change Cards to the Appropriate Section

Each breaking change gets a card in the section that best matches its component type. The sections
and their Alpine.js IDs are:

| Section | `active ===` value | Types of changes |
|---|---|---|
| Overview | `overview` | Release-by-release table only (Step 2) |
| Component Renames | `renames` | `name_change`-type changes (snake_case renames) |
| Kafka Migration | `kafka` | Any `receivers.kafka` or `exporters.kafka` changes |
| Processors | `processors` | Any `processors.*` changes |
| Receivers | `receivers` | Any `receivers.*` changes (non-kafka) |
| Exporters | `exporters` | Any `exporters.*` changes |
| OTTL Changes | `ottl` | Any OTTL function or type changes |
| Splunk-Specific | `splunk` | Smart Agent monitors, Splunk extensions, Splunk URLs, FluentD |
| Upgrade Checklist | `checklist` | No rule cards — checklist items only |

### Card HTML template

Every card follows this structure. Copy an existing card from the relevant section as a starting
point — do not guess at the CSS classes.

```html
<!-- P1-NN: short description (version) -->
<div class="card overflow-hidden">
  <div class="px-4 py-2 border-b border-neutral-700 flex items-center justify-between">
    <span class="text-xs font-semibold text-neutral-400 uppercase tracking-wide">ComponentName — What Changed (0.NN)</span>
    <span class="pill badge-p1">P1 Breaking</span>
    <!-- use badge-p2 or badge-p3 for lower-priority changes -->
  </div>
  <div class="p-4">
    <div class="callout callout-danger mb-3">One-sentence impact statement.</div>
    <p class="text-sm text-neutral-300 mb-3"><strong>Action:</strong> What the user must do.</p>
    <pre class="code-block"># BEFORE:
receivers:
  component:
    old_field: value

# AFTER:
receivers:
  component:
    new_field: value</pre>
  </div>
</div>
```

### Badge classes

| Priority | Badge class |
|---|---|
| P1 Breaking | `badge-p1` |
| P2 Degrading | `badge-p2` |
| P3 Advisory | `badge-p3` |

### Callout classes

| Severity | Class |
|---|---|
| Danger (startup failure / data loss) | `callout callout-danger` |
| Warning (degrading, planning required) | `callout callout-warning` |
| Info (advisory) | `callout callout-info` |

---

## Step 4 — Update the Upgrade Checklist (if needed)

The checklist section (`active === 'checklist'`) contains ordered steps. If the new release adds
a fundamentally new type of action (e.g. a new external system to update, a new package to
install), add a checklist item. For routine rule additions, no checklist update is needed.

---

## Step 5 — Verify in a Browser

Open the file locally:
```powershell
start splunk-otel-upgrade-guide.html
```

Check:
- The version range pill in the header shows the new version
- The new release row appears in the Overview table
- Clicking each tab shows the new cards in the correct sections
- No JavaScript errors in the browser console (F12 → Console)

---

## Source of Truth for Content

All rule content (before/after YAML, impact statements, action prose) comes from
`Skill/UPGRADE-KNOWLEDGE.md`. The `### P1-NN · …` entries are the canonical source — copy
content from there rather than inventing it.

Do not add content to the HTML that is not already in `Skill/UPGRADE-KNOWLEDGE.md`.

---

## What Was NOT Auto-Generated

This file was hand-authored in the initial commit. There is no build script or template engine.
An AI agent (Cursor) can add new cards by following the patterns above — the consistent card
structure makes this straightforward for well-scoped additions (one release at a time).
