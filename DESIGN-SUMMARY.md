# FLUXUS App Design Summary
## Fluxus — Standalone Application

---

## Core Pattern: State / Effect with Version Ticks

The engine models a config upgrade as a physics-style simulation (State/Effect pattern from game physics):

- **State** — the current parsed YAML configuration(s). Immutable during evaluation. Pure data.
- **Effect** — a detected rule match. Describes *what* needs to change, not *how*. Pure data. Never applied immediately.
- **Tick** — one version increment (e.g. 0.123 → 0.124). Each tick loads only the rules introduced at that version, evaluates them against the current state, accumulates effects, then applies approved effects to produce the next state.
- **Resolution pass** — effects are applied atomically at the end of each tick. Later ticks see the post-resolution state, which naturally handles version dependencies without an explicit dependency graph.

```
State(0.120)
  → Tick 0.121: evaluate rules → collect effects → apply → State(0.121)
  → Tick 0.122: evaluate rules → collect effects → apply → State(0.122)
  → ...
  → Tick 0.153: evaluate rules → collect effects → apply → State(0.153)
```

---

## Two-Phase Execution

### Phase 1 — Dry Run (Pre-Assessment)
The entire tick chain runs in **read-only mode** against the original config. No files are written. Every effect that fires is collected and tagged with the tick it came from.

```
dry_run(State(current), rules, target_version)
  → []Effect{fired_at, id, category, description, before, after}
  → PreAssessment.md
```

### Phase 2 — Apply Pass
The user reviews the Pre-Assessment and selects which effects to apply (`all`, by category, by ID, or with `include comments`). The tick chain runs again, skipping excluded effects.

```
apply(State(current), rules, target_version, approved_effects)
  → all config files written to --output-dir (modified + unmodified)
  → OperationalAssessment.md
```

Original input files are **never modified**. All output goes to `--output-dir`.

**Future Changes** — ticks beyond the target version run in dry-run-only mode during Phase 1. Effects are collected and presented as informational "Future Changes" in the Pre-Assessment but never applied.

---

## Conflict Detection (Narrowphase)

Between Phase 1 and Phase 2, a conflict detector scans collected effects for cases where two effects at different ticks modify the same config key in potentially incompatible ways. Conflicts are surfaced as warnings in the Pre-Assessment before the user approves. Analogous to collision narrowphase in game physics.

---

## Rule Store Structure

Rules are **declarative YAML files**, one per breaking change, stored in a flat `rules/` directory. Adding support for a new collector version requires only dropping new `.yaml` files into that directory — no code changes.

The `rules/` directory uses a **hierarchical phase structure** to keep rules small and focused:

```
rules/                     ← upgrade rules (B-xx, C-xx, I-xx, CA-xx)
rules/security/            ← security / credential hygiene checks (SEC-xx)
rules/pipeline/            ← pipeline topology checks (future)
rules/post_assessment/     ← post-upgrade correctness checks (future)
```

`LoadRulesTree()` loads all phases automatically. Rules in subdirectories have their
`phase` field stamped from the directory name if not explicitly set.

### Rule Phases

| Phase | Directory | Purpose | Rule IDs |
|---|---|---|---|
| `upgrade` | `rules/` | Versioned breaking-change rules | B-xx, C-xx, I-xx, CA-xx |
| `security` | `rules/security/` | Hardcoded credential and security hygiene checks | SEC-xx |
| `pipeline` | `rules/pipeline/` | Pipeline topology / anti-pattern checks (future) | PL-xx |
| `post_assessment` | `rules/post_assessment/` | Post-upgrade verification (future) | PA-xx |

Security rules use `introduced: "0.0.0"` so they run against every config regardless of target version. They always use `strategy: inform_only` — they inject comment guidance but never modify YAML structure.

