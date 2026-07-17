# Splunk OpenTelemetry Collector — Upgrade Guide (v0.120.0 → v0.153.0)

A comprehensive utility, AI Skill, and reference guide covering every **breaking change** and **deprecation** for the Splunk OpenTelemetry Collector v0.120.0 and v0.153.0.

The project now provides three ways to consume this knowledge, depending on your workflow:

---

## What's in this repo

| Path                                   | What it is                                                                                                                                                                           |
|----------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `splunk-otel-upgrade-guide.html`       | **Standalone HTML guide** — open in any browser, no build step. Share with your team.                                                                                                |
| `splunk-otel-upgrade-guide.canvas.tsx` | **Cursor IDE canvas** — renders as a live panel inside Cursor while you edit configs.                                                                                                |
| `Skill/`                               | **Cursor Agent Skill (FLUXUS)** — conversational AI skill that walks you through an upgrade interactively. See `Skill/README.md`.                                                    |
| `cmd/cli/`                             | **Go CLI** — `fluxus assess` / `fluxus apply` commands. Scans your YAML config files and generates a Pre-Assessment report and migrated output files.                                    |
| `cmd/server/`                          | **Go web server** — serves a browser-based 3-step wizard UI backed by the same engine as the CLI.                                                                                    |
| `engine/`                              | **Rule engine** — Go package implementing the State/Effect tick model, YAML scanner, key-move migrator, topology validator, and report renderer.                                     |
| `rules/`                               | **Rule files** — declarative YAML rules organised by phase (`rules/*.yaml` for upgrade rules, `rules/security/` for credential hygiene). See `rules/README.md` to add or edit rules. |
| `testdata/`                            | Sample agent and gateway configs used for testing and rule validation.                                                                                                               |
| `web/`                                 | Single-page web UI (Alpine.js + Tailwind CSS) served by the web server.                                                                                                              |
| `DESIGN-SUMMARY.md`                    | Architecture, patterns, and technology decisions for the Go application.                                                                                                             |
| `AGENTS.md`                            | Onboarding guide for AI agents continuing work on this project.                                                                                                                      |
| `UNLICENSE.md`                        | This is free and unencumbered software released into the public domain.                                                                                                                    |

---

## Quick start — Go CLI

```powershell
# Build once
go build -o fluxus.exe ./cmd/cli

# Assess your configs (dry run — no files changed)
.\fluxus.exe assess --rules-dir rules --output-dir ./assessment-output config/agent.yaml config/gateway.yaml

# Review assessment-output/PreAssessment.md, then apply approved changes
.\fluxus.exe apply --rules-dir rules --output-dir ./migrated-output --select all config/agent.yaml config/gateway.yaml
```

Output files are written to `--output-dir`. Original config files are never modified.

### Specifying input files

Both `assess` and `apply` accept exact file paths, directories, glob patterns, or any mix:

```powershell
# Exact files (one or more)
.\fluxus.exe assess agent.yaml gateway.yaml

# All *.yaml / *.yml files in a directory (non-recursive)
.\fluxus.exe assess "C:\OTel\configs\"

# Glob pattern — standard * and ? wildcards
.\fluxus.exe assess "configs\*.yaml"
.\fluxus.exe assess "C:\OTel\*\agent.yaml"

# Mix of all three — duplicates are deduplicated automatically
.\fluxus.exe assess agent.yaml "C:\OTel\gateways\" "k8s\*.yaml"
```

> **PowerShell note:** Always quote glob patterns so PowerShell does not try to expand them itself. The tool does its own expansion via `filepath.Glob`.

### `--include-comments`

Pass `--include-comments` to both `assess` and `apply` to also scan and migrate **commented-out config sections**. This is recommended for most upgrades — the engine will:

- Detect obsolete field names inside YAML comments and surface them in the assessment
- Perform text-based key renames inside comment lines when a rename rule fires on a commented block
- Inject inline upgrade guidance comments (from rules with `comment_path` key moves) into the output files — this works for **all strategies** (`auto`, `guided`, and `inform_only`), so your config file always carries the upgrade rationale even for changes that require manual action

```powershell
.\fluxus.exe assess --rules-dir rules --output-dir ./out --include-comments config/agent.yaml
.\fluxus.exe apply  --rules-dir rules --output-dir ./out --approve all --include-comments config/agent.yaml
```

See `fluxus --help` for the full flag reference.

---

## Quick start — web UI

```powershell
go run ./cmd/server --rules-dir rules --port 8080
```

Open `http://localhost:8080` in a browser. Upload your config files, review detected changes, and download the migrated output — no command line required.

---

## Quick start — HTML guide

Open `splunk-otel-upgrade-guide.html` directly in a browser:

