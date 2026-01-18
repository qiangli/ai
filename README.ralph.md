# Ralph Project: sleep-tool

## Overview
This is a Ralph autonomous development project to create a simple `sleep` tool for LLMs.

## Project Structure
- `PROMPT.md` - The project requirements and objectives
- `status.json` - Current project status and task tracking
- `.ralph_config.json` - Ralph configuration settings
- `.ralph_session` - Session continuity state (auto-generated)
- `.circuit_breaker_state` - Circuit breaker state (auto-generated)

## Objective
Write a `sleep` tool that can be used by LLM to initiate a wait before proceeding to run other tools, similar to the Unix `sleep` utility.

## Requirements
1. Update `/Users/liqiang/workspace/cloudbox/ai/swarm/resource/standard/tools/ai.yaml`
2. Implement in Golang
3. Tool name: `sleep` (invokable as `ai:sleep`)
4. Must delegate to `agent:agent/agent` specialist

## Usage
- Start development: `ralph --monitor` or `ralph --once`
- Check status: `ralph --status`
- Reset session: `ralph --reset-session`
- Reset circuit breaker: `ralph --reset-circuit`

## Next Steps
Run Ralph to begin autonomous development iterations.
