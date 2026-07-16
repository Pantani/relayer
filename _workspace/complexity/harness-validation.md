# Maximum-10 harness validation

- Auditor role: `complexity-verifier`
- Scope: materialized complexity harness and repository complexity gate only
- Reviewed PR head: `#10 @ c30f6cbbee8e37870e5a7eae168b3f61113cc6f3`
- Source mutations by verifier: none; only this validation artifact was rewritten

## Assertions

| assertion | resultado | evidencia | severidade | acao |
|---|---|---|---|---|
| Active maximum is consistently defined as 10 in the core harness | PASS | `CLAUDE.md`, all four agent definitions, all four domain skills, the campaign ledger, `inventory.sh`, and `check-complexity.sh` define score 10 as allowed, score 11 or `>10` as a violation, terminal `0/0/0`, and global maxima `<=10/<=10`. | - | None. |
| CLI boundary behavior accepts 10 and rejects scores above 10 | PASS | Both scripts invoke the analyzers with `-over 10`; `inventory.sh:69,101-103` independently filters only scores `>10` and cross-checks those counts against the CLI outputs at lines 108-114. Live example `(*appState).addPathFromUserInput` measured `10/9` and was absent from the violation inventory, while `(*appState).updatePathConfig` measured `7/11` and was present. | - | None. |
| Terminal and per-PR requirements remain complete | PASS | The campaign requires touched production files without violations, helpers `<=10/<=10`, focal race, package tests, lint, readonly build, diff-check, independent review, global inventory, terminal `make complexity`, global `0/0/0`, maxima `<=10/<=10`, CI enforcement, open PRs, and a complete ledger. | - | None. |
| Agent and skill structure/frontmatter is valid | PASS | YAML parsing succeeded for exactly four agent files and four domain `SKILL.md` files. Each has non-empty `name` and `description`; all agents specify `model: opus`. Responsibilities and writable scopes remain separated, and Git/GitHub integration remains exclusive to the orchestrator. | - | None. |
| References, triggers, and scenarios remain valid | PASS | Both referenced campaign assets resolve, `.claude/commands` is absent, and trigger coverage contains exactly 10 should-trigger plus 10 near-miss cases. The normal and failure/resume scenarios remain present. `bash -n` passed for both complexity scripts. | - | None. |
| Generated-file exclusions are canonical and no broader | PASS | Both scripts enumerate tracked plus non-ignored untracked `*.go` files and exclude only files whose first 40 lines contain the exact canonical `// Code generated ... DO NOT EDIT.` marker. No filename, directory, allowlist, baseline, or suppression exclusion is present. | - | None. |
| Inventory is pinned, reproducible, and cross-checked | PASS | `inventory.sh` fixes `gocyclo@v0.6.0`, `gocognit@v1.2.1`, and `MAX_ALLOWED=10`; it runs full-score and `-over 10` passes and rejects count mismatches. Two executions returned `63/107/110`, maxima `48/99`. After removing only `Generated:`, both runs and `_workspace/complexity/inventory.md` were byte-identical with SHA-256 `54dad39a1f198230858d9f8141addb5774f4fefb894c1caf0e33d28dd8790ff7`. | - | None. |
| Repository gate enforces the maximum-10 policy | PASS | `scripts/check-complexity.sh` has non-overridable `MAX_ALLOWED=10`, invokes both analyzers with `-over 10` and `-test` for gocognit, and fails when either analyzer reports violations. `make complexity` failed as expected on the current 110-function union and printed that every handwritten function must score at most 10. | - | None. |
| Repository gate uses immutable tool pins | PASS | Revalidation of `scripts/check-complexity.sh:7-8` found literal readonly `GOCYCLO_VERSION="v0.6.0"` and `GOCOGNIT_VERSION="v1.2.1"`; no environment-default or override expression remains. `bash -n` passed, and a fresh `make complexity` exercised those pins and failed only on the expected remaining violations. | - | None. |
| Current instructions contain no maximum-9 wording | PASS | Revalidation of `Makefile:74` found the inclusive description “Enforce cyclomatic and cognitive complexity at most 10”. A consistency scan found no active “below 10”, `-over 9`, `<=9`, maximum-9, or score-10-failure policy. The old `cli-version` review remains acceptable historical evidence because the ledger explicitly identifies approval under both the old maximum 9 and current maximum 10. | - | None. |

## Commands and outcomes

```text
bash -n .claude/skills/relayer-complexity-campaign/scripts/inventory.sh
bash -n scripts/check-complexity.sh
  PASS

YAML frontmatter parse for .claude/agents/*.md and .claude/skills/*/SKILL.md
  PASS; 4 agents, 4 skills, all agents model opus

trigger/reference/filesystem checks
  PASS; 10 should-trigger, 10 near-miss, references resolve, no .claude/commands

bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/relayer-max10-validation-a.md
bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/relayer-max10-validation-b.md
  PASS; both 63/107/110, max 48/99

diff normalized(run-a, run-b)
diff normalized(persisted inventory, run-a)
  PASS; all identical; SHA-256 54dad39a1f198230858d9f8141addb5774f4fefb894c1caf0e33d28dd8790ff7

full pinned scores for cmd/appstate.go plus inventory lookup
  PASS; 10/9 is allowed and omitted, 7/11 is violating and present

make complexity
  EXPECTED FAIL; current union has 110 violations, max 48/99

targeted revalidation after requested corrections
  PASS; Makefile says at most 10; check-complexity.sh has immutable literal pins

consistency scan across CLAUDE.md, .claude, scripts/check-complexity.sh, Makefile, and campaign state
  PASS; no contradictory active threshold instruction; old maximum-9 review is explicitly historical in the ledger
```

## Verdict

The two requested corrections are present and executable: repository help now uses inclusive maximum-10 wording and the gate uses immutable analyzer pins. The focused revalidation and consistency scan found no remaining active contradiction. All previously validated maximum-10 semantics, inventory counts, terminal conditions, exclusions, references, frontmatter, triggers, and gate behavior remain valid.

APPROVED