```powershell
# Windows
start splunk-otel-upgrade-guide.html
```

No server, no install, no dependencies. All assets load from CDN.

---

## Quick start — Cursor canvas

1. Open Cursor IDE and navigate to this repo.
2. Open `splunk-otel-upgrade-guide.canvas.tsx`.
3. Cursor renders it as an interactive panel alongside the editor.
4. Use the navigation tabs to jump between topic sections.

---

## Quick start — Cursor AI Skill

In any Cursor Agent chat, invoke the skill by referencing the `Skill/SKILL.md` file.  
The skill runs a Pre-Assessment against your config files, shows you what needs to change, and guides you through applying fixes. See `Skill/README.md` for full details.

---

## What the guide covers

The guide is organised into eight topic sections:

| Section | Content |
|---|---|
| **Overview** | Release-by-release summary table, v0.120 → v0.153 |
| **Component Renames** | ~30 snake_case renames (e.g. `filelog` → `file_log`) |
| **Kafka Migration** | Sarama → Franz-go: removed keys, client ID change, encoding renames, metadata_keys |
| **Processors** | filter/transform error_mode, k8s_attributes semconv gates, cumulativetodelta staleness, tail_sampling |
| **Receivers** | prometheus start time, kubeletstats attributes, docker API upgrade, windowseventlog body restructure |
| **Exporters** | SignalFx URL change, OTLP batcher removal, splunk_hec batcher removal, sapm changes |
| **OTTL Changes** | Type-strict setters (0.150, 0.153), truncate_all utf8_safe |
| **Splunk-Specific** | FluentD removal, Smart Agent monitor removals (0.131–0.149), signalfx receiver removed (0.153) |
| **Upgrade Checklist** | Step-by-step checklist: before, during, and after upgrade |

### Versions covered

```
0.120 · 0.121 · 0.122 · 0.123 · 0.124 · 0.125 · 0.126 · 0.127 · 0.128 · 0.129
0.130 · 0.131 · 0.132 · 0.134 · 0.135 · 0.136 · 0.137 · 0.138 · 0.139 · 0.140
0.141 · 0.142 · 0.143 · 0.144 · 0.145 · 0.146 · 0.147 · 0.148 · 0.149 · 0.150
0.151 · 0.152 · 0.153
```

---

## Highest-impact changes at a glance

Changes most likely to cause silent failures or data loss:

- **0.120** — routing connector `match_once` removed (startup failure)
- **0.121** — signalfx exporter `translation_rules` removed (startup failure)
- **0.123** — `service::telemetry::address` silently ignored → migrate to `readers:` format (**no internal telemetry emitted**); kafka exporter + kafkametrics default `client_id` changed `"sarama"` → `"otel-collector"` (breaks broker ACLs)
- **0.124** — `transform` processor Basic/Advanced Config mixing causes startup failure; `splunkenterprise` metrics all opt-in (silent data loss)
- **0.125** — `k8sattributes` `node_from_env_var` causes hard startup error if env var unset
- **0.126** — prometheus receiver resource attributes renamed (`net.host.name` → `server.address`) — dashboards break silently
- **0.128** — `sqlserver` collection flags renamed (startup failure); telemetry address gate stabilised — fallback flag no longer works
- **0.129** — `sending_queue::blocking` removed → use `block_on_overflow` (startup failure)
- **0.130** — Kafka receiver default client ID changed `"sarama"` → `"otel-collector"` (breaks broker ACLs); OTLP exporter `batcher` block removed
- **0.136** — `kubeletstats` no-op config sections now cause startup failure
- **0.137** — `access_token_passthrough` removed from `signalfx` receiver
- **0.144** — Kafka Sarama fully removed; 20+ Smart Agent monitors removed
- **0.146** — `signalfx` receiver formally deprecated upstream
- **0.148** — Kafka exporter batching requires explicit `metadata_keys`; `windowseventlog` body restructured
- **0.150** — OTTL setters return errors for type mismatches (previously silent no-ops)
- **0.151** — SignalFx default URLs changed to `*.observability.splunkcloud.com`; Windows MSI URL changed
- **0.153** — `filter`/`transform` processors default to `error_mode: ignore`; **`signalfx` receiver permanently removed**

---

## Project structure

