# AI Agent Onboarding — Fluxus

Welcome. This document gets you up to speed on the project so you can continue development without prior context.

---

## What This Project Is

A standalone Go application that automates upgrading Splunk OpenTelemetry Collector YAML configuration files across breaking-change versions (currently v0.120 → v0.157). It replaces a Cursor AI Skill with a proper CLI + web app.

**The problem it solves:** The Splunk OTel Collector releases every ~2 weeks. Each release contains breaking changes that require YAML config edits. Without tooling, upgrading a fleet of collector configs is manual, error-prone, and slow.

---

## Repository Layout

```
Collector upgRAde Process/
├── DESIGN-SUMMARY.md         ← Full architecture doc — READ THIS FIRST
├── AGENTS.md                 ← This file
├── go.mod / go.sum           ← Go module (module: github.com/bryan-splunk/fluxus)
│
├── rules/                    ← upgrade rule store (B-xx, C-xx, I-xx, CA-xx; ~93 files)
│   ├── README.md             ← AUTHORITATIVE rule authoring guide — read before editing rules
│   ├── p1-01.yaml             ← naming: {id-lowercase}.yaml
│   ├── ...
│   └── security/             ← security phase rules (SEC-xx — evergreen, run on every config)
│       ├── README.md         ← security rules authoring guide
│       ├── sec-p1-01.yaml       ← prometheus scrape_config hardcoded password (raw_pattern)
│       ├── sec-p1-02.yaml       ← general unquoted/double-quoted password/api_key
│       └── sec-p1-03.yaml       ← hardcoded bearer token or secret in exporters/headers
│
├── engine/                   ← Core Go library — all business logic lives here
│   ├── types.go              ← All shared types (Rule, Effect, State, KeyMove, Migration, Phase, etc.)
│   ├── loader.go             ← Loads rules: LoadRules (flat dir) + LoadRulesTree (root + subdirs)
│   ├── yamlpath.go           ← YAML tree navigation, node construction, path helpers (yaml.v3 seam)
│   ├── scanner.go            ← Match logic: Scan, matchesSelectors, checkMatch
│   ├── migrator.go           ← ApplyMigration and all key-move / comment-drift repair functions
│   ├── comment_scanner.go    ← Extracts commented YAML blocks for scanning
│   ├── ticker.go             ← Tick-per-version loop (dry-run and apply)
│   ├── conflict.go           ← Detects same-key conflicts across ticks
│   ├── topology.go           ← Cross-file pipeline graph validation
│   ├── reporter.go           ← Render layer: //go:embed vars + RenderPreAssessment/RenderOperationalAssessment
│   ├── reporter_transform.go ← Transform layer: view types + BuildPerFileAssessments + countEffects
│   ├── templates/            ← Report templates (embedded at compile time via //go:embed)
│   │   ├── preassessment.tmpl ← Pre-assessment markdown template
│   │   └── operational.tmpl  ← Operational assessment markdown template
│   └── engine_test.go        ← Unit tests for all engine components
│                                 (including TestRuleFixtures — the data-driven per-rule test)
│
├── cmd/cli/main.go           ← cobra CLI: assess, apply, server commands
├── cmd/server/main.go        ← HTTP server wrapping the engine
├── web/index.html            ← Single-page frontend (Alpine.js + Tailwind CDN)
│
├── testdata/                 ← Test data
│   ├── rules/                ← Per-rule fixture files (data-driven tests)
│   │   ├── p2-01.test.yaml    ← One file per rule, named <rule-id>.test.yaml
│   │   ├── p3-01.test.yaml
│   │   ├── sec-p1-01.test.yaml
│   │   └── ... (96 total — one for every rule in rules/ and rules/security/)
│   ├── agent-sample.yaml
│   └── gateway-sample.yaml
│
└── Skill/                    ← Original AI Skill (reference + rule source of truth)
    └── UPGRADE-KNOWLEDGE.md  ← Human-readable knowledge base all rules were derived from
```

> **Note:** Report templates live in `engine/templates/` (`preassessment.tmpl`, `operational.tmpl`)
> and are embedded into the binary at compile time via `//go:embed`. They are plain text files —
> edit them directly without any Go string-escaping.