```yaml
# rules/c-19.yaml  (upgrade phase — versioned)
id: C-19
category: critical          # critical | involved | casual
introduced: "0.129"
phase: upgrade              # optional — inferred from directory
title: "sending_queue::blocking Field Removed"
logic: or                   # or (default) | and — how look_for selectors combine
look_for:
  - path: "$.exporters.*.sending_queue.blocking"
    match: exists
  - path: "$.exporters.*.sending_queue.blocking"
    match: exists
    in_comments: true       # also scan commented-out blocks
migration:
  strategy: auto            # auto (default) | guided | inform_only
  key_moves:                # Option A — key-level migration (preferred)
    - from: $.exporters.*.sending_queue.blocking
      to:   $.exporters.*.sending_queue.block_on_overflow
  before: |                 # shown in report regardless of strategy
    sending_queue:
      blocking: true
  after: |
    sending_queue:
      block_on_overflow: true
description: >-
  Causes startup failure. The blocking field was deprecated in 0.123
  and removed in 0.129. Use block_on_overflow instead.
see_also:
  - C-09
```

### Rule Fields

| Field | Purpose |
|---|---|
| `id` | Unique change code (C-01, I-18, SEC-01, etc.) |
| `category` | `critical` / `involved` / `casual` |
| `introduced` | Version when this rule first applies; `"0.0.0"` for evergreen security rules |
| `phase` | `upgrade` (default) / `security` / `pipeline` — stamped from subdirectory name if omitted |
| `logic` | How `look_for` selectors combine: `or` (default, any fires) or `and` (all must fire) |
| `order` | Execution priority within a tick — lower runs first (default 0, EG-5) |
| `look_for` | List of YAMLPath selectors + match conditions |
| `look_for[].path` | YAMLPath expression (see syntax table below) |
| `look_for[].match` | `exists` / `absent` / `value` / `pattern` |
| `look_for[].value` | Expected value when `match: value` |
| `look_for[].pattern` | Regex when `match: pattern` |
| `look_for[].in_comments` | Legacy hint to also evaluate this selector against commented-out blocks. With the comment path (Gap B), comment matching is **auto-derived** from a rule's selectors regardless of this flag; it now only affects whether the selector also participates in the **active** eval (in_comments selectors are excluded from active). |
| `scan_comments` (rule-level) | Per-rule override for the comment path: `false` disables comment scanning for this rule; `true`/unset = auto-derive from `look_for`. |
| `look_for[].raw_pattern` | Regex matched against raw file text instead of YAML tree — useful for deep credential patterns in sequences that YAMLPath can't reach easily. Combine with `logic: and` and a path selector to scope to a specific component. **Uses Go's RE2 engine — lookahead/lookbehind (`(?!...)` etc.) are not supported; use character class negation instead.** |
| `migration.strategy` | `auto` — apply key_moves or string replacement; `guided` — report only; `inform_only` — detection only |
| `migration.key_moves` | Option A: list of key-level structural moves that preserve user values |
| `migration.before` | Example of old config (shown in report) |
| `migration.after` | Example of new config (shown in report) |
| `description` | Human-readable explanation and impact |
| `see_also` | Cross-references to related rules |

> **Authoritative rule authoring guide:** `rules/README.md` — read this before adding or editing rules.

### YAMLPath Selector Syntax

| Pattern | Meaning |
|---|---|
| `$.key` | Top-level key |
| `$.parent.child` | Nested key |
| `$.parent.*` | Any direct child of parent |
| `$.**.key` | Recursive: key at any depth in document |
| `$.parent.*.child` | `child` key under any direct child of `parent` |

**Named component instances:** `$.receivers.kafka` automatically matches `kafka`, `kafka/consumer`, `kafka/prod`, etc. The `/variant` suffix convention is handled by the engine — no extra selectors needed.

### Match Conditions

| `match` value | Triggers when |
|---|---|
| `exists` | Path resolves to any value (key is present) |
| `absent` | Path does NOT exist in the document |
| `value` | Path resolves to the specific `value` field string |
| `pattern` | Path value matches the regex in `pattern` field |

> **`absent` + `logic: and`:** use `logic: and` whenever combining `exists` with `absent`. Without it, the first `exists` match fires the rule before the `absent` check runs.

### Migration Strategies

