# M1.1b-d focal review

Date: 2026-07-15  
Scope: `interchaintest/`, `Makefile`, `go.work`, `.github/workflows/interchaintest.yml`, and the Docker build path used by the new jobs.  
Mode: read-only review of production/test code; this report is the only file created by the reviewer.

## Findings

### P1 — the two new Docker jobs are deterministically red

Locations:

- `.github/workflows/interchaintest.yml:165-193`
- `Makefile:10-11`, `Makefile:50-52`
- `local.Dockerfile:44`

Both new jobs call `BuildRelayerImage`, whose Dockerfile runs `make install`. The Makefile writes the binary to `$(GOPATH)/bin/rly`, while the final Docker stage copies `/bin/rly`. In the exact build base image (`golang:1.25.9-alpine3.22`), `go env GOPATH` is `/go`, so the produced artifact is `/go/bin/rly`; `/bin/rly` does not exist in the build stage. The final `COPY --from=build-env /bin/rly /bin` therefore fails before either conformance suite starts.

Recommendation: copy `/go/bin/rly`, or make the install target accept an explicit output path and use the same path in both stages. Keep the Docker jobs out of required CI until a complete local Docker image build succeeds.

### P1 — the new localhost job references a non-existent image

Locations:

- `.github/workflows/interchaintest.yml:195-210`
- `interchaintest/localhost_client_test.go:289-297`

The job always selects both `TestLocalhost_` tests. `TestLocalhost_InterchainAccounts` requires `ghcr.io/cosmos/ibc-go-simd:v8.0.0-beta.1`; a registry manifest lookup returns `manifest unknown`. Consequently this required job cannot reach test setup. The other localhost test still uses the old `v8.0.0` simd runtime, so neither case validates an IBC-Go v11.2 application runtime.

Recommendation: remove/gate this job for this batch, or publish/pin an immutable, available simd image built from the intended IBC-Go v11.2 source and update both localhost cases to it.

### P2 — the in-process adapter loses the v11 signing-algorithm contract on key restore

Locations:

- `interchaintest/relayer.go:69-89`
- `interchaintest/relayer.go:116-121`

Interchaintest v11 carries `ChainConfig.SigningAlgorithm`, and its reference rly adapter passes that value to `keys restore --signing-algorithm`. The local adapter neither stores `chainConfig.SigningAlgorithm` in `CosmosProviderConfig` nor passes it to `keys restore`. The CLI then falls back to the empty provider setting; the Cosmos key code defaults unknown/empty algorithms to secp256k1. A chain configured for `sr25519` therefore restores a different key type and address even though the v11 chain config requested sr25519.

Recommendation: propagate `SigningAlgorithm` into `CosmosProviderConfig` and/or append `--signing-algorithm <cfg.SigningAlgorithm>` in `RestoreKey`. Add an argument-level regression test for both secp256k1 and sr25519.

### P2 — the newly added adapter unit tests are not executed by CI

Locations:

- `Makefile:78-88`
- `interchaintest/relayer_args_test.go:11-72`
- `.github/workflows/interchaintest.yml:18-19`, `.github/workflows/interchaintest.yml:42-267`

`interchaintest-contract` compiles tests with `-run '^$'` in both workspace and isolated modes. The root `make test` does not traverse the nested module, and every remaining interchaintest job uses a regex that selects an E2E/scenario test. Therefore the new tests for `AddKey`, client options, link options, path updates, and unsupported methods are never run in CI; a behavior regression can merge as long as it compiles.

Recommendation: add a cheap nested-module unit job or make the contract execute the explicit non-Docker test set (for example, a maintained `Test...Args|TestUnsupported...` regex) before the compile-only checks.

## Checks and evidence

- `GOWORK=off go test -mod=readonly` for the five new adapter unit tests: pass.
- `go test -list '^TestScenario' ./...`: seven root scenarios plus the Stride scenario enumerated successfully; the hardened matrix filter produces real test names rather than package-status lines.
- YAML parse for `.github/workflows/interchaintest.yml`: pass.
- `git diff --check`: pass.
- Registry checks: all newly pinned Gaia heighliner tags inspected (`v7.0.0`, `v7.0.3`, `v8.0.0`, `v14.1.0`) have manifests; `ghcr.io/cosmos/ibc-go-simd:v8.0.0-beta.1` does not.
- The v11 command builders for `CreateClient(s)`, `UpdatePath`, and `LinkPath` match the official interchaintest v11 rly commander flag names/order for the covered options.
- CodeRabbit CLI 0.6.5 was authenticated, but the repository-wide uncommitted review was rejected because the dirty worktree contains 193 files, above the 150-file OSS limit. Its two stored findings were outside this focal integration scope, so this report relies on the narrowed manual review and executable checks above.

## Conclusion

The v11 module/API migration compiles and its covered argument builders are consistent with the upstream commander. It should not enable the three newly added runtime jobs as required CI yet: all three currently have deterministic infrastructure/image failures. Fix the Docker artifact path, replace the missing/stale localhost simd image, propagate signing algorithms on restore, and execute the adapter unit tests in CI before treating M1.1b-d as a green runtime harness.

## Resolution by the integrator

All four findings were addressed after this read-only review:

- `local.Dockerfile` now copies `/go/bin/rly` to `/bin/rly`;
- the localhost job using the unavailable beta image was removed;
- `SigningAlgorithm` is stored in the provider config and passed to
  `keys restore`, with secp256k1/sr25519 argument coverage;
- `make interchaintest-contract` now runs the explicit adapter unit suite with
  `-race`, and the CI module-contract job invokes that target.

The focused Gaia v14.1.0 + Osmosis v22.0.0 in-process relayer setup then passed
all nine subtests. Runtime v11.2 coverage remains a separate open gate.
