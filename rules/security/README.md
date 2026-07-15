# Security Rules (`rules/security/`)

Security rules are **evergreen** checks that run on every config file regardless of
the upgrade target version. They detect security misconfigurations that belong outside
the upgrade rules to keep rules small and focused.

## Phase: `security`

| ID     | Title                                               | Severity |
|--------|-----------------------------------------------------|----------|
| SEC-P1-01 | Hardcoded Credential in prometheus scrape_config    | p1 |
| SEC-P1-02 | Hardcoded password/api_key (unquoted or double-quoted) | p1 |
| SEC-P1-03 | Hardcoded bearer token or secret in exporter/header | p1 |

## When these rules fire

- **SEC-P1-01** — prometheus receiver present AND a single-quoted `password:` value
  is found in the raw file that does not start with `$` (env var reference).
- **SEC-P1-02** — any `password:` or `api_key:` line with an unquoted or double-quoted
  literal value that does not start with `$`.
- **SEC-P1-03** — a `token:`, `secret:`, or `Authorization: Bearer …` line with a long
  literal value that does not start with `$`.

All three rules use `strategy: inform_only` — they inject an inline comment into the
output file but never modify the YAML structure. The comment points operators to the
`${env:VAR_NAME}` syntax for safe credential management.

## Rule authoring principles

1. Security rules **must not** use `strategy: auto`. They are advisory only.
2. Use `raw_pattern` for `look_for` when the credential may be inside a deep
   sequence (e.g. `scrape_configs[*].basic_auth`) that the YAMLPath engine
   cannot address via `$.**.key` easily.
3. **Go uses the RE2 regex engine.** Lookahead and lookbehind assertions are
   **not supported** and will cause the pattern to fail silently (the rule never fires):
   - `(?!...)` negative lookahead — **forbidden**
   - `(?=...)` positive lookahead — **forbidden**
   - `(?<!...)` negative lookbehind — **forbidden**
   - `(?<=...)` positive lookbehind — **forbidden**

   Use character class negation instead:

   | Intent | Lookahead (invalid RE2) | RE2 equivalent |
   |--------|------------------------|----------------|
   | Exclude `$`-prefixed values | `(?!\$)\S+` | `[^$\s]\S*` |
   | Exclude quoted/env values | `(?!['"\$#{])` | `[^\s'"$#{]` |

   Always verify a new pattern compiles by running its fixture test:
   ```powershell
   go test -v ./engine/... -run "TestRuleFixtures/SEC-XX"
   ```
4. Use `logic: and` with a path-based `look_for` entry to scope the rule to a
   specific component type (prevents false positives on other sections).
5. Always set `comment_once: true` on `comment_path` entries to avoid injecting
   the same security notice multiple times when a receiver has named instances.
6. All security rules use `introduced: "0.0.0"` so they run against every config
   regardless of which collector version is being upgraded to.

## Future expansion

The rule phase concept supports additional non-upgrade rule sets:

| Phase            | Directory               | Purpose                                       |
|------------------|-------------------------|-----------------------------------------------|
| `upgrade`        | `rules/` (root)         | Versioned breaking-change migration rules     |
| `security`       | `rules/security/`       | Credential and security hygiene checks        |
| `pipeline`       | `rules/pipeline/`       | Pipeline topology / anti-pattern checks       |
| `post_assessment`| `rules/post_assessment/`| Post-upgrade correctness verification         |

Each phase directory is loaded automatically by `LoadRulesTree()` when `fluxus assess`
or `fluxus apply` is invoked. All phases share the same rule file format and engine.
