# AI Command Line Tool

As a software engineer, I need two types of tools for my daily tasks: one for working inside a file and the other outside.

This AI tool assists you with all the tasks beyond file editing on your system complementing [Continue](https://github.com/openaide/awesome/tree/main/docker/continue), [Cline](https://github.com/openaide/awesome/tree/main/docker/continue) and [the like](https://github.com/openaide/awesome).

Specialist agents - such as shell, web, git, pr, code, and sql - will empower you to be much more productive...

If you prefer graphical UI, this tool can serve as backend hub service to the web widget, browser/vscode extensions, and desktop app, including a web terminal: [AI Chatbot](https://github.com/qiangli/chatbot) Conversation history is shared among all the different UIs so LLMs won't lose the context when switching the interfaces.


- [AI Command Line Tool](#ai-command-line-tool)
  - [Build](#build)
  - [Run](#run)
  - [Test](#test)
  - [Debug](#debug)
  - [Usage](#usage)
    - [Command line](#command-line)
    - [Interactive shell](#interactive-shell)
    - [AI Hub service](#ai-hub-service)
    - [AI Graphical UI](#ai-graphical-ui)
  - [Credits](#credits)


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

ai --help

# setup (optional)
# minimum requirement: OPENAI_API_KEY or GEMINI_API_KEY environment variable is set
ai /setup
ai /help info

# @ask is optional and the default agent by default
ai @ask what is the capital of France?

# generate a commit message based on the diff from stdin
git diff origin main | ai @git/long commit message

# / is short for @shell/
ai / What tools could I use to search for a pattern in files

# interactive shell
ai -i

# run ai (interative) over ssh
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
  ai @agent [message...] --pb-tail
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
  -a, --agent string         Specify the agent to use or @agent
      --allow string         List of comma separated system commands allowed for tool calls
      --api-key string       LLM API key
      --base-url string      LLM Base URL
      --config string        config file (default "/Users/liqiang/.ai/config.yaml")
      --deny string          List of comma separated system commands disallowed for tool calls. Approval is required to proceed. Ignored if 'unsafe' is true (default "rm")
  -e, --edit                 Launch editor
      --editor string        Specify the editor to use. default: builtin
      --format string        Output format: raw, text, json, or markdown. (default "markdown")
  -h, --help                 help for ai
      --image-model string   Image LLM model
      --input string         Read input message from a file
  -i, --interactive          Interactive mode
      --l1-model string      Level1 basic LLM model
      --l2-model string      Level2 standard LLM model
      --l3-model string      Level3 advanced LLM model
      --max-turns int        Max number of turns (default 16)
      --model string         LLM default model
  -n, --new                  Start a new converston
      --output string        Save final response to a file.
      --pb-read              Read input from clipboard. '{'
      --pb-tail              Read input from clipboard and wait. '{{'
      --pb-write             Copy output to clipboard. '}'
      --pb-append            Append output to clipboard. '}}'
      --quiet                Operate quietly. Only show final response
      --shell string         Shell to use for interactive mode (default "/bin/bash")
      --stdin                Read input from stdin. '-'
      --unsafe               Skip command security check to allow unsafe operations. Use with caution
      --verbose              Show progress and debugging information
  -v, --version              version for ai

Environment variables:
  AI_AGENT, AI_ALLOW, AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_CONTENT, AI_DENY, AI_DRY_RUN, AI_DRY_RUN_CONTENT, AI_EDIT, AI_EDITOR, AI_FILE, AI_FORMAT, AI_HELP, AI_HOST, AI_IMAGE_API_KEY, AI_IMAGE_BASE_URL, AI_IMAGE_MODEL, AI_IMAGE_VIEWER, AI_INPUT, AI_INTERACTIVE, AI_INTERNAL, AI_L1_API_KEY, AI_L1_BASE_URL, AI_L1_MODEL, AI_L2_API_KEY, AI_L2_BASE_URL, AI_L2_MODEL, AI_L3_API_KEY, AI_L3_BASE_URL, AI_L3_MODEL, AI_LOG, AI_MAX_HISTORY, AI_MAX_SPAN, AI_MAX_TIME, AI_MAX_TURNS, AI_MCP_SERVER_URL, AI_MESSAGE, AI_MODEL, AI_NEW, AI_OUTPUT, AI_PB_READ, AI_PB_TAIL, AI_PB_WRITE, AI_PB_APPEND, AI_PORT, AI_QUIET, AI_ROLE, AI_ROLE_PROMPT, AI_SHELL, AI_SQL_DB_HOST, AI_SQL_DB_NAME, AI_SQL_DB_PASSWORD, AI_SQL_DB_PORT, AI_SQL_DB_USERNAME, AI_STDIN, AI_TEMPLATE, AI_UNSAFE, AI_VERBOSE, AI_VERSION, AI_WATCH, AI_WORKSPACE

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

### AI Hub service

Start AI in Hub service mode

```bash
# example:
# ai --hub --agent swe --l2-model openai/gpt-4.1 --verbose --new
#
# ai --agent ask --verbose --hub --hub-address ":58080" --hub-pg-address ":25432" --hub-mysql-address ":3306" --hub-redis-address ":6379"
just hub
```

### AI Graphical UI

Please see Chatbot at [https://github.com/qiangli/chatbot](https://github.com/qiangli/chatbot)



## Credits

+ @code system role prompts adapted from [screenshot-to-code](https://github.com/abi/screenshot-to-code)
+ @git system role prompt adapted from [Aider](https://github.com/Aider-AI/aider.git)
+ @pr  system role prompt adapted from [PR Agent](https://github.com/qodo-ai/pr-agent.git)
+ @sql system role prompt adapted from [Vanna](https://github.com/vanna-ai/vanna.git)

+ @aider runs [Aider](https://github.com/Aider-AI/aider.git) in docker
+ @oh runs [OpenHands](https://github.com/All-Hands-AI/OpenHands.git) in docker
+ @gptr runs [GPT Researcher](https://github.com/assafelovic/gpt-researcher.git) in docker
