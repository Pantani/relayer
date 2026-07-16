# cli-version characterization

- Base SHA: `4c1029d1177f797f11bcd560f14a18305a2e7353` (`origin/main@4c1029d`)
- Ownership: `cmd/version_test.go` and `_workspace/complexity/characterization/cli-version.md` only; no production or Git integration changes.
- Functions/scores: `cmd/version.go:28 getVersionCmd` — cyclomatic `7`, cognitive `16` (campaign inventory).

## Observable contracts

### Cobra API and arguments

- Command metadata is exact: `Use: "version"`, alias `v`, short description `Print the relayer version info`, and example `$ rly version --json\n$ rly v`.
- The alias is executable through a parent command and Cobra reports `CalledAs() == "v"`.
- The command accepts no positional arguments. An extra argument returns `unknown command "unexpected" for "rly version"` and `withUsage` changes both the command and root `SilenceUsage` to `false`, so Cobra prints usage.
- The only local output-format flag is the boolean `--json` / `-j`, default `false`, `NoOptDefVal` `true`, usage `returns the response in json format`; it remains bound to the command's `appState.viper`.
- YAML is selected by absence of `--json`; there is no `--yaml` flag. Passing `--yaml` returns `unknown flag: --yaml` without producing a version payload.

### Output and build metadata

- The payload fields and serialized order are `version`, `commit`, `cosmos-sdk`, `go` in both JSON and YAML.
- Default YAML is emitted by `yaml.v3`; because `yaml.Marshal` already terminates its bytes with `\n` and the command writes through `fmt.Fprintln`, observable YAML stdout ends with two newlines.
- JSON is compact and emitted with one trailing newline.
- `version` comes from the package-level `Version` build symbol.
- `commit` comes from `Commit`. Only the literal `Dirty == "0"` is clean. Empty string, `"1"`, `"false"`, and every other value append ` (dirty)`.
- `cosmos-sdk` is the first `debug.ReadBuildInfo().Deps` entry whose path is `github.com/cosmos/cosmos-sdk`; its runtime version is asserted dynamically. If build info is unavailable or the dependency is absent, the observable fallback is `(unable to determine)`.
- `go` is exactly `<runtime.Version()> <runtime.GOOS>/<runtime.GOARCH>` and is asserted dynamically.
- Serialization/GetBool errors are returned. The four string fields cannot induce a marshal failure in the current shape. Errors returned by `cmd.OutOrStdout().Write` are ignored; the command still returns `nil`.
- The command emits no logs, metrics, files, network calls, caches, retries, goroutines, or other external state.

## Tests added before refactor

- `TestGetVersionCmdMetadata`
- `TestGetVersionCmdSerializesVersionInfo`
- `TestGetVersionCmdCommitDirtyMarker`
- `TestGetVersionCmdAliasExecutes`
- `TestGetVersionCmdRejectsArgumentsWithUsage`
- `TestGetVersionCmdRejectsYAMLFlag`
- `TestGetVersionCmdIgnoresOutputWriterErrors`

The tests use fresh Cobra/Viper instances and synthetic build symbols restored with `t.Cleanup`. They intentionally do not use `t.Parallel`, because `Version`, `Commit`, and `Dirty` are package globals. Runtime Go and Cosmos SDK values are derived from the running test binary, preserving exact contracts without hard-coding the local toolchain or dependency version.

## Original implementation evidence

Executed in `/Users/pantani/.codex/worktrees/complexity-cli-version/relayer` before any production refactor:

```text
$ rtk go test ./cmd -run '^TestGetVersionCmd' -count=1
Go test: 14 passed in 1 packages

$ rtk go test -race ./cmd -run '^TestGetVersionCmd' -count=1
Go test: 14 passed in 1 packages
```

Both commands exited `0` against the original `cmd/version.go` at the base SHA.

## Gaps/blockers

- No blocker: the focal characterization is green on the original implementation, with and without the race detector.
- `debug.ReadBuildInfo() == false`, absence of the Cosmos SDK dependency, `Flags().GetBool` failure, and marshal failure do not have safe injectable seams in `getVersionCmd`; their exact branches are documented from the original code but are not independently forced by tests.
- No global repository gates were run in this characterization subwave; validation is deliberately limited to the focal package/test selection requested by the orchestrator.