---

## Core Concepts (Read Before Touching Engine Code)

### State / Effect Pattern
The engine uses a game-physics-inspired State/Effect pattern:
- **State**: Immutable snapshot of parsed YAML config(s) at a point in time
- **Effect**: A detected rule match — pure data describing what needs to change
- **Tick**: One version step. Evaluates rules, accumulates effects, applies them to produce next State
- Never apply effects immediately — accumulate for the tick, then resolve atomically

### Two-Phase Execution
1. **Dry Run** → reads all files, runs full tick chain, collects all Effects → writes `PreAssessment.md`
2. **Apply Pass** → user approves changes → tick chain runs again, writes all config files (modified + unchanged) to `--output-dir` + `OperationalAssessment.md`

> Original source files are **never modified** by `apply`. All output goes to `--output-dir`.

### Version Scoping
- Ticks only run rules where `rule.Introduced <= targetVersion`
- Ticks beyond `targetVersion` run in read-only mode → effects become "Future Changes" in the report

---

## Rule YAML Schema

See `rules/README.md` for the full authoritative authoring guide. Key fields:

| Field | Notes |
|---|---|
| `id` | Change code (P1-01, P2-18, SEC-P1-01, etc.) |
| `category` | `p1` (breaking) / `p2` (degrading) / `p3` (advisory) |
| `introduced` | First version this rule applies (e.g. `"0.129"`); use `"0.0.0"` for evergreen security rules |
| `phase` | `config` (default) / `security` / `pipeline` — inferred from directory if omitted |
| `logic` | `or` (default — any selector fires) or `and` (all selectors must fire) |
| `order` | Integer; controls execution sequence within a tick (lower = first, default 0) |
| `look_for` | List of `{path, match, value, pattern, in_comments, raw_pattern}` selectors |
| `migration.strategy` | `auto` (default) / `guided` (report only) / `inform_only` (detection only) |
| `migration.key_moves` | Option A key-level migration — moves/renames/injects/deletes keys and sequence items |
| `migration.before/after` | YAML snippets shown in the report |

### `look_for` Selector — New `raw_pattern` Field

When `raw_pattern` is set on a selector, it matches against the **raw file text** using a Go regexp instead of the YAML tree. This is the correct approach for security rules that need to detect credential values inside deep sequence structures (e.g. `scrape_configs[*].basic_auth.password`) that YAMLPath cannot address directly.

```yaml
look_for:
  # Structural check via YAMLPath (uses YAML tree)
  - path: $.receivers.prometheus
    match: exists
  # Content check via raw file text — use logic: and to require both
  # NOTE: Go uses RE2; lookahead/lookbehind ((?!...) etc.) are NOT supported.
  # Use character class negation instead: [^$'] rather than (?!\$).
  - raw_pattern: "(?m)password\\s*:\\s*'[^$'][^']{2,}'"
logic: and
```

- `raw_pattern` selectors participate fully in `logic: "or"` and `logic: "and"` alongside path-based selectors.
- When `raw_pattern` is set, `path` and `match` are **ignored** for that selector.
- Security rules in `rules/security/` always use `raw_pattern` for credential detection.

### YAMLPath Selectors (implemented in engine/scanner.go)

| Pattern | Meaning |
|---|---|
| `$.key` | Top-level key |
| `$.a.b.c` | Nested path |
| `$.a.*` | Any child of `a` |
| `$.**.key` | Recursive: `key` at any depth |
| `$.receivers.kafka` | Named instance: matches `kafka`, `kafka/consumer`, `kafka/prod`, etc. |

### Match Conditions

| `match` | Triggers when |
|---|---|
| `exists` | Path resolves to any value (key is present) |
| `absent` | Path does NOT exist in the document |
| `value` | Path resolves to the specific `value` string |
| `pattern` | Path value matches the regex in `pattern` |

### Migration Strategy

