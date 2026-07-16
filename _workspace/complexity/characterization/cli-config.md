# cli-config characterization

- Base SHA: `c30f6cbbee8e37870e5a7eae168b3f61113cc6f3` (stacked on draft PR #10)
- Ownership: exclusive `complexity-characterization-engineer` lease for `cmd/config.go`, `cmd/appstate.go` characterization tests and this evidence file; no production or Git integration ownership.
- Functions/scores:
  - `cmd/config.go:configInitCmd`: cyclomatic 9, cognitive 26
  - `cmd/config.go:configShowCmd`: cyclomatic 11, cognitive 20
  - `cmd/config.go:addChainsFromDirectory`: cyclomatic 8, cognitive 18
  - `cmd/config.go:addPathsFromDirectory`: cyclomatic 8, cognitive 18
  - `cmd/appstate.go:(*appState).updatePathConfig`: cyclomatic 7, cognitive 11
  - `cmd/appstate.go:(*appState).addPathFromUserInput`: cyclomatic 10, cognitive 9

## Observable contracts

### `configShowCmd`

- Metadata remains `show`, aliases `s`, `list`, `l`, no positional arguments, with both `--json/-j` and `--yaml/-y` registered.
- The physical `<home>/config/config.yaml` must exist even when the runtime config is already present in `appState`.
- A missing home returns `home path does not exist: <home>`; an existing home without the file returns `config does not exist: <path>`.
- YAML is the default and explicit `--yaml` is equivalent. YAML currently ends with two newlines because `yaml.Marshal` supplies one and `Fprintln` adds another. JSON ends with one newline.
- Supplying both formats returns `can't pass both --json and --yaml, must pick one`.
- Errors returned by the output writer are ignored.

### `configInitCmd`

- Metadata remains `init`, alias `i`, no positional arguments, with `--memo` registered.
- It creates a previously absent home, then `config/`, then `config.yaml`, and writes exactly `defaultConfigYAML(memo)` with no stdout/stderr output.
- Repeating initialization returns `config already exists: <path>` without replacing the file.
- It uses one-level `os.Mkdir`, not `MkdirAll`; a missing parent is returned as `*os.PathError` and no home is created.

### `addChainsFromDirectory`

- An unreadable/missing input directory returns the underlying `*os.PathError` before locking or mutation.
- Entries are processed in lexical order. Directories, unreadable files, malformed JSON, invalid provider configs, and duplicate chain IDs are reported to stderr and skipped rather than returned.
- A later valid entry is still added, the chain name is the filename segment before the first dot, success reports the provider chain ID, and the successful subset is persisted.

### `addPathsFromDirectory`

- Directories are skipped and reported; a path name is the filename segment before the first dot.
- Valid paths are validated, added in lexical order, reported, and persisted. Missing configured chains emit validation warnings but do not reject a structurally valid path.
- Unlike the chain importer, file-read, JSON, validation, and add errors are wrapped with the file path and returned immediately.
- If an earlier path was added in memory and a later file fails, `appState.config.Paths` retains that partial mutation, later entries are not processed, and the failed locking operation does not write the partial subset to disk.

### `(*appState).updatePathConfig`

- An empty path name returns `empty path name not allowed` before loading or mutating config.
- A missing named path returns `config does not exist for that path: <name>` from inside the locking operation.
- Each of the four identifiers changes only when its supplied string is non-empty; empty strings preserve the prior field.
- A successful update is visible in memory and persisted to `config.yaml`.

### `(*appState).addPathFromUserInput`

- Prompt order and bytes are source client, source connection, destination client, destination connection.
- Every `readLine` error is returned immediately; the path map remains unchanged.
- Each client/connection identifier is validated immediately after its prompt, before the next prompt.
- A successful structurally valid path is validated against configured chains, added under the requested name, and missing-chain warnings are preserved.
- `Config.AddPath` conflicts and exact conflict messages propagate without mutating the existing path.
- The original recreates a buffered reader on every prompt. A bulk `strings.Reader` can therefore have later lines prefetched and discarded: the characterized bulk input reaches the second prompt and returns `io.EOF`. A one-byte streaming test reader is used where the contract under test requires reaching later prompts.

## Tests added before refactor

- `cmd/config_characterization_test.go`
- 17 top-level tests, 31 cases/subcases covering the six violating functions on the original implementation.
- Tests are deterministic, use isolated temporary homes, do not contact RPC endpoints, do not use package globals, and do not run in parallel.

## Original implementation evidence

The following commands passed against the unmodified production files at the base SHA:

```sh
go test ./cmd -run 'Test(ConfigShowCmd|ConfigInitCmd|AddChainsFromDirectory|AddPathsFromDirectory|UpdatePathConfig|AddPathFromUserInput)' -count=1
go test -race ./cmd -run 'Test(ConfigShowCmd|ConfigInitCmd|AddChainsFromDirectory|AddPathsFromDirectory|UpdatePathConfig|AddPathFromUserInput)' -count=1
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 9 cmd/config_characterization_test.go
go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 9 -test cmd/config_characterization_test.go
```

- Focused non-race result: 31 passed, 0 failed.
- Focused race result: 31 passed, 0 failed.
- Both strict `-over 9` checks emitted no violations for the characterization file.

## Gaps/blockers

- File-lock contention is not stress-tested because the local `flock` result and scheduler timing would make that coverage environment-sensitive; persistence and failure atomicity are covered deterministically.
- Real provider initialization and live RPC validation are outside this structural subwave. The chain importer exercises real Cosmos provider construction without initialization, while path tests intentionally use absent chains to avoid network access.
- Filesystem permission/umask variants and injected flag-access errors are not portable to exercise. Missing-directory, missing-parent, malformed-input, writer-error, validation, conflict, partial-effect, and persistence paths are covered.
- No blocker remains. The production refactor is ready for handoff provided it preserves these tests and performs structural extraction only.