```
.
├── cmd/
│   ├── cli/          Go CLI entry point (fluxus assess / apply / server)
│   └── server/       HTTP server entry point
├── engine/           Rule engine Go package
│   ├── types.go      Core data structures (Rule, Effect, State, KeyMove…)
│   ├── loader.go     Load and filter rules from disk
│   ├── scanner.go    YAML path evaluation + Option A key-move migration
│   ├── ticker.go     State/Effect tick orchestration (DryRun / Apply)
│   ├── reporter.go   Markdown report renderer
│   ├── topology.go   Pipeline topology validator
│   ├── conflict.go   Cross-tick conflict detection
│   └── engine_test.go
├── rules/            Declarative YAML rule files organised by phase
│   ├── *.yaml        Versioned breaking-change upgrade rules
│   ├── security/     Evergreen credential / hygiene rules (SEC-xx)
│   └── README.md     Rule authoring guide — read this before adding rules
├── Skill/            Cursor Agent Skill files
├── testdata/
│   ├── rules/        Per-rule fixture files (<rule-id>.test.yaml) — one per rule
│   ├── agent-sample.yaml
│   └── gateway-sample.yaml
├── web/              Web UI (Alpine.js + Tailwind CSS)
├── DESIGN-SUMMARY.md Application architecture and design decisions
├── AGENTS.md  Onboarding guide for AI agents
├── go.mod
└── splunk-otel-upgrade-guide.html   Standalone HTML reference guide
```

---

## Prerequisites & developer setup

Follow these steps on a fresh machine to go from zero to a running build.

### Step 1 — Install Go

`go.mod` requires **Go 1.26.4 or higher**. Download the installer from [go.dev/dl](https://go.dev/dl/).

On Windows the installer adds `go` to your PATH automatically. Verify it worked:

```powershell
go version
```

### Step 2 — Get the code

```powershell
git clone https://github.com/bryan-splunk/Fluxus.git
cd Fluxus
```

### Step 3 — Configure GoLand

1. Open the cloned folder in GoLand (`File → Open`).
2. GoLand detects `go.mod` automatically and shows a **"Configure Go SDK"** banner at the top of the editor.
3. Click it (or go to `File → Settings → Go → GOROOT`) and point it at your Go installation (e.g. `C:\Program Files\Go`).
4. GoLand indexes the module — all dependencies resolve automatically, no further configuration needed.

### Step 4 — Run configurations (GoLand)

The tool has two modes. The web server is the recommended starting point for most users.

**Web server (recommended)**

The web server runs as a subcommand of the CLI binary. Create the run config manually:

1. `Run → Edit Configurations → + → Go Build`
2. Set **Package path** to `github.com/bryan-splunk/Fluxus/cmd/cli`
3. Set **Program arguments** to `server --rules-dir rules --port 8080`
4. Set **Working directory** to the repo root (the `web/` folder containing the UI is resolved relative to this path at runtime)
5. Name it `fluxus-server`
6. Click Run, then open `http://localhost:8080` in a browser

**CLI (assess / apply)**

Open `cmd/cli/main.go` and click the green Run triangle next to `func main()`. GoLand auto-generates a run config with the correct package path. Add program arguments for a quick test run:

```
assess --rules-dir rules --output-dir ./out testdata/agent-sample.yaml
```

**Terminal alternative (no run config needed)**

```powershell
# Start web server
go run ./cmd/cli server --rules-dir rules --port 8080

# Run a CLI assessment
go run ./cmd/cli assess --rules-dir rules --output-dir ./out testdata/agent-sample.yaml
```

### Step 5 — Verify

```powershell
go test ./...
```

All tests should pass. The output will confirm the module path:

```
ok  github.com/bryan-splunk/Fluxus/engine
```

> **Note on IDE config files:** `.idea/` (GoLand) and `.vscode/` are excluded from git by `.gitignore`. This is intentional — IDE run configurations embed machine-specific paths and do not transfer between machines. The terminal commands above are the portable alternative.

---

## Development

```powershell
# Run all tests (engine unit tests + per-rule fixture tests)
go test ./...

# Build the binary
go build -o fluxus.exe ./cmd/cli

# Run tests with verbose output
go test -v ./engine/...

# Run only the per-rule fixture tests
go test -v ./engine/... -run TestRuleFixtures

# Run the fixture test for a single rule (e.g. P2-01)
go test -v ./engine/... -run "TestRuleFixtures/P2-01"
```

To add or update a breaking-change rule, see `rules/README.md` — also add or update the companion fixture at `testdata/rules/<rule-id>.test.yaml`.  
For architecture decisions and design context, see `DESIGN-SUMMARY.md`.  
For onboarding a new AI agent to continue development, see `AGENTS.md`.

---

## Data sources

All breaking change data sourced from official GitHub release pages:

- [signalfx/splunk-otel-collector releases](https://github.com/signalfx/splunk-otel-collector/releases)
- [open-telemetry/opentelemetry-collector-contrib releases](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases)

---

## License

Please read UNLICENSE.md  
Content derived from public OpenTelemetry and Splunk release notes.