| `strategy` | Behaviour |
|---|---|
| `auto` (default) | Apply `key_moves` if present; fall back to `before`/`after` string replacement |
| `guided` | Report the change with full guidance; do NOT apply structural key_moves. **Exception:** comment-only `key_moves` (those with only `comment_path`/`comment_text`, no `from`/`to`/`sequence_path`) ARE applied so upgrade guidance travels with the config. |
| `inform_only` | Detection + reporting only; nothing to migrate in YAML (e.g. dashboard/alert updates). Same comment-only exception as `guided`. |

### Key Moves — Option A

`key_moves` applies key-level structural migration, preserving user values:

| Operation | Fields | Effect |
|---|---|---|
| Move / rename | `from` + `to` | Value at `from` written to `to`; `from` deleted |
| Delete | `from` only | Key at `from` removed |
| Inject scalar | `to` + `default` (no `from`) | Scalar `default` written at `to` only if absent |
| Inject block | `to` + multi-line `default` | `default` parsed as YAML sub-tree, merged at `to` if absent |
| Inject at each instance | `to` + `default` + `inject_at_each: true` | Injects `default` into every named instance (e.g. `filter/foo`, `filter/bar`) where the leaf is absent — use for components that have multiple named instances |
| Sequence replace | `sequence_path` + `old_value` + `new_value` | Rename items in array (pipeline refs) |
| Sequence delete | `sequence_path` + `old_value`, `new_value: ""` | Remove items from array |
| Sequence add | `sequence_path`, `old_value: ""` + `new_value` | Append item if not present |
| Comment inject | `comment_path` + `comment_text` | Prepend comment on key node; add `comment_once: true` when the component type may have multiple named instances to avoid duplicate comments |

### Rule Order Within a Tick

When multiple rules share the same `introduced` version they run in the same tick.
By default they are sorted by `order` (ascending) using a stable sort — rules with
`order: 0` (the default) retain their alphabetical file-load sequence. Use `order`
to ensure one rule's key-move completes before a dependent rule fires in the same tick:

```yaml
# Run this rule first within its tick
order: 10
```

```yaml
# Run this rule after all order:10 rules in the same tick
order: 20
```

> The cross-tick dependency problem (rules in *different* ticks) is solved automatically
> by the State/Effect pattern — each tick operates on the state produced by the previous
> tick. `order` is only needed for within-tick sequencing.

---

## Key Files to Read When Starting Work

| File | Why |
|---|---|
| `DESIGN-SUMMARY.md` | Full architecture, patterns, all design decisions |
| `rules/README.md` | Authoritative guide for adding/editing rules — read before any rule work |
| `engine/types.go` | All type definitions — understand these before touching other engine files |
| `engine/ticker.go` | Orchestrates the core loop — most important engine file |
| `engine/yamlpath.go` | YAML tree navigation and node construction — understand before touching scanner or migrator |
| `engine/scanner.go` | Match logic (`Scan`, `matchesSelectors`, `checkMatch`) |
| `engine/migrator.go` | `ApplyMigration` and all key-move / comment-drift repair helpers |
| `Skill/UPGRADE-KNOWLEDGE.md` | Source of truth for all breaking change knowledge |

---

## Go Module Info

```
module: github.com/bryan-splunk/fluxus
go version: 1.26.4
```

Main dependencies:
- `gopkg.in/yaml.v3` — YAML parsing (preserves comments, normalizes indentation)
- `github.com/Masterminds/semver/v3` — semver comparison
- `github.com/spf13/cobra` — CLI framework
- `github.com/stretchr/testify` — test assertions

---

## CLI Usage

```bash
# Build CLI binary (from project root)
go build -o fluxus ./cmd/cli

# Run pre-assessment (dry run) — output files written to ./out/
./fluxus assess --rules-dir rules --output-dir ./out agent.yaml gateway.yaml

# Apply approved changes — migrated files written to ./out/ (source files untouched)
./fluxus apply --rules-dir rules --output-dir ./out --select all agent.yaml gateway.yaml

# Include commented-out sections (recommended — detects obsolete fields in comments
# and injects inline upgrade guidance comments from auto rules with comment_path moves)
./fluxus assess --rules-dir rules --output-dir ./out --include-comments agent.yaml gateway.yaml
./fluxus apply  --rules-dir rules --output-dir ./out --select all --include-comments agent.yaml gateway.yaml

# Target a specific version (partial upgrade)
./fluxus assess --rules-dir rules --target-version 0.145 --output-dir ./out agent.yaml

# Start web server
./fluxus server --rules-dir rules --port 8080
```

