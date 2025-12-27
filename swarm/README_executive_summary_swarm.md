Executive summary: Swarm codebase (Go) in /swarm

Purpose:
- A lightweight, pluggable, multi-agent orchestration framework. It loads agent configurations, executes actions via a mix of built-in commands, templates, and LLM-powered tools, and supports memory/history for conversations.

Core architecture:
- Swarm (core orchestrator): central runtime that initializes, loads agents, parses inputs, dispatches actions to either toolkits or LLM adapters, and formats outputs.
- api package: core domain models and interfaces, including Agent, ToolFunc, Model, MemStore, and ArgMap. Provides kits/toolbar abstraction and memory/history support.
- atm: template and resource templates, and format rendering. Root agent data and root/root templates live here, enabling dynamic behavior with Go templates.
- mcp: MCP client support for external memory/context providers (optional extension for distributed memory).
- llm adapters: integration layer to call external LLMs through adapters, with support for caching and agent/tool scoping.
- tool system: registry and dispatch logic for toolkits (func, web, mcp, system, agent, ai, bin), plus kit loading and lifetime management.
- agent execution helpers: agent_script.go supports script-based execution via a virtual shell with environment propagation and command execution.
- assets and loader patterns: NewConfigLoader and related code load agent/tool configurations (YAML/JSON) and instantiate runnable agents.
- memory/history: api.MemStore/History used to persist chat context, enabling recall and context carryover across sessions.
- mcp integration: McpClient provides a client surface to MCP servers for distributed memory or tool access.

Key data flows:
- Init: Build root memory and root agent, wire up environment, and bootstrap the runtime.
- Parse/Exec: Inputs are parsed into an ArgMap; the root agent runs actions via Runner; ToolFunc dispatch leads to various backends (bin commands, AI calls, other agents).
- Tool flow: Tools come from registered Kits; AIKit handles LLM prompts, chaining prompts, and tool selection; history is appended with system/user/assistant messages and persisted.
- Template support: Templates in the root resource allow dynamic prompt generation and format rendering for outputs.

Notable files and responsibilities:
- core.go: main orchestration and lifecycle of swarm; loading root agent; creating/enabling child agents; dispatching actions.
- agent_script.go: Run bash-like scripts and integrate with the agent environment; handles nested AI/tool calls.
- tool_ai.go: high-level AI kit, including LLM call orchestration, model loading, caching of agents/tools, tool invocation, and history management.
- tool_system.go: registry for toolkits; GetKit/AddKit to wire up different toolkits (func, web, mcp, system).
- api/*: core data models; actions, tools, assets, memory, and environments.
- atm/resource: templates including root agent data and format templates.
- mcp/*: MCP client integration for external memory/compute backends.

Implementation notes:
- Uses Go template engine for dynamic config, with rechargeable root agent base.
- Supports embedding agents (Embed) and inheriting tools.
- Caching for agent/tool listings to speed up CLI-like introspection.
- Uses a pluggable model integration (llm adapters) supporting provider keys and secret handling.

If you want, I can:
- Produce a one-page diagram mapping file -> responsibility.
- Extract a quick-start guide (boot sequence, how to add an agent, how to call a tool).
- Generate a radar chart of major risks and mitigations.