| `strategy` | Behaviour |
|---|---|
| `auto` (default) | Apply `key_moves` if present; fall back to `before`/`after` string replacement |
| `guided` | Report the change with guidance; do NOT apply structural key_moves automatically. **Exception:** comment-only `key_moves` (`comment_path`/`comment_text` with no `from`/`to`/`sequence_path`) ARE applied regardless of strategy so that upgrade guidance travels with the config file even when manual action is required. |
| `inform_only` | Detection and reporting only; nothing to migrate in collector YAML (e.g. dashboard/alert updates). Same exception as `guided` — comment-only `key_moves` are applied. |

### Key Moves (Option A)

`key_moves` performs key-level structural migration — moves, renames, or injects keys while **preserving the user's actual values**. This is more robust than string replacement because it works regardless of what values the user has set.

| Operation | Fields set | Effect |
|---|---|---|
| Move / rename | `from` + `to` | Value at `from` is written to `to`; `from` is deleted |
| Delete | `from` only | Key at `from` is removed |
| Inject scalar | `to` + `default` (no `from`) | Scalar `default` written at `to` only if `to` doesn't exist |
| Inject block | `to` + multi-line `default` (no `from`) | `default` parsed as YAML sub-tree, merged at `to` if absent (EG-3) |
| Inject at each | `to` + `default` + `inject_at_each: true` | Injects `default` into every named instance of the parent path (e.g. all `filter/xxx`) where the leaf is absent |
| Sequence replace | `sequence_path` + `old_value` + `new_value` | Rename items in array nodes (e.g. pipeline references) |
| Sequence delete | `sequence_path` + `old_value`, `new_value: ""` | Remove matching items from array (e.g. deleted component, EG-1) |
| Sequence add | `sequence_path`, `old_value: ""` + `new_value` | Append item if not already present (e.g. new component, EG-2) |
| Comment inject | `comment_path` + `comment_text` | Prepend `comment_text` as `HeadComment` on the key node (EG-4). Runs for ALL strategies including `guided` and `inform_only`. |
| Comment inject (once) | `comment_path` + `comment_text` + `comment_once: true` | Same, but only injects on the first match — prevents duplicates when a component type has multiple named instances (e.g. prometheus, prometheus/internal) |

Wildcards (`*`) in paths are tracked and substituted into the target path so named instances move consistently.

---

## Project Structure

```
Fluxus/
│
├── DESIGN-SUMMARY.md             ← this file
├── AGENTS.md                     ← onboarding guide for AI agents
├── go.mod                        ← Go module definition
├── go.sum                        ← dependency checksums
│
├── rules/                        ← upgrade rule store (B-xx, C-xx, I-xx, CA-xx; ~75 files)
│   ├── README.md                 ← AUTHORITATIVE rule authoring guide
│   ├── c-19.yaml                 ← example: Critical sending_queue::blocking removed
│   ├── i-01.yaml                 ← example: Involved kafka receiver client_id inject
│   ├── ...                       ← ~75 total upgrade rule files
│   └── security/                 ← security phase rules (SEC-xx — evergreen, all versions)
│       ├── README.md             ← security rules authoring guide
│       ├── sec-01.yaml           ← prometheus scrape_config hardcoded password
│       ├── sec-02.yaml           ← general hardcoded password/api_key (unquoted/double-quoted)
│       └── sec-03.yaml           ← hardcoded bearer token or secret in exporters/headers
│
├── engine/                       ← core processing library (Go package)
│   ├── types.go                  ← Rule, Effect, State, KeyMove, Migration types
│   ├── loader.go                 ← loads + version-filters rules from rules/ dir
│   ├── yamlpath.go               ← YAML tree navigation, node construction, path helpers
│   ├── scanner.go                ← match logic: Scan, matchesSelectors, checkMatch
│   ├── migrator.go               ← ApplyMigration and all key-move / comment-drift repair helpers
│   ├── comment_scanner.go        ← extracts + re-parses commented YAML blocks
│   ├── ticker.go                 ← orchestrates the tick-per-version loop
│   ├── conflict.go               ← narrowphase: detects same-key conflicts across ticks
│   ├── topology.go               ← cross-file pipeline graph + validation
│   ├── reporter.go               ← render layer: //go:embed vars + Render* functions
│   ├── reporter_transform.go     ← transform layer: view types + BuildPerFileAssessments
│   ├── templates/                ← report templates (embedded at compile time via //go:embed)
│   │   ├── preassessment.tmpl    ← pre-assessment markdown template
│   │   └── operational.tmpl      ← operational assessment markdown template
│   └── engine_test.go            ← unit tests for all engine components
│                                      (TestRuleFixtures: data-driven per-rule fixture tests)
│
├── cmd/
│   ├── cli/
│   │   └── main.go               ← cobra CLI (assess, apply, server commands)
│   └── server/
│       └── main.go               ← HTTP server wrapping the engine
│
├── web/                          ← static frontend (Alpine.js + Tailwind CDN)
│   └── index.html                ← single-page upload → assess → apply UI
│
├── testdata/                     ← test data
│   ├── rules/                    ← per-rule fixture files (data-driven tests)
│   │   └── <rule-id>.test.yaml   ← 78 files, one per rule
│   ├── agent-sample.yaml
│   └── gateway-sample.yaml
│
├── Skill/                        ← existing AI skill (source of truth for rule content)
│   ├── SKILL.md
│   ├── UPGRADE-KNOWLEDGE.md      ← human-readable knowledge base → converted to rules/
│   ├── TEMPLATES.md
│   └── DATA-PATH-KNOWLEDGE.md
│
├── splunk-otel-upgrade-guide.html    ← static HTML reference guide (do not modify for app work)
├── splunk-otel-upgrade-guide.canvas.tsx ← Cursor canvas reference guide (do not modify for app work)
└── README.md                     ← project overview
```

