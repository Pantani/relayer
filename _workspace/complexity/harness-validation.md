# Harness validation

- Auditor role: `complexity-verifier`
- Scope: materialized complexity harness only
- Base/HEAD: `4c1029d1177f797f11bcd560f14a18305a2e7353`
- Source mutations: none

## Assertions

| assertion | resultado | evidencia | severidade | acao |
|---|---|---|---|---|
| Structure contains the required persistent team and skills | PASS | Exactly four agent files and four domain `SKILL.md` files were enumerated under `.claude/agents/` and `.claude/skills/`: orchestrator, characterization engineer, engineer, and verifier. The campaign also contains `references/trigger-tests.md` and `scripts/inventory.sh`. | - | None. |
| Agent and skill frontmatter is valid | PASS | YAML parsing succeeded for all eight files. Every definition has non-empty `name` and `description`; all four agents also have `model: opus`. | - | None. |
| Harness references resolve from their declaring skill | PASS | `.claude/skills/relayer-complexity-campaign/SKILL.md:70` resolves to the existing bundled `references/trigger-tests.md`; `.claude/skills/relayer-complexity/SKILL.md:18` now anchors both executable references explicitly at `<repo>/scripts/check-complexity.sh` and `<repo>/.claude/skills/relayer-complexity-campaign/scripts/inventory.sh`, and both files exist. | - | None. |
| The four required responsibilities are separated | PASS | The agent definitions independently assign orchestration/integration, original-behavior characterization, structural refactoring, and independent verification; the campaign pipeline at `.claude/skills/relayer-complexity-campaign/SKILL.md:26-32` preserves that producer-reviewer sequence. | - | None. |
| Git/GitHub integration is exclusive to the orchestrator | PASS | `.claude/agents/complexity-orchestrator.md` makes the orchestrator the only agent allowed to branch/integrate/commit/push/open or edit PRs; each specialist definition prohibits Git integration, commit, push, or PR work. The campaign reinforces exclusivity at lines 10 and 34-42. | - | None. |
| Writable-file ownership is disjoint | PASS | `_workspace/complexity/ownership.md:18` restricts the orchestrator to `.claude/**`, `CLAUDE.md`, the three exact core state files, and Git integration. Characterization, production, review, and harness-validation paths at lines 19-22 are mutually disjoint and no longer covered by an orchestrator workspace glob. | - | None. |
| Worktree exclusivity is enforceable from ownership state | PASS | `_workspace/complexity/ownership.md:14` defines an exclusive sequential lease and requires handoff from a non-active prior holder; lines 18-22 show the orchestrator idle, characterization completed, production/review pending, and only harness validation active. Line 24 makes the one-active-editor invariant explicit. Live team inspection confirmed the characterization and analysis agents completed before this final revalidation. | - | None. |
| Initial execution, continuation, failure recovery, partial resume, and base change are covered | PASS | Campaign description includes initiate/continue/recover/partial resume/update/fix/reexecute. Phase 0 at lines 12-17 distinguishes no state, partial request, changed base, and prior gate failure; Phase 2 retries a diff-related failure once; the failure scenario at lines 78-80 blocks the scope, frees ownership when safe, continues disjoint work, and resumes at the failed gate. | - | None. |
| Required per-PR gates and terminal gates are present | PASS | Campaign lines 44-53 require strict local scores, touched production files clean, focal race, package tests, lint, readonly build, diff-check, independent review, global inventory, available GitHub checks, terminal `make complexity`, global `0/0/0`, maxima `<=9/<=9`, tests/race, CI enforcement, open PRs, and complete ledger. Verification skill independently enumerates the same local/global assertions. | - | None. |
| Trigger coverage is materialized and scoped | PASS | The campaign description contains initial and follow-up keywords. `references/trigger-tests.md` has 10 should-trigger cases covering start/continue/partial resume/failure/base change/review and 10 near-miss should-not-trigger cases covering conceptual questions, unrelated review, functional changes, merge, and cleanup. | - | None. |
| Normal and failure/resume scenarios are present | PASS | `.claude/skills/relayer-complexity-campaign/SKILL.md:72-80` contains a complete normal path and a failure path with one producer retry, ledger blocking, ownership release, disjoint continuation, and resume from the failed gate. | - | None. |
| No `.claude/commands` artifact exists | PASS | Read-only filesystem enumeration returned no `.claude/commands` directory or files. | - | None. |
| Persistent campaign state is present and sufficient | PASS | `_workspace/complexity/{inventory,ownership,ledger}.md` exist. Ledger lines 12-16 include branch/base, PR, files, violating functions, before/after local and global scores, characterization/tests, gates, review, blockers, and state. Agent/skill outputs define per-subwave characterization and review paths. | - | None. |
| Inventory script syntax, pins, coverage, and normalized reproducibility are valid | PASS | `bash -n` passed. The script pins `gocyclo@v0.6.0` and `gocognit@v1.2.1`, enumerates tracked and untracked Go files, and excludes only the canonical generated marker in the first 40 lines. Two executions to `/tmp` both returned `79/122/126`, maxima `48/99`; after removing only `Generated:`, both SHA-256 values were `717e1989e55dd6c47fcfe90898f6173d592ec42ba5680ef5ea93d9b4c413f343`, and the normalized output matched the persisted inventory. | - | None. |
| Inventory tool invocations honor the mandatory CLI threshold | PASS | `inventory.sh:29-41` invokes both pinned CLIs with `-over "${MAX_ALLOWED}"`, includes `-test` for gocognit, accepts only expected statuses 0/1, and preserves other failures. Lines 108-114 cross-check CLI violation counts against the full-score inventory. Final execution returned `79/122/126`, maxima `48/99`, and normalized output identical to the persisted snapshot. | - | None. |
| `CLAUDE.md` is a minimal synchronized harness pointer | PASS | `CLAUDE.md:1-12` contains only objective, campaign trigger, conceptual-question exception, and dated change history; it does not duplicate the agent/skill inventory or workflow. | - | None. |

## Safe checks executed

```text
bash -n .claude/skills/relayer-complexity-campaign/scripts/inventory.sh
go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -h
go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -h
bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/relayer-harness-inventory-a.md
bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/relayer-harness-inventory-b.md
bash .claude/skills/relayer-complexity-campaign/scripts/inventory.sh /tmp/relayer-harness-inventory-final.md
diff normalized(run-a, run-b)
diff normalized(persisted inventory, run-a)
diff normalized(persisted inventory, final corrected run)
YAML frontmatter parse for all agent and skill definitions
filesystem enumeration for agents, skills, state, references, and .claude/commands
live team-status inspection for the exclusive worktree lease
```

## Verdict

The harness is structurally complete, references resolve, responsibilities and ownership are disjoint, the shared worktree is protected by an exclusive sequential lease, required flows/gates/triggers/scenarios are present, persistent state is complete, and the inventory script is reproducible while exercising the mandatory pinned `-over 9` CLI gates.

APPROVED
