# AI Command Line Tool

## Build

```bash
git clone https://github.com/qiangli/ai.git
cd ai
make build
make install
```

## Run

```bash
ai [OPTIONS] COMMAND [message...]

ai @ask "What is the capital of France?"
git diff origin main|ai @ask generate commit message for git

ai / What tools could I use to search for a pattern in files

ai --help
```

## Test

No API calls will be made in `dry-run` mode.

```bash
ai --dry-run --dry-run-content "fake data" ...
```

Default system prompts can be replaced for testing and evaluation.

```bash
ai --role "system" --role-content "custom prompt" ...
```

## Debug

```json
//https://github.com/jfcg/sorty/issues/6
// go test -c -o bin/test  ./internal/db
{
    "name": "TestGetByVector",
    "type": "go",
    "request": "launch",
    "mode": "exec",
    "program": "./bin/test",
    "args": ["-test.run", "^TestGetByVector$"],
},
```

## Security and Privacy

Certain system information is shared with the LLM through function and tool calls. This enables the AI to provide responses that are most relevant to your system.
In addition to the `man` page, `help` output, and the results of executing `command` or `which`, command names collected from your PATH and the names of environment variables in the current shell may potentially be sent, depending on your queries. You can use `ai list` and `ai info` to inspect the details.

If this is a concern for you, consider updating your PATH or unsetting some of your environment variables before using `ai`.