> **Note:** Report templates live in `engine/templates/` (`preassessment.tmpl`, `operational.tmpl`)
> and are embedded into the binary at compile time via `//go:embed` in `engine/reporter.go`.
> Edit the `.tmpl` files directly — no Go string escaping required.

---

## Technology Stack

### Primary Language: Go

| Reason | Detail |
|---|---|
| Single binary | One executable, no runtime required for end users |
| Native YAML | `gopkg.in/yaml.v3` is mature and well-supported |
| Concurrency | Parallel dry-run ticks via goroutines |
| Cross-platform | Windows / Linux / Mac from one build |
| Team familiarity | Go developers available |

### Dependencies

| Purpose | Package |
|---|---|
| YAML parsing | `gopkg.in/yaml.v3` |
| Semver comparison | `github.com/Masterminds/semver/v3` |
| CLI framework | `github.com/spf13/cobra` |
| Test assertions | `github.com/stretchr/testify` |
| Standard library | `text/template`, `path/filepath`, `net/http` |

> YAMLPath evaluation and key-move migration are implemented directly in `engine/scanner.go` using a recursive tree walker — no additional library needed.

### Web Layer

- **Backend**: Go `net/http` standard library — thin HTTP wrapper around the engine
- **Frontend**: Static HTML + Alpine.js + Tailwind CDN (consistent with existing HTML guide)
- **No framework**: Single `web/index.html` file; no build step required

---

## CLI Interface

```
fluxus assess [config-files...] [flags]
  --target-version  string   Target collector version (default: latest — all rules)
  --include-comments         Also scan commented-out config sections via the line/regex
                             comment path; rules auto-match commented components from their
                             own selectors (Gap B) and are listed under PreAssessment's
                             "Changes Required (Commented-Out Config)" section
  --rules-dir       string   Path to rules directory (default: ./rules)
  --output-dir      string   Directory for PreAssessment.md and report files (default: .)

fluxus apply [config-files...] [flags]
  --target-version  string   Target collector version (default: latest — all rules)
  --include-comments         Process commented-out config: apply key renames inside comment
                             lines and inject each rule's comment_path annotation immediately
                             above the matching commented component (any strategy, idempotent).
                             Processed findings are listed under OperationalAssessment's
                             "📝 Commented-Out Config Processed" section (Gap C)
  --rules-dir       string   Path to rules directory (default: ./rules)
  --select          string   Changes to apply: "all", "critical", "involved", "casual",
                             or comma-separated rule IDs (default: all)
  --output-dir      string   Directory for migrated config files and report (default: .)

fluxus server [flags]
  --port            int      HTTP port (default: 8080)
  --rules-dir       string   Path to rules directory (default: ./rules)
```

