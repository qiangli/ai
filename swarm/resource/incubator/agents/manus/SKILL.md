---
name: planning-with-files
version: "2.4.1"
description: 
user-invocable: true
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash
  - Glob
  - Grep
  - WebFetch
  - WebSearch
hooks:
  PreToolUse:
    - matcher: "Write|Edit|Bash|Read|Glob|Grep"
      hooks:
        - type: command
          command: "cat task_plan.md 2>/dev/null | head -30 || true"
  PostToolUse:
    - matcher: "Write|Edit"
      hooks:
        - type: command
          command: "echo '[planning-with-files] File updated. If this completes a phase, update task_plan.md status.'"
  Stop:
    - hooks:
        - type: command
          command: |
            SCRIPT_DIR="${CLAUDE_PLUGIN_ROOT:-$HOME/.claude/plugins/planning-with-files}/scripts"
            if command -v pwsh &> /dev/null && [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" || "$OS" == "Windows_NT" ]]; then
              pwsh -ExecutionPolicy Bypass -File "$SCRIPT_DIR/check-complete.ps1" 2>/dev/null || powershell -ExecutionPolicy Bypass -File "$SCRIPT_DIR/check-complete.ps1" 2>/dev/null || bash "$SCRIPT_DIR/check-complete.sh"
            else
              bash "$SCRIPT_DIR/check-complete.sh"
            fi
---
