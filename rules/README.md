# FLUXUS Rule Authoring Guide

This folder contains one YAML file per breaking change. The engine loads every
`*.yaml` file here at startup, evaluates each rule against the user's config
files, and produces a Pre-Assessment report and (optionally) an automatically
migrated output.

---

## Table of Contents

1. [File naming](#1-file-naming)
2. [Rule schema reference](#2-rule-schema-reference)
3. [look_for — detection](#3-look_for--detection)
   - 3.1 [YAMLPath syntax](#31-yamlpath-syntax)
   - 3.2 [Named component instances](#32-named-component-instances)
   - 3.3 [Match types](#33-match-types)
   - 3.4 [Scanning commented-out blocks](#34-scanning-commented-out-blocks)
   - 3.4a [Raw text scanning with raw_pattern](#34a-raw-text-scanning-with-raw_pattern)
   - 3.5 [logic — OR vs AND](#35-logic--or-vs-and)
4. [migration — automated fix](#4-migration--automated-fix)
   - 4.1 [strategy](#41-strategy)
   - 4.2 [key_moves (Option A)](#42-key_moves-option-a)
   - 4.3 [before / after (string fallback)](#43-before--after-string-fallback)
5. [Common patterns](#5-common-patterns)
6. [Rule phases and subdirectories](#6-rule-phases-and-subdirectories)
7. [Adding a new rule — step by step](#7-adding-a-new-rule--step-by-step)
8. [Testing your rule](#8-testing-your-rule)
9. [Known limitations](#9-known-limitations)

---

## 1. File naming

| Convention | Meaning |
|---|---|
| `p1-NN.yaml` | P1 Breaking — startup failure or silent data loss if not fixed |
| `p2-NN.yaml` | P2 Degrading — config change, planning, or operational action required |
| `p3-NN.yaml` | P3 Advisory — no immediate failure; cleanup or informational only |
| `rules/security/sec-p1-NN.yaml` | Security / credential hygiene check (evergreen, P1 severity) |
| `rules/pipeline/pipe-p1-NN.yaml` | Pipeline topology / anti-pattern check (future) |

Use the next unused number within the category. IDs in the YAML `id:` field must
be unique across all files and all subdirectories.

---

## 2. Rule schema reference

```yaml
# ── Required ──────────────────────────────────────────────────────────────────
id: P1-99                        # Unique rule identifier (matched in approval lists)
category: p1                    # p1 (breaking) | p2 (degrading) | p3 (advisory)
introduced: "0.153"             # Collector version that introduced this change
                                # Use "0.0.0" for evergreen security rules (always applicable)
title: "Short human-readable title"

# ── Phase (optional — inferred from directory if omitted) ─────────────────────
phase: upgrade                  # upgrade (default) | security | pipeline | post_assessment

# ── Detection ─────────────────────────────────────────────────────────────────
logic: or                       # or (default) | and   — how selectors are combined
look_for:
  - path: $.some.yaml.path
    match: exists               # exists | absent | value | pattern
    value: ""                   # required only when match: value
    pattern: ""                 # regex, required only when match: pattern
    in_comments: false          # legacy: exclude this selector from the ACTIVE eval
                                # (comment matching is auto-derived — see §3.4)
  # OR — use raw_pattern for credential / content scanning (ignores path+match):
  # NOTE: Go uses RE2 — lookahead/lookbehind (?!...) (?<!...) are NOT supported.
  - raw_pattern: "(?m)password\\s*:\\s*'[^$'][^']{2,}'"

# ── Comment path (optional) ───────────────────────────────────────────────────
scan_comments: true             # auto (default) — set false to skip --include-comments for this rule

# ── Migration ─────────────────────────────────────────────────────────────────
migration:
  strategy: auto                # auto (default) | guided | inform_only
  key_moves:                    # Option A — key-level moves (preferred)
    - from: $.old.path
      to:   $.new.path
      default: ""               # inject scalar or block when 'from' is absent
  before: |                     # Option B fallback — shown in report regardless
    # example of old config
  after: |
    # example of new config

# ── Ordering (optional) ───────────────────────────────────────────────────────
order: 0                        # execution priority within a tick (lower = first, default 0)

# ── Documentation ─────────────────────────────────────────────────────────────
description: >-
  Full explanation for the report. Include impact, risk, and remediation steps.
see_also:                       # optional list of related rule IDs
  - P1-01
```

All fields marked *Required* above must be present. All others are optional but
strongly recommended for report quality.

---

## 3. look_for — detection

### 3.1 YAMLPath syntax

Paths start with `$.` and use `.` to separate key segments.

| Expression | Matches |
|---|---|
| `$.exporters.otlp` | Top-level key `otlp` under `exporters` |
| `$.exporters.otlp.endpoint` | Nested key three levels deep |
| `$.exporters.*` | Any direct child of `exporters` |
| `$.exporters.*.sending_queue.blocking` | Any exporter → sending_queue → blocking |
| `$.**.blocking` | Key `blocking` at any depth (recursive descent) |

**Tip:** prefer the most specific path that uniquely identifies the problem.
`$.**.blocking` is broad and could cause false positives in deeply nested configs.

### 3.2 Named component instances

The engine automatically matches **named instances** of components. A path
segment `kafka` also matches any key beginning with `kafka/`:

```
$.receivers.kafka    →  matches: kafka, kafka/consumer, kafka/traces, etc.
$.exporters.otlp     →  matches: otlp, otlp/prod, otlp/eu-west, etc.
```

You do **not** need to add separate selectors for named instances. One selector
covers all instances of a component type.

### 3.3 Match types

| `match:` value | Fires when… | Notes |
|---|---|---|
| `exists` | The path is present (any value) | Most common |
| `absent` | The path is **not** present | Use with `logic: and` — see §3.5 |
| `value` | The resolved node equals `value:` | Exact string comparison |
| `pattern` | The resolved node matches `pattern:` regex | Uses Go `regexp` |

**`absent` example** — detect that a required key is missing:

```yaml
logic: and
look_for:
  - path: $.receivers.kafka
    match: exists
  - path: $.receivers.kafka.client_id
    match: absent
```

**`value` example** — detect a specific boolean:

```yaml
look_for:
  - path: $.exporters.otlp.sending_queue.blocking
    match: value
    value: "true"
```

**`pattern` example** — detect any endpoint using an old port range:

```yaml
look_for:
  - path: $.exporters.otlp.endpoint
    match: pattern
    pattern: ".*:4(317|318)$"
```

### 3.4 Scanning commented-out blocks

When the engine runs with `--include-comments`, a **separate comment path**
(line/regex based, independent of the active-config path) detects commented-out
components and evaluates rules against them. **You no longer need an
`in_comments: true` selector for this** — the comment path **auto-derives**
matches from your rule's existing `look_for` selectors. A rule like:

```yaml
look_for:
  - path: $.receivers.iis
    match: exists
```

will automatically fire on a commented-out `# iis:` block (even one commented at
the child level inside a live `receivers:` section, with prose/separator comments
interleaved). The detector (`DetectCommentedComponents` in
`engine/comment_scanner.go`) finds each commented component, wraps it under its
inferred section, and evaluates your selectors against it.

**Per-rule override — `scan_comments`:**

```yaml
scan_comments: false   # disable comment scanning for this rule (default: auto)
```

Use `scan_comments: false` to silence a rule whose commented match is noisy or
not actionable. `true` or unset = auto-derive.

**Notes & limitations:**
- `match: absent` and `raw_pattern` selectors are **excluded** from comment
  matching (absent is meaningless against a single extracted component; security
  `raw_pattern` scanning in comments is deferred — see the gap tracker in
  `AGENTS.md`).
- The legacy `in_comments: true` flag still parses, but now only affects the
  **active** eval: a selector marked `in_comments: true` is excluded from
  active-config matching (use it when a path should be checked *only* in
  comments). Comment matching itself no longer depends on the flag.
- On apply, comment effects perform two text edits inside the `# …` lines
  (`applyCommentBlock`): **renames** (`# oldKey:` → `# newKey:`), then **annotation
  injection** — each `comment_path`/`comment_text` move is inserted as `#`-prefixed
  lines immediately above the matching commented component, indented to align and
  idempotent on re-apply. This applies for **any** strategy (auto/guided/inform_only),
  so a guided rule still carries its guidance into the commented template without
  changing the active-config classification. Processed findings are listed in
  `OperationalAssessment.md` under "📝 Commented-Out Config Processed".
- Structural `key_moves` with no safe text representation (deletes, child-key
  injections, sequence ops) are **skipped** in comments — re-enabling the template
  surfaces them on the active path.

### 3.4a Raw text scanning with `raw_pattern`

For security rules and cases where the credential or value is buried deep inside
a sequence (e.g. `scrape_configs[*].basic_auth.password`) that YAMLPath cannot
address conveniently, use `raw_pattern` instead of `path`:

```yaml
look_for:
  # raw_pattern matches against the raw file text using a Go regexp (RE2 engine).
  # When set, path and match are ignored for this selector.
  - raw_pattern: "(?m)password\\s*:\\s*'[^$'][^']{2,}'"
```

Combine with `logic: and` and a structural `path` check to scope the rule to a
specific component type (avoids false positives):

```yaml
logic: and
look_for:
  - path: $.receivers.prometheus
    match: exists
  - raw_pattern: "(?m)password\\s*:\\s*'[^$'][^']{2,}'"
```

**Guidelines for `raw_pattern`:**
- Always use `(?m)` (multiline mode) so `^` and `$` match line boundaries.
- **Go uses the RE2 regex engine — lookahead and lookbehind assertions are NOT supported.**
  `(?!...)`, `(?=...)`, `(?<!...)`, `(?<=...)` will all cause the pattern to fail to compile
  and the rule will silently never fire. Use character class negation instead:
  - Instead of `(?!\$)\S+` (exclude `$`-prefixed values), write `[^$\s]\S*`
  - Instead of `(?!['"\$#{])` (exclude quoted/env values), write `[^\s'"$#{]`
- Double-escape backslashes in double-quoted YAML strings: `\\s` → regex `\s`.
- Use YAML single-quoted strings to avoid double-escaping: `'\s*'` stays as-is.
- Security rules in `rules/security/` **must** use `raw_pattern` (and `strategy: inform_only`).
- After writing a new `raw_pattern`, verify it compiles with: `go tool regexp -compile "YOUR_PATTERN"` or add a fixture test case and run `go test ./engine/... -run TestRuleFixtures/YOUR-RULE-ID`.

### 3.5 logic — OR vs AND

```yaml
logic: or    # DEFAULT — fire if ANY selector matches
```

```yaml
logic: and   # fire only if ALL selectors match
```

`logic: and` is essential when combining an `exists` selector with an `absent`
selector. Without it, the first `exists` match would fire the rule even if the
`absent` condition is not met:

**Wrong** — fires for every kafka config because the first selector always matches:

```yaml
logic: or
look_for:
  - path: $.receivers.kafka
    match: exists
  - path: $.receivers.kafka.client_id
    match: absent
```

**Correct** — fires only when kafka exists AND client_id is absent:

```yaml
logic: and
look_for:
  - path: $.receivers.kafka
    match: exists
  - path: $.receivers.kafka.client_id
    match: absent
```

> **Note:** `logic` applies uniformly to all selectors in the rule, including
> comment selectors. If you need separate logic for the comment case, create a
> second rule.

---

## 4. migration — automated fix

### 4.1 strategy

| `strategy:` | Engine behaviour |
|---|---|
| `auto` (default) | Apply `key_moves` if present; fall back to string replacement |
| `guided` | Structural key_moves NOT applied — report the change with guidance for manual action. **Exception:** comment-only `key_moves` (entries with only `comment_path`/`comment_text`, no `from`/`to`/`sequence_path`) ARE applied so upgrade guidance travels with the config file even when manual action is required. |
| `inform_only` | Detection + report only; nothing to change in collector YAML (e.g. dashboard/alert updates). Same comment-only exception as `guided`. |

Use `guided` for changes that require user judgement, like complete pipeline
rewiring (P1-01). Use `inform_only` for changes that are entirely outside the
collector config file (P2-18 — Prometheus resource attribute renames in
dashboards).

Even with `guided` or `inform_only`, the `before` / `after` blocks are still
shown in the report so the user knows what to do manually.

> **Key rule-authoring principle:** Always add a `comment_path` / `comment_text`
> entry to every `guided` and `inform_only` rule. This embeds the upgrade rationale
> directly in the config file — not just in the assessment report. The IT engineer
> who applies the config may never see the assessment, but they will see the comment.
> The config also gets committed to source control, where the comment becomes the
> permanent record of why the change was made.

### 4.2 key_moves (Option A)

`key_moves` performs **key-level structural migration** — it moves, renames, or
injects YAML keys while preserving the user's actual values. This is the
preferred approach because it works regardless of what values the user has.

```yaml
migration:
  strategy: auto
  key_moves:
    - from: $.old.path.to.key
      to:   $.new.path.to.key
```

**Six operation types:**

#### Move / rename (from + to)

Reads the value at `from`, writes it at `to`, deletes `from`.

```yaml
key_moves:
  # Rename key: user's value is preserved wherever it lives
  - from: $.exporters.splunk_hec.batcher.min_size_items
    to:   $.exporters.splunk_hec.sending_queue.batch.min_size_items
```

If `from` is not found and `default` is set, the default is injected at `to`
instead (no error).

#### Delete (from only, to is empty)

Removes the key at `from`.

```yaml
key_moves:
  - from: $.exporters.splunk_hec.batcher          # remove the emptied block
    to:   ""
```

#### Inject scalar default (to + default, no from)

Writes `default` at `to` only if `to` does not already exist.

```yaml
key_moves:
  - to:      $.receivers.kafka.client_id
    default: sarama
```

#### Inject block default (to + multi-line default, no from)

When `default` contains newlines it is parsed as a YAML sub-tree and merged at
`to`, enabling full component definition injection (EG-3).

```yaml
key_moves:
  - to: $.receivers.file_log
    default: |
      include: [/var/log/app/*.log]
      start_at: beginning
```

#### Sequence operations (sequence_path + old_value / new_value)

Operates on scalar items inside YAML sequence (array) nodes. Three modes:

| `old_value` | `new_value` | Effect |
|---|---|---|
| set | set | **replace** — rename `old_value` → `new_value` (preserves `/suffix`) |
| set | empty | **delete** — remove all items equal to `old_value` (or `old_value/suffix`) |
| empty | set | **add** — append `new_value` if not already present |

```yaml
key_moves:
  # replace: rename component reference in all pipelines
  - sequence_path: $.service.pipelines.*.receivers
    old_value: hostmetrics
    new_value: host_metrics

  # delete: remove signalfx from all pipeline receiver arrays
  - sequence_path: $.service.pipelines.*.receivers
    old_value: signalfx
    new_value: ""

  # add: inject file_log into all log pipeline receivers
  - sequence_path: $.service.pipelines.logs.receivers
    old_value: ""
    new_value: file_log
```

#### Comment injection (comment_path + comment_text)

Prepends `comment_text` as a `HeadComment` on the key node at `comment_path`.
Use this to add inline upgrade guidance lines into the YAML output alongside
structural changes (EG-4).

**Important:** Comment injection runs for ALL strategies (`auto`, `guided`,
`inform_only`). For `guided` and `inform_only` rules the structural migration
is not applied automatically, but the comment IS injected. This ensures upgrade
guidance travels with the config file even when manual action is required.

```yaml
key_moves:
  - comment_path: $.receivers.file_log
    comment_text: "# UPGRADE NOTE: replaced fluentforward — update include paths below"
```

**`comment_once: true`** — when the component type may have multiple named
instances (e.g. `prometheus`, `prometheus/internal`, `prometheus/netscaler`),
the comment would normally be injected on every instance. Add `comment_once: true`
to inject only on the first match (the "type-level" notice):

```yaml
key_moves:
  - comment_path: $.receivers.prometheus
    comment_once: true
    comment_text: "# UPGRADE(P2-07): prometheus start time adjustment removed"
```

#### inject_at_each: true — per-instance inject-if-absent

For the `to + default (no from)` inject-if-absent mode, setting `inject_at_each: true`
changes the scope from "check the full path once" to "find every named instance of
the parent path and inject the leaf key into each one where it is absent."

This is the correct pattern for rules like P1-13 where a default changed and you
want to inject the explicit value into every `filter/xxx` and `filter/yyy` instance:

```yaml
key_moves:
  - to:             $.processors.filter.error_mode
    default:        ignore
    inject_at_each: true   # injects into filter, filter/staging, filter/prod, etc.
```

> **Do not use `from == to` for inject-if-absent.** Writing `from: $.x.key` and
> `to: $.x.key` looks like it should be a no-op, but the engine treats it as a
> move (read value → write at same path → delete source), which will **delete
> the key**. Always use the to-only form shown above for inject operations.

#### sequence_map_path — delete a map item from a sequence

Deletes all items from a YAML sequence where a specific key equals a specific value.
Use when a sequence contains mappings (e.g. a `monitors:` list) and you need to remove
entries that match a known type.

```yaml
key_moves:
  - sequence_map_path: $.receivers.smartagent.monitors
    match_key:         type
    match_value:       jmx     # removes every {type: jmx, ...} entry
```

- `sequence_map_path` must resolve to a YAML sequence node whose items are mappings.
- Named-instance matching applies automatically (e.g. `$.receivers.smartagent` also matches `smartagent/windows`).
- Items that do not contain `match_key: match_value` are preserved unchanged.
- If the sequence is empty after deletion, the key is left as an empty sequence (not removed).
- This operation is mutually exclusive with `from`, `to`, `comment_path`, and `sequence_path`.

#### add_to_pipelines_with — wire an injected component into existing pipelines

On an inject-only (`to + default`, no `from`) key_move, set `add_to_pipelines_with` to the name of
an existing component. After injecting the new component, the engine finds every pipeline whose
matching array (receivers/exporters/processors/connectors — derived from the `to` path prefix)
contains the named source component, and appends the new component's name to those same arrays if
not already present.

Use this when a monitor or sub-feature is replaced with a standalone receiver or exporter that must
join the same pipelines as its predecessor:

```yaml
key_moves:
  # Remove the jmx monitor entry from the smartagent monitors list.
  - sequence_map_path: $.receivers.smartagent.monitors
    match_key: type
    match_value: jmx
  # Inject a replacement jmx receiver and wire it into every pipeline
  # that already contains smartagent in its receivers array.
  - to: $.receivers.jmx
    default: |
      jar_path: /opt/opentelemetry-java-contrib-jmx-metrics.jar
      endpoint: service:jmx:rmi:///jndi/rmi://localhost:7199/jmxrmi
      target_system: jvm
    add_to_pipelines_with: smartagent   # adds jmx to every pipeline that has smartagent
```

- Only applies to the inject-only (`to + default`, no `from`) key_move mode.
- Does nothing when the source component is not present in any pipeline array.
- Does nothing when the new component is already in the pipeline array.
- Does NOT remove the source component from pipelines — the source may still have other active monitors.
- The `to` path prefix determines which pipeline array type to scan:
  `$.receivers.*` → `pipelines.*.receivers`, `$.exporters.*` → `pipelines.*.exporters`, etc.

#### wrap_as_sequence — rename a scalar field to a list field

On a `from → to` move, set `wrap_as_sequence: true` to wrap the moved scalar value in a
new single-item flow sequence at the destination path. This is the correct pattern when a
field was renamed from a scalar to a list type.

```yaml
key_moves:
  - from:             $.receivers.kafka.group_rebalance_strategy
    to:               $.receivers.kafka.group_rebalance_strategies
    wrap_as_sequence: true   # value "sticky" becomes [sticky]
```

- Preserves the original value; only the key name and YAML type change.
- Named instances are handled automatically (e.g. `kafka/consumer`, `kafka/prod`).
- The wrapping always produces a flow-style sequence (`[value]`), matching the scalar inline style.
- In comment blocks the key is renamed but the value is not wrapped (the comment text advises the user).

#### Wildcards in paths

Use `*` to match all direct children at a level. The wildcard position is
tracked and substituted into the `to` path, so the same component instance
receives both sides of the move:

```yaml
key_moves:
  # Moves the field for otlp, otlp/prod, otlp/eu, etc. individually
  - from: $.exporters.*.sending_queue.blocking
    to:   $.exporters.*.sending_queue.block_on_overflow
```

Named instance suffixes (`/prod`, `/consumer`) are also propagated automatically
without requiring a wildcard.

#### Notes on yaml.v3 round-trip

When `key_moves` are applied, the engine parses the file into a yaml.v3 node
tree, applies the moves, and re-marshals. yaml.v3 preserves inline and block
comments during round-trip. The output is normalised to 2-space indentation.

### 4.3 before / after (string fallback)

When `key_moves` are absent and `strategy` is `auto`, the engine attempts a
plain string replacement: it looks for the exact `before` text in the raw file
and replaces it with `after`.

```yaml
migration:
  before: |
    connectors:
      routing:
        match_once: true
  after: |
    connectors:
      routing: {}
```

**This approach is fragile** — it fails if the user's config differs in
indentation, comments, or leaf values. Prefer `key_moves` for any change that
involves key renames or structural moves. Use the string fallback only for
exact field removals where the entire block is expected to be identical.

The `before` / `after` blocks are always shown in the Pre-Assessment report
regardless of strategy, giving the user a visual diff of the expected change.

> **Indentation danger zone — `before`/`after` must be siblings of `key_moves`, not children of it:**
>
> `before:` and `after:` sit at **2-space indent** directly under `migration:`,
> at the same level as `strategy:` and `key_moves:`. They are **never** nested inside
> `key_moves`. The trap is that when you add a new `- comment_path:` entry at the
> bottom of an existing `key_moves:` list, a pre-existing `before:` block
> immediately below it looks visually attached — but it must stay at 2 spaces.
>
> ```yaml
> migration:
>   strategy: guided
>   key_moves:              # 2-space indent under migration
>     - comment_path: ...   # 4-space indent — list items
>       comment_text: |
>         # comment text here
>   before: |               # 2-space indent — sibling of key_moves, NOT inside it
>     receivers:
>       my_component: {}
>   after: |                # also 2-space indent
>     ...
> ```
>
> If `before:` accidentally ends up at 4-space indent (inside the sequence),
> the YAML parser will throw `did not find expected '-' indicator` at the
> `key_moves:` line. Run `go test ./engine/...` after any rule edit to catch this.

---

## 5. Common patterns

### Pattern A — Simple key rename

A field was renamed. Detect the old name, move the value to the new name.

```yaml
id: P1-XX
category: p1
introduced: "0.XXX"
title: "otlp Exporter — old_field Renamed to new_field"
look_for:
  - path: $.exporters.*.old_field
    match: exists
  - path: $.exporters.*.old_field
    match: exists
    in_comments: true
migration:
  strategy: auto
  key_moves:
    - from: $.exporters.*.old_field
      to:   $.exporters.*.new_field
  before: |
    exporters:
      otlp:
        old_field: somevalue
  after: |
    exporters:
      otlp:
        new_field: somevalue
description: >-
  The old_field was renamed to new_field in 0.XXX. Collector fails to start if
  old_field is still present.
```

### Pattern B — Field removed entirely

The field no longer exists in the schema. Remove it.

```yaml
look_for:
  - path: $.connectors.routing.match_once
    match: exists
  - path: $.connectors.routing.match_once
    match: exists
    in_comments: true
migration:
  strategy: auto
  key_moves:
    - from: $.connectors.routing.match_once
      to:   ""
  before: |
    connectors:
      routing:
        match_once: true
  after: |
    connectors:
      routing:
        # match_once removed — no replacement
```

### Pattern C — Inject required field when absent

A field that was previously optional (or had a hardcoded default) must now be
explicitly set.

```yaml
logic: and
look_for:
  - path: $.receivers.somecomponent
    match: exists
  - path: $.receivers.somecomponent.required_field
    match: absent
migration:
  strategy: auto
  key_moves:
    - from: $.receivers.somecomponent.required_field
      to:   $.receivers.somecomponent.required_field
      default: legacy_value
```

### Pattern D — Structural restructure (nested → different nesting)

A sub-tree moved to a different parent. Move each leaf, then delete the old
parent block.

```yaml
migration:
  strategy: auto
  key_moves:
    - from: $.exporters.splunk_hec.batcher.min_size_items
      to:   $.exporters.splunk_hec.sending_queue.batch.min_size_items
    - from: $.exporters.splunk_hec.batcher.max_size_items
      to:   $.exporters.splunk_hec.sending_queue.batch.max_size_items
    - from: $.exporters.splunk_hec.batcher    # delete the now-empty block
      to:   ""
```

### Pattern E — Guided rewrite (cannot be automated)

The change requires user judgement. Set `strategy: guided` and explain clearly
in `description` and `after` what the user must do.

```yaml
migration:
  strategy: guided
  before: |
    # Old approach
  after: |
    # New approach — the user must adapt this to their specific pipeline
description: >-
  This change requires manual work because [reason]. See 'after' for the
  target structure. Refer to <link> for full migration guidance.
```

### Pattern F — Inform only (no collector YAML change)

The breaking change is entirely outside the collector config (dashboards, ACLs,
downstream systems). Set `strategy: inform_only`.

```yaml
migration:
  strategy: inform_only
  before: |
    # Old attribute name used in dashboards:
    # net.host.name
  after: |
    # New attribute name:
    # server.address
description: >-
  This change affects dashboards and alert queries, not collector YAML.
  Update all references to net.host.name → server.address.
```

---

## 6. Rule phases and subdirectories

Rules live in a **two-level directory tree** rooted at `rules/`. The engine uses
`LoadRulesTree` which loads `rules/*.yaml` first (the upgrade rules), then
recurses into each immediate subdirectory and stamps those rules with the
directory name as their `phase`.

```
rules/
├── *.yaml            ← phase: upgrade   (versioned breaking-change rules)
├── security/
│   └── sec-p1-NN.yaml ← phase: security  (evergreen credential/hygiene checks)
├── pipeline/         ← phase: pipeline  (reserved for future topology rules)
└── post_assessment/  ← phase: post_assessment (reserved)
```

### Phase values

| Phase | Directory | When rules fire | Typical strategy |
|---|---|---|---|
| `upgrade` | `rules/*.yaml` | When their `introduced` version is within the scan window | `auto`, `guided`, `inform_only` |
| `security` | `rules/security/` | Always (every run, any version) | `inform_only` only |
| `pipeline` | `rules/pipeline/` | Always (reserved) | `inform_only` |
| `post_assessment` | `rules/post_assessment/` | Always (reserved) | `inform_only` |

You can also set `phase:` explicitly inside a rule file to override the
directory-inferred value.

### Security rules (`rules/security/`)

Security rules are **evergreen** — they apply regardless of the upgrade version
range. They detect credential hygiene issues such as hardcoded passwords, API
keys, and bearer tokens.

Key authoring rules for `rules/security/`:

| Field | Required value |
|---|---|
| `introduced` | `"0.0.0"` (always applicable) |
| `phase` | `security` (or rely on directory inference) |
| `strategy` | `inform_only` — security rules never auto-modify config |
| Detection | Use `raw_pattern` (regex against raw file text) |
| Comment | Inject a `comment_path` / `comment_text` to tell the operator what to fix |

**Example security rule:**

```yaml
id: SEC-P1-01
category: p1
introduced: "0.0.0"
phase: security
title: "Prometheus — Hardcoded password in scrape_configs"
logic: and
look_for:
  - path: $.receivers.prometheus
    match: exists
  - raw_pattern: "password\\s*:\\s*'[^$'][^']{2,}'"
migration:
  strategy: inform_only
  key_moves:
    - comment_path: $.receivers.prometheus
      comment_once: true
      comment_text: |
        # [SEC-P1-01 SECURITY] Hardcoded password detected in prometheus scrape_configs.
        # Replace with an environment variable reference: password: ${env:PROM_PASSWORD}
description: >-
  Fires when a single-quoted password value that does not start with $ is found
  inside a prometheus receiver block. Hardcoded credentials are a security risk.
```

---

## 7. Adding a new rule — step by step

1. **Identify the change.** Locate the upstream changelog entry or release note.
   Determine the version it was introduced, the component affected, and whether
   it causes a start failure (`p1`), behaviour difference / operational action (`p2`), or
   is informational / advisory (`p3`).

2. **Choose a file name.** Use the next unused number for the appropriate
   priority prefix (e.g. `p1-27.yaml`).

3. **Write the `look_for` selectors.** Start with a simple `exists` on the most
   specific path that identifies the affected config. Test whether named
   instances need a second selector (they usually don't — the engine handles
   `key/variant` automatically).

4. **Choose `logic`.** Use `logic: and` whenever you combine `exists` with
   `absent`. Leave it unset (defaults to `or`) for all other cases.

5. **Choose a migration strategy.**
   - If the fix is a key rename or structural move → use `key_moves`.
   - If the fix requires user judgement → use `strategy: guided`.
   - If there is nothing to fix in the YAML → use `strategy: inform_only`.
   - Only fall back to `before`/`after` string replacement for exact removals
     that have no varying values.

6. **Always add a `comment_path` key_move** for `guided` and `inform_only` rules
   (and recommended for `auto` rules too). This embeds the upgrade rationale directly
   into the config file — not just the assessment report. The config gets committed to
   source control, so the comment becomes the permanent record of why the change was
   made. Use `comment_once: true` if the component may have multiple named instances.

7. **Write the `before` / `after` blocks.** These appear in the report
   regardless of strategy. Keep them concise — just enough to show the shape of
   the change, not a full working config.

8. **Write a complete `description`.** Include:
   - What breaks (start failure? silent data loss? behaviour change?)
   - Why it breaks
   - Exactly what to do to fix it
   - Any caveats (e.g. ACL changes, downstream dashboard updates)

9. **Add a `see_also` list** if the rule is related to other rules a user might
   need to apply together.

10. **Create the companion fixture.** Add `testdata/rules/{id-lowercase}.test.yaml` with at
   least one positive and one negative case. For `auto` rules, add `apply_contains` checks
   to verify the migration output. See §8 for the full fixture format.

11. **Run the tests.**

```powershell
go test ./engine/... -run "TestRuleFixtures/{YOUR-RULE-ID}"
```

---

## 8. Testing your rule

Every rule must have a companion fixture file at `testdata/rules/<rule-id>.test.yaml`.
The `TestRuleFixtures` function auto-discovers and runs all fixtures — you never need to
edit `engine_test.go` for a new rule.

### Adding a fixture file

Create `testdata/rules/{id-lowercase}.test.yaml`:

```yaml
cases:
  - name: "fires when <component> present"
    config: |
      receivers:
        mycomponent:
          some_field: value
    should_fire: true
    # For auto-strategy rules, also check migration output:
    apply_contains:
      - "new_field: value"
    apply_not_contains:
      - "some_field:"

  - name: "silent when <component> absent"
    config: |
      receivers:
        otlp: {}
    should_fire: false
```

**Minimum required:** one positive case (`should_fire: true`) and one negative case (`should_fire: false`).

**For `auto`-strategy rules:** add `apply_contains` / `apply_not_contains` to verify the migration output is correct.

**For `logic: and` rules:** test both the fully-satisfied condition (all selectors match → fires) and the partially-satisfied condition (only some match → does not fire).

**For `raw_pattern` rules (SEC-xx):** include a case with a literal credential string (fires) and a case with an `${env:...}` reference (does not fire). Remember that Go uses the RE2 engine — always verify patterns compile before committing (a bad pattern silently never fires).

**For deletion rules (`from` + `to: ""`):** add a case that places a **commented-out block** and a **following live component** around the component being deleted, then assert (via `apply_contains`) that both survive while the live component is removed (`apply_not_contains`). yaml.v3 attaches a commented-out block sitting above a live key to that key's comments, so deleting the key used to take the comments with it (the P1-17 collateral-deletion bug). `deleteFromParent` now re-homes those comments to the next surviving sibling — see the regression case in `testdata/rules/P1-17.test.yaml`.

### Running the tests

```powershell
# Run all fixture tests
go test ./engine/... -run TestRuleFixtures

# Run only your new rule's fixture
go test -v ./engine/... -run "TestRuleFixtures/C-XX"
```

### Manual testing (optional — for quick iteration before writing fixture)

You can also test interactively with the CLI:

```powershell
.\fluxus.exe assess --rules-dir rules --output-dir ./out your-config.yaml
```

Check that `./out/PreAssessment.md` lists your rule under the expected component.

---

## 9. Known limitations

| Limitation | Detail |
|---|---|
| **`**` recursive descent in `key_moves`** | `findAll` does not yet support `**` wildcards in `key_moves` paths. Use explicit multi-level paths instead. |
| **YAML indentation normalised on key_moves** | When `key_moves` are applied, the file is round-tripped through yaml.v3 and re-indented to 2 spaces. Original indentation is not preserved. |
| **Multi-file topology changes** | `key_moves` operates on one file at a time. Changes that require coordinating two config files (e.g. gateway + agent) must use `strategy: guided`. Use `ScanCrossFileDependencies` after apply to surface stale cross-file references. |
| **logic applies to all selectors** | There is no per-selector group logic. If you need mixed OR/AND logic, split the rule into two separate rules. |
| **comment scanner parses blocks independently** | The comment scanner strips `#` and re-parses each contiguous commented block as a standalone YAML document. YAML anchors or cross-block references inside comments are not resolved. |
| **comment injection order** | Multiple `comment_path` moves in the same rule prepend in forward order; if the exact comment position matters, use a single comment injection per `key_move`. |
| **deleted-node comments are re-homed, not anchored** | When a `from` + `to: ""` delete removes a live key, `deleteFromParent` preserves any commented-out block that yaml.v3 attached to it by moving that block to the **next surviving sibling key**. This prevents data loss, but a comment that genuinely described *only* the deleted component (rare) will also move to the sibling rather than being removed. Inject a fresh `comment_path` notice if you need precise wording on the surviving component. |