### Input selection — files, directories, and globs

Both `assess` and `apply` accept exact file paths, a directory, a glob pattern, or any mix:

```bash
# All *.yaml / *.yml files in a directory (non-recursive)
./fluxus assess --rules-dir rules --output-dir ./out /etc/otel/configs/

# Glob pattern — standard * and ? wildcards, quoted to prevent shell expansion
./fluxus assess --rules-dir rules --output-dir ./out "configs/*.yaml"
./fluxus assess --rules-dir rules --output-dir ./out "/etc/otel/*/agent.yaml"

# Mix of exact files, a directory, and a glob — duplicates deduplicated automatically
./fluxus apply --rules-dir rules --output-dir ./out --select all \
    agent.yaml /etc/otel/gateways/ "k8s/*.yaml"
```

> Directories are scanned **non-recursively** for `*.yaml` and `*.yml`. For nested
> subdirectory trees, use a glob pattern like `"configs/*/*.yaml"`.

> `apply` writes **all** input files to `--output-dir` (modified + unmodified). It never
> touches the original source files.

---

## Per-Rule Test Fixtures

All 96 rules have a companion fixture file at `testdata/rules/<rule-id>.test.yaml`.
The `TestRuleFixtures` function in `engine/engine_test.go` auto-discovers and runs them.

### Fixture file format

```yaml
cases:
  - name: "fires when kafka present and client_id absent"
    config: |
      receivers:
        kafka:
          brokers: [kafka:9092]
    should_fire: true
    apply_contains:
      - "client_id: sarama"    # only checked for auto-strategy rules

  - name: "silent when client_id already set"
    config: |
      receivers:
        kafka:
          brokers: [kafka:9092]
          client_id: otel-collector
    should_fire: false
```

Fields:
- `name` — sub-test label (shows in `go test -v` output as `TestRuleFixtures/<RULE-ID>/<name>`)
- `config` — inline YAML string passed to `engine.NewState`
- `should_fire` — whether `Scan()` must return at least one effect for the rule
- `apply_contains` — strings that must appear in `ApplyMigration` output (auto rules only)
- `apply_not_contains` — strings that must NOT appear in output

### Running fixture tests

```powershell
# Run all fixture tests
go test ./engine/... -run TestRuleFixtures

# Run fixtures for a single rule
go test -v ./engine/... -run "TestRuleFixtures/P1-07"

# Run one specific case within a rule
go test -v ./engine/... -run "TestRuleFixtures/P2-01/fires_when_kafka_present"
```

### Rule for maintainers

**When you add a new rule, you must also add its fixture.** The convention is:
1. Create `rules/{id-lowercase}.yaml`
2. Create `testdata/rules/{id-lowercase}.test.yaml` with at minimum:
   - One positive case (`should_fire: true`) matching your `look_for` selector
   - One negative case (`should_fire: false`) with only `receivers: {otlp: {}}` or similar
3. For `auto`-strategy rules: add `apply_contains` / `apply_not_contains` to verify the migration output

---

## Development Quick Reference

