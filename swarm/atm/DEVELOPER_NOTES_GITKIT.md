# DEVELOPER_NOTES_GITKIT (swarm/atm)

This workspace provides a **pure agentic** Git interaction suite:

- A small **pure-Go** library (`swarm/atm/gitkit`) that previously invoked the
  system `git` binary. It now uses go-git (pure Go) for core operations.
- A JSON-driven Go helper program (`swarm/atm/tools/git-tool`) invoked only by a YAML tool.
- A **fully mechanical** Git orchestrator agent (no LLM) plus deterministic Git sub-agents under `swarm/resource/incubator/agents/git`.
- A deterministic **rule-based classifier** Go helper (`swarm/atm/tools/classifier`) invoked by a YAML tool, used to translate freeform user text into structured actions.

No user-facing CLI entry is provided; users should interact via agents/tools.

---

## Migration notes: go-git

The gitkit package now uses github.com/go-git/go-git/v5 instead of shelling out to the
system `git` binary. This makes the package pure-Go and removes reliance on the
external git program.

Limitations / differences:
- Not all arbitrary `git` subcommands are supported by flyweight RunGitExitCode/ExecGit.
  These functions implement a subset used by the project (init, remote add, add,
  commit, basic clone, status, list branches/remotes, rev-parse, latest-commit,
  show file at rev, pull/push best-effort). Calls to unsupported subcommands will
  return an error explaining the limitation.
- ExecGit/RunGitExitCode only supports a small set of subcommands; attempting arbitrary git subcommands will return exit code 127 or a "not supported" error.
- go-git implements many plumbing operations but lacks exact parity with all
  porcelain behaviors (e.g., complex merge/push configs, credential helpers,
  hooks). For network operations (clone/pull/push) basic behaviour is implemented
  and should work for standard unauthenticated URLs used in tests.
- ExecGit kept for compatibility but intentionally routes to RunGitExitCode which
  only supports implemented subcommands. Downstream callers that relied on
  arbitrary `git` flags should be migrated to use specific helpers.

If you need additional subcommands implemented, extend RunGitExitCode with a
minimal mapping to go-git operations.

---

Rest of the README content remains the same.
