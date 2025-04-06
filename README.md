# AI Command Line Tool

As a software engineer, I need two types of tools for my daily tasks: one for working inside a file and the other outside.

This AI tool assists you with all the tasks beyond file editing on your system complementing Continue, Cline and the like.

## Build

```bash
git clone https://github.com/qiangli/ai.git
cd ai
make build
make install
```

## Run

```bash
ai [OPTIONS] AGENT [message...]

ai @ask "What is the capital of France?"
git diff origin main|ai @ask generate commit message for git

ai / What tools could I use to search for a pattern in files

ai --help

#
tsh ssh --tty user@host ai -i
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

## Usage

```bash
$ ai
```

```text
AI Command Line Tool

Usage:
  ai [OPTIONS] [AGENT] message...

Examples:

ai what is fish
ai / what is fish
ai @ask what is fish


Use ai help [info|agents|commands|tools] for more details.
```

```bash
$ ai help
```

```text
AI Command Line Tool

Usage:
  ai [OPTIONS] AGENT message...

There are multiple ways to interact with the AI tool.

+ Command line input:

  ai AGENT what is fish?

+ Read from standard input:

  ai AGENT -
Ctrl+D to send, Ctrl+C to cancel.

+ Here document:

  ai AGENT <<eof
what is the weather today?
eof

+ Piping input:

  echo "What is Unix?" | ai AGENT
  git diff origin main | ai AGENT [message...]
  curl -sL https://en.wikipedia.org/wiki/Artificial_intelligence | head -100 | ai AGENT [message...]

+ File redirection:

  ai AGENT [message...] < file.txt

+ Reading from system clipboard:

  ai AGENT [message...] =
Use system copy (Ctrl+C on Unix) to send selected contents.
Ctrl+C to cancel.

+ Composing with text editor:

  export AI_EDITOR=nano # default: vi
  ai AGENT


Agent:
  /[command]       [message...] Get help with system command and shell scripting tasks
  @[agent/command] [message...] Engage specialist agents for assistance with complex tasks

Miscellaneous:
  setup                   Setup the AI configuration

Options:
      --api-key string          LLM API key
      --base-url string         LLM Base URL (default "https://api.openai.com/v1/")
      --config string           config file (default "/Users/qiang.li/.ai/config.yaml")
      --doc-template string     Document template file
      --editor string           Specify editor to use (default "vi")
      --file string             Read input from files.  May be given multiple times to add multiple file content
      --format string           Output format, must be either raw or markdown. (default "markdown")
  -h, --help                    help for ai
      --image-model string      Image LLM model (default "dall-e-3")
      --l1-model string         Level1 basic LLM model (default "gpt-4o-mini")
      --l2-model string         Level2 standard LLM model (default "gpt-4o")
      --l3-model string         Level3 advanced LLM model (default "o1-mini")
      --max-time int            Max number of seconds for timeout (default 3600)
      --max-turns int           Max number of turns (default 32)
      --mcp-server-url string   MCP server URL (default "http://localhost:58080/sse")
      --model string            LLM model (default "gpt-4o")
  -o, --output string           Save final response to a file.
      --pb-read                 Read input from the clipboard. Alternatively, append '=' to the command
      --pb-write                Copy output to the clipboard. Alternatively, append '=+' to the command
      --quiet                   Operate quietly
      --verbose                 Show debugging information
  -w, --workspace string        Workspace directory

Environment variables:
  AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_DOC_TEMPLATE, AI_DRY_RUN, AI_DRY_RUN_CONTENT, AI_EDITOR, AI_FILE, AI_FORMAT, AI_HELP, AI_IMAGE_API_KEY, AI_IMAGE_BASE_URL, AI_IMAGE_MODEL, AI_IMAGE_VIEWER, AI_INTERACTIVE, AI_L1_API_KEY, AI_L1_BASE_URL, AI_L1_MODEL, AI_L2_API_KEY, AI_L2_BASE_URL, AI_L2_MODEL, AI_L3_API_KEY, AI_L3_BASE_URL, AI_L3_MODEL, AI_LOG, AI_MAX_TIME, AI_MAX_TURNS, AI_MCP_SERVER_URL, AI_MESSAGE, AI_MODEL, AI_NO_META_PROMPT, AI_OUTPUT, AI_PB_READ, AI_PB_WRITE, AI_QUIET, AI_ROLE, AI_ROLE_PROMPT, AI_SQL_DB_HOST, AI_SQL_DB_NAME, AI_SQL_DB_PASSWORD, AI_SQL_DB_PORT, AI_SQL_DB_USERNAME, AI_TRACE, AI_VERBOSE, AI_WORKSPACE
```

```bash
$ ai help agents
```

```text
Available agents:

aider:	Integrate LLMs for collaborative coding, refactoring, bug fixing, and test development.
ask:	Deliver concise, reliable answers on a wide range of topics.
doc:	Create a polished document by integrating draft materials into the provided template.
draw:	Generate images based on user input, providing visual representations of text-based descriptions.
eval:	Evaluate and test tools.
git:	Automate git commit message creation for clarity and consistency in version control
git/conventional:	Generate concise git commit messages based on the provided diffs using the Conventional Commits specification
git/short:	Generate concise, one-line git commit messages based on the provided diffs.
gptr:	Deliver live, realtime, accurate, relevant insights from diverse online sources.
launch:	Dispatch to the most appropriate agent based on the user's input.
meta-prompt:	Generates a system prompt based on the user's input.
oh:	Engineering assistant promoting incremental development and detailed refactoring support.
pr:	Enhance PR management with automated summaries, reviews, suggestions, and changelog updates.
pr/changelog:	Update the CHANGELOG.md file with the PR changes
pr/describe:	Generate PR description - title, type, summary, code walkthrough and labels
pr/improve:	Provide code suggestions for improving the PR
pr/review:	Give feedback about the PR, possible issues, security concerns, review effort and more
shell:	Assist with scripting, command execution, and troubleshooting shell tasks.
sql:	Streamline SQL query generation, helping users derive insights without SQL expertise.
workspace:	Determines the user's workspace based on user's input.


/ is shorthand for @shell

Not sure which agent to use? Simply enter your message and AI will choose the most appropriate one for you:

ai "message..."
```

## Credits

+ @git System role prompt adapted from [Aider](https://github.com/Aider-AI/aider.git)
+ @pr  system role prompt adapted from [PR Agent](https://github.com/qodo-ai/pr-agent.git)
+ @sql system role prompt adapted from [Vanna](https://github.com/vanna-ai/vanna.git)

+ @code runs [Aider](https://github.com/Aider-AI/aider.git) in docker
+ @oh runs [OpenHands](https://github.com/All-Hands-AI/OpenHands.git) in docker
+ @seek runs [GPT Researcher](https://github.com/assafelovic/gpt-researcher.git) in docker
