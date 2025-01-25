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
  ai message...
  ai [OPTIONS] AGENT [message...]

Examples:

ai what is fish?
ai / what is fish?
ai @ask what is fish?

Agent:
  /[command]       [message...] Get help with system command and shell scripting tasks
  @[agent/command] [message...] Engage specialist agents for assistance with complex tasks

Use "ai help" for more info.
```

```bash
$ ai help
```

```text
AI Command Line Tool

Usage:
  ai message...
  ai [OPTIONS] AGENT [message...]

. Ask for help with writing or debugging shell scripts.
. Request explanations for specific shell commands or scripts.
. Get assistance with writing, optimizing, or debugging SQL queries.
. Seek guidance on writing code or debugging in various programming languages.


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
  /                       List system commands available in the path
  @                       List all supported agents
  setup                   Setup the AI configuration

Options:
      --api-key string           LLM API key
      --base-url string          LLM Base URL (default "https://api.openai.com/v1/")
      --config string            config file (default "/Users/qiang.li/.ai/config.yaml")
      --doc-template string      Document template file
      --editor string            Specify editor to use (default "vi")
      --file string              Read input from files.  May be given multiple times to add multiple file content
      --format string            Output format, must be either raw or markdown. (default "markdown")
  -h, --help                     help for ai
  -i, --interactive              Interactive mode to run, edit, or copy generated code
      --l1-model string          Level1 basic LLM model (default "gpt-4o-mini")
      --l2-model string          Level2 standard LLM model (default "gpt-4o")
      --l3-model string          Level3 advanced LLM model (default "o1-mini")
      --log string               Log all debugging information to a file
      --model string             LLM model (default "gpt-4o")
  -n, --no-meta-prompt           Disable auto generation of system prompt
      --output string            Save final response to a file.
      --pb-read                  Read input from the clipboard. Alternatively, append '=' to the command
      --pb-write                 Copy output to the clipboard. Alternatively, append '=+' to the command
      --quiet                    Operate quietly
      --sql-db-host string       Database host
      --sql-db-name string       Database name
      --sql-db-password string   Database password
      --sql-db-port string       Database port
      --sql-db-username string   Database username
      --verbose                  Show debugging information
  -w, --workspace string         Workspace directory

Environment variables:
  AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_DOC_TEMPLATE, AI_DRY_RUN, AI_DRY_RUN_CONTENT, AI_EDITOR, AI_FILE, AI_FORMAT, AI_HELP, AI_INTERACTIVE, AI_L1_API_KEY, AI_L1_BASE_URL, AI_L1_MODEL, AI_L2_API_KEY, AI_L2_BASE_URL, AI_L2_MODEL, AI_L3_API_KEY, AI_L3_BASE_URL, AI_L3_MODEL, AI_LOG, AI_MODEL, AI_NO_META_PROMPT, AI_OUTPUT, AI_PB_READ, AI_PB_WRITE, AI_QUIET, AI_ROLE, AI_ROLE_PROMPT, AI_SQL_DB_HOST, AI_SQL_DB_NAME, AI_SQL_DB_PASSWORD, AI_SQL_DB_PORT, AI_SQL_DB_USERNAME, AI_TRACE, AI_VERBOSE, AI_WORKSPACE
```

```bash
$ ai @
```

```text
Available agents:

ask:	All-encompassing Q&A platform providing concise, reliable answers on diverse topics.
code:	Integrates LLMs for collaborative coding, refactoring, bug fixing, and test development.
doc:	Create a polished document by integrating draft materials into the provided template.
git:	Automates Git commit message creation for clarity and consistency in version control.	
  /short:        Generate a short commit message for Git based on the provided information
  /conventional: Generate a commit message for Git based on the provided information according to the Conventional Commits specification at https://www.conventionalcommits.org/en/v1.0.0/#summary

oh:	Engineering assistant promoting incremental development and detailed refactoring support.
pr:	Enhances PR management with automated summaries, reviews, suggestions, and changelog updates.	
  /describe:  Generate PR description - title, type, summary, code walkthrough and labels
  /review:    Feedback about the PR, possible issues, security concerns, review effort and more
  /improve:   Code suggestions for improving the PR
  /changelog: Update the CHANGELOG.md file with the PR changes

script:	Receive assistance to execute system commands, create and troubleshoot various shell scripts.	
  Run "ai list-commands" tool to get the complete list of system commands available in the path.

seek:	Digital exploration tool delivering accurate, relevant insights from diverse online sources.
sql:	Streamlines SQL query generation, helping users derive insights without SQL expertise.


/ is shorthand for @script

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