All commands assume you are in the project root (`C:\CodeBaseLocal\Fluxus\`).
The project is on **Windows / PowerShell** — use `.\fluxus.exe` not `./fluxus`.

```powershell
# Build CLI binary
go build -o fluxus.exe ./cmd/cli

# Build web server binary
go build -o fluxus-server.exe ./cmd/server

# Run all tests (engine unit tests + all 96 rule fixture tests)
go test ./...

# Run all tests in a specific package
go test ./engine/...

# Run a single named test function
go test -v ./engine/... -run TestLoadRules

# Run only rule fixture tests (all rules)
go test ./engine/... -run TestRuleFixtures

# Run fixture tests for a single rule (verbose)
go test -v ./engine/... -run "TestRuleFixtures/P1-07"

# Run one specific case within a rule fixture
go test -v ./engine/... -run "TestRuleFixtures/P2-01/fires_when_kafka_present"

# Assess configs (dry run — writes PreAssessment.md to ./out/)
.\fluxus.exe assess --rules-dir rules --output-dir ./out agent.yaml

# Apply changes (writes migrated files to ./out/ — source files never touched)
.\fluxus.exe apply --rules-dir rules --output-dir ./out --select all agent.yaml

# Start web server
.\fluxus.exe server --rules-dir rules --port 8080
```

> After **any** code change to the engine or rules, run `go test ./...` before considering work done.
> After adding or editing a rule, also verify `go build ./...` succeeds (catches YAML parse errors at load time).

---

## Coding Conventions

Code readability takes priority over brevity in names. Use full words for every variable, parameter, and function: `ruleCopy` not `rc`, `sourceFile` not `src`, `parsedNode` not `pn`. Avoid initialisms unless the full term is a widely-known acronym in the domain (e.g. `yaml`, `url`, `id`).

---

## Rule ID Convention

All rules follow a two-axis scheme: **priority** (what happens if you do nothing?) and **migration.strategy** (how much work to fix it?).

### ID format

```
<DOMAIN>-<PRIORITY>-<SEQUENCE>
```

Config rules (the default domain) drop the domain prefix — the priority IS the primary sort:

```
P1-01   P2-01   P3-01        ← config / upgrade rules (rules/ root)
SEC-P1-01  SEC-P2-01         ← security rules (rules/security/)
PIPE-P1-01                   ← pipeline rules (rules/pipeline/, future)
PRE-P1-01 / POST-P1-01       ← pre/post assessment (future)
```

### Priority levels

| Code | `category:` value | Meaning |
|------|-------------------|---------|
| `P1` | `p1` | **Breaking** — collector fails or key functionality stops without this change |
| `P2` | `p2` | **Degrading** — collector runs but behaves sub-optimally, silently misconfigures, or loses data |
| `P3` | `p3` | **Advisory** — no functional impact; best practice / future-proofing |

### Sequence numbering

Within each priority level, rules are numbered by existing order (numbered series first, former N-series appended). The `-N` series no longer exists — `migration.strategy: inform_only` already communicates "nothing to auto-apply here."

Current ranges:

| Range | Priority | Count |
|-------|----------|-------|
| P1-01 … P1-20 | P1 — former P1-01…P1-20 + P1-05 | 20 |
| P1-21 … P1-26 | P1 — former P1-21…P1-26 | 6 |
| P1-27 … P1-32 | P1 — v0.153–v0.157 additions | 6 |
| P2-01 … P2-18 | P2 — former P2-01…P2-18 | 18 |
| P2-19 … P2-35 | P2 — former P2-19…P2-35 | 17 |
| P2-36 … P2-43 | P2 — v0.154–v0.157 additions | 8 |
| P3-01 … P3-10 | P3 — former P3-01…P3-10 | 10 |
| P3-11 … P3-14 | P3 — former P3-11…P3-14 | 4 |
| P3-15 … P3-18 | P3 — v0.154–v0.157 additions | 4 |
| SEC-P1-01 … SEC-P1-03 | Security P1 | 3 |

### Go constants

```go
CategoryP1  Category = "p1"   // was CategoryCritical
CategoryP2  Category = "p2"   // was CategoryInvolved
CategoryP3  Category = "p3"   // was CategoryCasual

PhaseConfig   Phase = "config"          // was PhaseUpgrade ("upgrade")
PhaseSecurity Phase = "security"
PhasePipeline Phase = "pipeline"
PhasePost     Phase = "post"            // was PhasePostAssess ("post_assessment")
```

---

## Adding a New Collector Version

1. Read the Splunk OTel Collector release notes
2. For each breaking change, determine priority: **P1** (breaking), **P2** (degrading), **P3** (advisory)
3. Create `rules/<new-id-lowercase>.yaml` (e.g. `p1-27.yaml`) — follow **`rules/README.md`** for the full schema, patterns, and testing instructions
4. Create a companion fixture at `testdata/rules/<new-id-lowercase>.test.yaml` with at least one positive and one negative case; add `apply_contains`/`apply_not_contains` checks for `auto`-strategy rules
5. Reference `Skill/UPGRADE-KNOWLEDGE.md` for conventions on descriptions and migration guidance
6. Run `go test ./...` to verify all tests pass
7. No binary changes needed — rules are loaded at runtime from the `rules/` directory

---

## Current Status

All four phases are complete:
- Phase 1: All ~93 rules converted to `rules/*.yaml` files + 3 security rules in `rules/security/`
- Phase 2: Full `engine/` package implemented, including:
  - YAMLPath evaluation with named instance support (`kafka/consumer` matches `$.receivers.kafka`)
  - `match: absent` fully working
  - `logic: and` for compound selectors
  - `raw_pattern` on `look_for` selectors — matches against raw file text (used by security rules)
  - Option A key-level migration via `key_moves` (preserves user values)
  - Sequence delete/add operations (EG-1/EG-2)
  - Complex YAML block injection via multi-line `default` (EG-3)
  - Inline comment injection via `comment_path` / `comment_text` (EG-4) — runs for ALL strategies
  - `comment_once: true` — injects comment only on first match (prevents duplicates for multi-instance components)
  - `inject_at_each: true` — injects default into every named instance of the parent path where leaf is absent
  - Comment-only key_moves applied even for `guided`/`inform_only` strategies (guidance travels with config)
  - `--include-comments`: a **separate comment path** (line/regex based). `DetectCommentedComponents` (`comment_scanner.go`) finds commented-out components — tolerant of interleaved prose/separators and of child-level comments inside a live section (e.g. `# iis:` under an active `receivers:`) (Gap A). `Scan` then **auto-derives** comment matches from each rule's own `look_for` selectors (no explicit `in_comments:` needed), with a per-rule `scan_comments: false` override (Gap B). On apply, `applyCommentBlock` applies **renames** and injects each rule's `comment_path` **annotation** directly above the matching commented component (idempotent, any strategy), and the processed findings are listed in `OperationalAssessment.md` under "📝 Commented-Out Config Processed" (Gap C). See the Comment-Processing Gap Tracker below.
  - Section-level FootComment drift repair: `fixSectionCommentDrift` promotes sunk FootComments from deep leaf scalars to HeadComments on the following sibling key — fixes BUG-1/BUG-2 where section-separator comments appeared at 12+ space indentation after renames
  - Comment-preserving deletion: `deleteFromParent` re-homes orphaned comments (the deleted key's `HeadComment` + the predecessor value's `FootComment`) onto the next surviving sibling key instead of discarding them. A commented-out block sitting above a live key (e.g. disabled `# windowsperfcounters:` / `# iis:` templates above a live `fluentforward:`) is stored by yaml.v3 in the following key's comments; the earlier "clear to prevent drift" behaviour silently deleted that unrelated config when the live key was removed (BUG-3, hit by P1-17 FluentD removal). Only the deleted node's own `LineComment`/`FootComment` are dropped. Regression fixture: `testdata/rules/p1-17.test.yaml` (case "preserves commented-out template block …")
  - Within-tick rule ordering via `order` field + `sortRulesByOrder` (EG-5)
  - Unused component detection in `ValidateTopology` (EG-6)
  - Cross-file receiver dependency scan via `ScanCrossFileDependencies` (EG-7)
  - `guided` and `inform_only` strategy flags
  - Multi-phase rule loading via `LoadRulesTree` — loads config rules (root) + all subdirectory phases (security, pipeline, etc.)
  - `Phase` field on `Rule` struct — stamped from subdirectory name when loading
  - **Per-rule data-driven test framework** — `TestRuleFixtures` in `engine_test.go` auto-discovers `testdata/rules/*.test.yaml` and runs each case as a named sub-test; 96 fixture files covering all rules with positive, negative, and apply-output checks
- Phase 3: CLI (`cmd/cli/`) implemented with cobra; `--output-dir` on both assess and apply
- Phase 4: Web server (`cmd/server/`) + frontend (`web/index.html`) implemented

The original AI Skill in `Skill/` remains functional as a fallback and regression reference.

---

## Comment-Processing Gap Tracker

The `--include-comments` workflow is being reworked into a **separate comment path**
(line/regex based) that runs independently of the active-config path but shares the
same rules. Track work here so nothing is lost. **Do not delete entries** — mark them
DONE with the commit/PR that closed them.

| Gap | Status | Summary |
|---|---|---|
| **A — prose-tolerant detection** | ✅ DONE | `DetectCommentedComponents` (`engine/comment_scanner.go`) is a line/regex detector that finds commented-out components even when prose/separators are interleaved and when components are commented at the child level inside a live section (e.g. `# iis:` under an active `receivers:`). The old `ExtractCommentedBlocks` re-parsed the whole block as YAML and silently dropped it on any prose. Tests: `engine/comment_scanner_test.go`. |
| **B — rule coverage / auto-derivation** | ✅ DONE | `Scan` now evaluates each rule against detected commented components by auto-deriving from the rule's own `look_for` selectors (no explicit `in_comments:` needed). Per-rule override: `scan_comments: false` disables; `true`/unset = auto. Active-path behaviour is unchanged (in_comments selectors still excluded from the active eval). `raw_pattern` and `absent` selectors are excluded from comment matching (see Gap F). Tests: `engine/comment_scan_test.go`. |
| **C — comment apply + reporting breadth** | ✅ DONE | Comment effects now route to `applyCommentBlock` (`engine/scanner.go`), which applies **renames** then injects each `comment_path`/`comment_text` **annotation** as `#`-prefixed lines immediately above the matching commented component (indented to align), idempotently. This runs for **every** comment effect regardless of strategy (`Apply` no longer skips guided/inform_only comment matches) — guidance travels into the commented template without changing the active-config classification. Comment effects are collected in `ApplyResult.CommentEffects` and rendered in a new **"📝 Commented-Out Config Processed"** section of `OperationalAssessment.md` (plus a summary-row count). Structural moves that cannot be expressed as safe text edits (deletes, child-key injections, sequence ops) are still skipped — re-enabling the template surfaces them on the active path. Tests: `engine/comment_apply_test.go`. Note: the comment-path annotation anchoring here sidesteps Gaps D/E because it is text-anchored to the component, not the active-path HeadComment. |
| **D — blank-line regression (active path)** | ⏳ PENDING | When the active path reserializes, yaml.v3 drops the blank-line separators between commented blocks and glues disabled templates onto the **following key's HeadComment**, producing one dense run (observed: the windows-standard templates jammed under `jaeger:` with the injected P3-05 note on top). Preserve intended blank-line separators between distinct commented blocks on round-trip. |
| **E — inserted comment location (active path)** | ⏳ PENDING | Injected `comment_path` annotations land at the **top of a merged HeadComment blob**, far above their target key (e.g. the P3-05 jaeger note appearing ~50 lines above `jaeger:`), instead of immediately adjacent to the target component. Anchor injected annotations to their component. |
| **F — security / raw_pattern in comments** | ⏳ PENDING (after C) | Decide whether SEC-xx (`raw_pattern`) rules should scan commented blocks. Deferred until C and the blank-line/location issues (D/E) are resolved. |

---

## Important Context

- The repo is at `https://github.com/bryan-splunk/Fluxus.git`
- JetBrains GoLand is the primary IDE for direct coding and debugging; **Cursor** is used alongside it to run the AI agent (which benefits from full workspace access)
- The `Skill/UPGRADE-KNOWLEDGE.md` is the authoritative source for rule content — if a rule file and UPGRADE-KNOWLEDGE.md disagree, UPGRADE-KNOWLEDGE.md wins
- Version range covered: v0.120 → v0.157 (93 config rules total: 32 P1, 43 P2, 18 P3 — plus 3 SEC-P1 security rules)
- Do NOT modify `splunk-otel-upgrade-guide.html` or `.canvas.tsx` as part of app development — those are the standalone reference guide, not part of the app
- Report templates are in `engine/templates/` (`.tmpl` files) and embedded at compile time via `//go:embed` in `engine/reporter.go` — edit the `.tmpl` files directly, no Go string escaping needed
