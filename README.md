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

```bash
ai --verbose ...
```