> **apply never modifies original files.** All input configs are copied to `--output-dir`
> with changes applied. The source files are left untouched.

---

## Key Design Properties

| Property | How it's achieved                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
|---|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| No code for new versions | Drop `.yaml` file in `rules/`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Version dependency safety | Tick-per-version — later rules see post-migration state                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Within-tick rule ordering | `order` field on rules; `sortRulesByOrder` sorts within each tick before evaluation                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Non-destructive by default | Originals never modified; all output to `--output-dir`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Key-level migration (Option A) | `key_moves` in rules move/rename/inject/delete keys and sequence items, preserving user values                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Complex block injection | Multi-line `default` in `key_moves` parsed as YAML sub-tree (EG-3)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Inline comment injection | `comment_path` + `comment_text` in `key_moves` adds HeadComments; runs for all strategies including `guided`/`inform_only` (EG-4)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Comment deduplication | `comment_once: true` on a `comment_path` move injects only on the first matched instance — prevents duplicate comments for multi-instance components                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Per-instance inject | `inject_at_each: true` injects `default` into every named instance of the parent path where the leaf is absent — correct pattern for global default changes like `error_mode`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Sequence item management | `sequence_path` + `old_value`/`new_value` handles replace, delete, and add for pipeline array entries (EG-1/2)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Named component instance support | `kafka/consumer` matches `$.receivers.kafka` selectors automatically                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Commented-out config scanning (comment path) | `--include-comments` runs a **separate, line/regex comment path**. `DetectCommentedComponents` (`comment_scanner.go`) finds commented-out components — tolerant of interleaved prose/separators and of components commented at the child level inside a live section (`# iis:` under an active `receivers:`), which the old `ExtractCommentedBlocks` (whole-block YAML reparse) could not handle (Gap A). `Scan` wraps each detected component under its inferred section and evaluates the rule's own selectors against it, **auto-deriving** comment matches without an explicit `in_comments:` (Gap B). Per-rule `scan_comments: false` opts out. See the Comment-Processing Gap Tracker in `AGENTS.md`.                                                                            |
| Commented-out config migration | When `IsComment=true`, `applyCommentBlock` (`scanner.go`) applies text-based key renames inside comment lines (regex substitution on `#` lines) and then injects each `comment_path` annotation as `#`-prefixed lines immediately above the matching commented component, indented to align and idempotent (Gap C). This runs for every comment effect regardless of strategy (`Apply` no longer skips guided/inform_only comment matches), so guidance travels into the template without reclassifying active config. Structural moves with no safe text representation (deletes, child-key injections, sequence ops) are skipped — re-enabling the template surfaces them on the active path. Findings are collected in `ApplyResult.CommentEffects` and reported in `OperationalAssessment.md`. |
| Comment drift prevention | `fixSectionCommentDrift` promotes sunk FootComments from deep leaf scalars to HeadComments of the following sibling key in section mappings — prevents section-separator comments from appearing at 12+ space indentation after renames (BUG-1/BUG-2)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Comment-preserving deletion | `deleteFromParent` re-homes orphaned comments (the deleted key's `HeadComment` + the predecessor value's `FootComment`) onto the next surviving sibling key (or the predecessor's `FootComment` when deleting the last pair) instead of discarding them. yaml.v3 stores a commented-out block sitting above a live key inside that key's comments, so the old "clear to prevent drift" behaviour silently deleted unrelated commented-out config when the live key was removed — e.g. C-16 (FluentD removal) erased `# windowsperfcounters:`/`# iis:`/`# sqlserver:` template receivers that sat above a live `fluentforward:` (BUG-3). Only the deleted node's own `LineComment`/`FootComment` are dropped.                                                                                    |
| Partial upgrades | `--target-version` stops apply ticks; future ticks are read-only                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Guided and inform-only changes | `migration.strategy: guided/inform_only` flags changes requiring manual action while still injecting comment-only `key_moves` automatically                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Cross-file topology | `topology.go` builds pipeline graph across all provided configs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| Unused component detection | `ValidateTopology` surfaces declared-but-not-referenced components (EG-6)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Cross-file receiver dependency scan | `ScanCrossFileDependencies` detects exporters pointing to removed/absent receivers (EG-7)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Conflict awareness | `conflict.go` flags same-key conflicts across ticks before apply                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Security checks | `rules/security/` phase holds evergreen `inform_only` rules (SEC-01..03) that detect hardcoded credentials using `raw_pattern` text scanning; loaded by `LoadRulesTree()` alongside upgrade rules                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Rule phase separation | Rules are loaded from `rules/` (upgrade) and all subdirectories (`rules/security/`, etc.); `Phase` field on `Rule` stamped from directory name; keeps security/pipeline checks separate from version-specific upgrade rules                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |

---

## Migration Path: AI Skill → Standalone App

| Phase | Work | Status |
|---|---|---|
| 1 | Convert `UPGRADE-KNOWLEDGE.md` entries to `rules/*.yaml` files | ✅ Complete |
| 2 | Implement `engine/` package (types, scanner, ticker, reporter, topology, conflict) | ✅ Complete |~~~~
| 3 | Build `cmd/cli/` with cobra | ✅ Complete |
| 4 | Build `cmd/server/` + `web/` frontend | ✅ Complete |

---

## Testing

### Test layers

| Layer | Where | What it checks |
|---|---|---|
| `TestLoadRules` | `engine_test.go` | Every rule file parses without error; required fields present |
| `TestFilterByVersion` | `engine_test.go` | Semver filter returns correct applicable/future split |
| `TestEvalPath_*` | `engine_test.go` | YAMLPath `exists`/`absent` detection on synthetic configs |
| `TestMatchAbsent_*` | `engine_test.go` | `logic: and` + `match: absent` fires/silences correctly |
| `TestNamedComponentInstance_*` | `engine_test.go` | `kafka/consumer` matched by `$.receivers.kafka` |
| `TestApplyMigration_*` | `engine_test.go` | Option A key-move rename, inject-default, guided/inform_only no-change |
| `TestConflictDetection` | `engine_test.go` | Same-path effects across ticks detected as conflicts |
| `TestTopologyValidation_*` | `engine_test.go` | Empty-exporter pipeline flagged |
| **`TestRuleFixtures`** | `engine_test.go` | **Data-driven — one sub-test per rule, see below** |

### Per-rule fixtures (`TestRuleFixtures`)

`TestRuleFixtures` auto-discovers `testdata/rules/*.test.yaml` (78 files, one per rule) and runs each case as `t.Run(ruleID / caseName)`.

Each fixture specifies:
- `config` — minimal inline YAML that should (or should not) trigger the rule
- `should_fire` — whether `Scan()` must return at least one effect
- `apply_contains` / `apply_not_contains` — strings checked in `ApplyMigration` output (auto-strategy rules only)

The tick is derived from the rule's own `introduced` version so the rule is always within the scan window.

```powershell
# Run all engine tests
go test ./engine/...

# Run only per-rule fixture tests, verbose
go test -v ./engine/... -run TestRuleFixtures

# Single rule
go test -v ./engine/... -run "TestRuleFixtures/C-06"
```

---

## Adding Support for a New Collector Version

When a new Splunk OTel Collector version ships:

1. Read the release notes for breaking changes and deprecations
2. For each change, create a new file in `rules/` — see **`rules/README.md`** for the full authoring guide including all fields, patterns, and testing instructions
3. Create a companion fixture at `testdata/rules/{id-lowercase}.test.yaml` — include at least one positive (`should_fire: true`) and one negative (`should_fire: false`) case; add `apply_contains`/`apply_not_contains` checks for `auto`-strategy rules
4. Reference `Skill/UPGRADE-KNOWLEDGE.md` for conventions on IDs, categories, and descriptions
5. Run `go test ./...` to verify all tests pass
6. Commit and push — no binary recompilation needed for rule-only changes
