# DEVELOPER_NOTES_GITKIT (swarm/atm)

This workspace provides a **pure agentic** Git interaction suite:

- A small **pure-Go** library (`swarm/atm/gitkit`) that safely invokes the **system `git`** binary (no shell interpolation).
- A JSON-driven Go helper program (`swarm/atm/tools/git-tool`) invoked only by a YAML tool.
- A **fully mechanical** Git orchestrator agent (no LLM) plus deterministic Git sub-agents under `swarm/resource/incubator/agents/git`.
- A deterministic **rule-based classifier** Go helper (`swarm/atm/tools/classifier`) invoked by a YAML tool, used to translate freeform user text into structured actions.

No user-facing CLI entry is provided; users should interact via agents/tools.

---

## Components

### 1) Go package: `swarm/atm/gitkit`

All functions invoke `git` via `exec.Command` with an argv list (no shell).

Exports:
- `RunGitExitCode(dir, args...) (stdout, stderr string, exitCode int, err error)`
- `ExecGit(dir, args...) (stdout, stderr string, err error)`
- `Clone(repoURL, destDir string) error`
- `Status(dir string) (stdout, stderr string, err error)`
- `Commit(dir, message string) (stdout, stderr string, err error)`
- `Pull(dir string, args ...string) (stdout, stderr string, err error)`
- `Push(dir string, args ...string) (stdout, stderr string, err error)`
- `CurrentBranch(dir string) (branch, stderr string, err error)`
- `RemoteURL(dir string) (url, stderr string, err error)` (origin)
- `RevParse(dir, rev string) (hash, stderr string, err error)`

New helper APIs for agent needs:
- `ListBranches(dir string) (stdout, stderr string, err error)`
- `ListRemotes(dir string) (stdout, stderr string, err error)`
- `LatestCommit(dir string) (stdout, stderr string, err error)`
- `ShowFileAtRev(dir, rev, path string) (stdout, stderr string, err error)`

Notes:
- `ExecGit` wraps errors for convenience; prefer `RunGitExitCode` if you need exact exit codes.

### 2) Go tool: `swarm/atm/tools/git-tool`

- Reads JSON from stdin.
- Produces JSON to stdout.
- Intended to be invoked by `sh:git-tool`.

Supported actions:
- `status`, `clone`, `commit`, `pull`, `push`, `branch`, `remote-url`, `rev-parse`
- `list-branches`, `list-remotes`, `latest-commit`, `show-file`
- `raw` (requires args containing full argv including `git`)

Input formats:

**Envelope (preferred):**
```json
{
  "id": "req-1",
  "user": "alice",
  "payload": {
    "action": "status",
    "dir": "/path/to/repo"
  }
}
```

**Bare payload (backward compatible):**
```json
{ "action": "status", "dir": "/path/to/repo" }
```

**Raw command array (backward compatible):**
```json
{ "command": ["git", "status"], "dir": "/path/to/repo" }
```

Output:
```json
{
  "id": "req-1",
  "user": "alice",
  "stdout": "...",
  "stderr": "...",
  "exit_code": 0,
  "ok": true,
  "error": ""
}
```

### 3) Go tool: `swarm/atm/tools/classifier`

- Deterministic, rule-based (no external binaries).
- Reads JSON from stdin:

```json
{ "query": "git status in /repo" }
```

Or envelope:
```json
{ "id":"req-1", "user":"alice", "payload": { "query": "latest commit in /repo" } }
```

- Produces JSON:
```json
{ "action":"status", "dir":"/repo", "args":[], "message":"", "confidence":1.0 }
```

Confidence:
- 1.0: clear keyword match and enough arguments
- 0.5: ambiguous / missing required args
- 0.0: unknown

### 4) Agent suite: `swarm/resource/incubator/agents/git`

Main orchestrator:
- `agent:git/git` (fully mechanical)

Deterministic sub-agents:
- `agent:git/status`
- `agent:git/clone`
- `agent:git/commit`
- `agent:git/pull`
- `agent:git/push`
- `agent:git/branch`
- `agent:git/remote-url`
- `agent:git/rev-parse`
- `agent:git/list-branches`
- `agent:git/list-remotes`
- `agent:git/latest-commit`
- `agent:git/show-file`

Tools:
- `sh:git-tool` (defined in `sh_git_tool.yaml`)
- `sh:classifier` (defined in `sh_classifier_tool.yaml`)

Orchestrator behavior:
- If `payload.action` is present: deterministically spawn the matching sub-agent.
- Else if `payload.query` is present:
  - call `sh:classifier` to obtain `{action,dir,args,message,confidence}`
  - if `confidence < 0.7`: return a structured clarification JSON (no LLM)
  - else spawn the matching sub-agent

---

## Build / run

Tools are invoked via `go run` from YAML.

No static binaries are checked in.

---

## Testing

```bash
cd /Users/liqiang/workspace/cloudbox/ai/swarm/atm

go test ./gitkit -v
```

Notes:
- Tests skip if system `git` is missing.
- Tests avoid network access.
