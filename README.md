# AI Command Line Tool

As a software engineer, I need two types of tools for my daily tasks: one for working inside a file and the other outside.

This AI tool assists you with all the tasks beyond file editing on your system complementing Continue, Cline and the like.

## Build

```bash
git clone https://github.com/qiangli/ai.git
cd ai
# make build
# just build
./build.sh
```

## Run

```bash
# command line
ai [OPTIONS] AGENT [message...]

# setup (optional)
# minimum requirement: OPENAI_API_KEY or GEMINI_API_KEY is set
ai /setup
ai /help info

#
ai @ask "What is the capital of France?"
git diff origin main|ai @git/long commit message
ai / What tools could I use to search for a pattern in files

ai --help

# interactive shell
ai -i

ssh --tty user@host ai -i
```

## Test

No API calls will be made in `dry-run` mode.

```bash
ai --dry-run --dry-run-content "fake data" ...
```

## Debug

```bash
ai --verbose ...

ai /help info
```

## Usage

### Command line

```bash
$ ai
```

```text
AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...

Examples:

ai what is fish
ai / what is fish
ai @ask what is fish


Use "ai /help [agents|commands|tools|info]" for more information.
```

```bash
$ ai /help
```

```text
ai /help
AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...

There are multiple ways to interact with this AI tool.

+ Command line input:

  ai @agent what is fish?

+ Read from standard input:

  ai @agent --stdin
  ai @agent -

Ctrl+D to send, Ctrl+C to cancel.

+ Here document:

  ai @agent <<eof
what is the weather today?
eof

+ Piping input:

  git diff origin/main | ai @agent [message...]

+ File redirection:

  ai @agent [message...] < file.txt

+ Reading from system clipboard:

  ai @agent [message...] --pb-read
  ai @agent [message...] {
  ai @agent [message...] --pb-read-wait
  ai @agent [message...] {{

Use system copy (Ctrl+C on Unix) to add selected contents.
Ctrl+C to cancel.

+ Composing with text editor:

  export AI_EDITOR=nano # default: vi
  ai @agent


Miscellaneous:

  ai /mcp                        Manage MCP server
  ai /setup                      Setup configuration


Options:
      --agent string            Specify the agent/command to use. Same as @agent/command (default "ask")
      --api-key string          LLM API key
      --base-url string         LLM Base URL (default "https://api.openai.com/v1/")
      --config string           config file (default "/Users/liqiang/.ai/config.yaml")
      --file string             Read input from files.  May be given multiple times to add multiple file content
      --format string           Output format, one of text, json, or markdown. (default "markdown")
  -h, --help                    help for ai
      --image-model string      Image LLM model (default "dall-e-3")
      --input string            Read input message from a file
  -i, --interactive             Interactive mode
      --internal                Enable internal agents and tools
      --l1-model string         Level1 basic LLM model (default "gpt-4.1-mini")
      --l2-model string         Level2 standard LLM model (default "gpt-4.1")
      --l3-model string         Level3 advanced LLM model (default "o4-mini")
      --max-time int            Max number of seconds for timeout (default 3600)
      --max-turns int           Max number of turns (default 16)
      --mcp-server-url string   MCP server URL
      --message string          Specify input message. Overrides all other input methods
      --model string            LLM model (default "gpt-4.1")
  -o, --output string           Save final response to a file.
      --pb-read                 Read input from the clipboard. Alternatively, use '{'
      --pb-read-wait            Read input from the clipboard and wait for confirmation. Alternatively, use '{{'
      --pb-write                Copy output to the clipboard. Alternatively, use '}'
      --pb-write-append         Append output to the clipboard. Alternatively, use '}}'
      --quiet                   Operate quietly. Only show final response
      --shell string            Shell to use for interactive mode (default "/bin/bash")
      --stdin                   Read input message from stdin. Alternatively, use '-'
      --template string         Document template file
      --unsafe                  Skip command security check to allow unsafe operations. Use with caution
      --verbose                 Show progress and debugging information
  -v, --version                 version for ai
      --watch                   Watch the workspace directory and respond to embedded ai requests in files
  -w, --workspace string        Workspace directory

Environment variables:
  AI_AGENT, AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_DRY_RUN, AI_DRY_RUN_CONTENT, AI_EDITOR, AI_FILE, AI_FORMAT, AI_HELP, AI_HOST, AI_IMAGE_API_KEY, AI_IMAGE_BASE_URL, AI_IMAGE_MODEL, AI_IMAGE_VIEWER, AI_INPUT, AI_INTERACTIVE, AI_INTERNAL, AI_L1_API_KEY, AI_L1_BASE_URL, AI_L1_MODEL, AI_L2_API_KEY, AI_L2_BASE_URL, AI_L2_MODEL, AI_L3_API_KEY, AI_L3_BASE_URL, AI_L3_MODEL, AI_LOG, AI_MAX_TIME, AI_MAX_TURNS, AI_MCP_SERVER_URL, AI_MESSAGE, AI_MODEL, AI_OUTPUT, AI_PB_READ, AI_PB_READ_WAIT, AI_PB_WRITE, AI_PB_WRITE_APPEND, AI_PORT, AI_QUIET, AI_ROLE, AI_ROLE_PROMPT, AI_SHELL, AI_SQL_DB_HOST, AI_SQL_DB_NAME, AI_SQL_DB_PASSWORD, AI_SQL_DB_PORT, AI_SQL_DB_USERNAME, AI_STDIN, AI_TEMPLATE, AI_UNSAFE, AI_VERBOSE, AI_VERSION, AI_WATCH, AI_WORKSPACE

Use "ai /help [agents|commands|tools|info]" for more information.
```

### Interactive shell

```bash
ai -i
```

```
ai.git@main/. ai> help
  exit                     │  exit ai shell
  history [-c]             │  display or clear command history
  alias [name[=value]      │  set or print aliases
  env [name[=value]        │  export or print environment
  source [file]            │  set alias and environment from file
  edit [file]              │  text editor
  explore [--help] [path]  │  explore local file system
  | page                   │  similar to more or less
  help                     │  help for ai shell
  @[agent]                 │  agent command
  /[command]               │  slash (shell agent) command

  Key Bindings:
  Ctrl + A	Go to the beginning of the line (Home)
  Ctrl + E	Go to the end of the line (End)
  Ctrl + P	Previous command (Up arrow)
  Ctrl + N	Next command (Down arrow)
  Ctrl + F	Forward one character
  Ctrl + B	Backward one character
  Ctrl + D	Delete character under the cursor
  Ctrl + H	Delete character before the cursor (Backspace)
  Ctrl + W	Cut the word before the cursor to the clipboard
  Ctrl + K	Cut the line after the cursor to the clipboard
  Ctrl + U	Cut the line before the cursor to the clipboard
  Ctrl + L	Clear the screen

```

## Credits

+ @git System role prompt adapted from [Aider](https://github.com/Aider-AI/aider.git)
+ @pr  system role prompt adapted from [PR Agent](https://github.com/qodo-ai/pr-agent.git)
+ @sql system role prompt adapted from [Vanna](https://github.com/vanna-ai/vanna.git)

+ @code runs [Aider](https://github.com/Aider-AI/aider.git) in docker
+ @oh runs [OpenHands](https://github.com/All-Hands-AI/OpenHands.git) in docker
+ @seek runs [GPT Researcher](https://github.com/assafelovic/gpt-researcher.git) in docker
