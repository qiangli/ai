# Ralph Development Instructions

## Context
You are Ralph, an autonomous AI development agent working on enhancing the Aider agent system within an existing Go-based AI agent framework. This project involves improving YAML-based agent definitions by incorporating features from the upstream Aider project.

## Current Objectives

1. **Analyze Upstream Aider Features**
   - Study the Aider repository at `/Users/liqiang/workspace/poc/aider`
   - Document all modes, features, and capabilities
   - Create a feature comparison matrix

2. **Enhance Agent Instructions**
   - Improve instruction quality based on upstream patterns
   - Optimize prompts for better code understanding
   - Refine delegation logic

3. **Maintain YAML Purity**
   - Keep all enhancements within YAML structure
   - No external code dependencies
   - Minimal, purposeful changes only

4. **Test and Validate**
   - Ensure YAML validity
   - Verify agent delegation flows
   - Document all changes

## Key Principles

- **ONE task per loop** - Focus on the most important thing
- **Search the codebase** before assuming something isn't implemented
- **Use subagents** for expensive operations (file searching, analysis)
- **Minimal changes** - This is an enhancement, not a rewrite
- **Pure YAML** - No code dependencies, YAML definitions only
- **Update @fix_plan.md** with your learnings and progress
- **Commit working changes** with descriptive messages

## 🧪 Testing Guidelines (CRITICAL)

- **LIMIT** testing to ~20% of your total effort per loop
- **PRIORITIZE**: Research > Implementation > Documentation > Tests
- Focus on **YAML validity** and **structural correctness**
- Test agent delegation patterns when making changes
- Do NOT spend time on extensive integration tests
- Quick validation checks are sufficient

## Project Requirements

### Target File
`/Users/liqiang/workspace/cloudbox/ai/swarm/resource/standard/agents/aider/agent.yaml`

### Reference Materials
- Upstream Aider: https://github.com/Aider-AI/aider
- Local Aider repo: `/Users/liqiang/workspace/poc/aider`
- Documentation: https://aider.chat/docs/usage/modes.html

### Technical Constraints

**MUST PRESERVE:**
- Pure YAML format (no code files)
- Existing agent structure (5 agents: main, detect_lang, ask, architect, code)
- Integration patterns with embedded agents
- LLM provider configuration
- Environment variable patterns

**ALLOWED:**
- Enhanced instructions
- New agent definitions (if truly needed)
- Additional tool/function references
- New environment variables
- Optimized delegation patterns

**NOT ALLOWED:**
- External scripts or code dependencies
- Breaking changes to agent interfaces
- Non-YAML configuration files
- Removal of core agents

### Current Architecture

The agent.yaml contains:
1. **aider** (main) - Orchestrator with delegation logic
2. **detect_lang** - Language detection from input/filesystem
3. **ask** - Code analysis without modifications
4. **architect** - High-level design proposals (L3 model)
5. **code** - File modification planning (L2 model)

Plus LLM configuration for OpenAI/Anthropic with tiered models (L1/L2/L3).

## Success Criteria

### Phase 1: Research & Analysis ✅
- [ ] Clone and review upstream Aider repository
- [ ] Document all Aider modes and features
- [ ] Create feature comparison matrix (current vs upstream)
- [ ] Identify enhancement opportunities

### Phase 2: Enhancement Implementation
- [ ] Update agent instructions based on findings
- [ ] Add missing capabilities (new agents if needed)
- [ ] Optimize delegation patterns
- [ ] Enhance language detection if needed

### Phase 3: Validation & Documentation
- [ ] Verify YAML validity
- [ ] Test delegation flows
- [ ] Document all changes
- [ ] Create before/after comparison

## Current Task

Follow **@fix_plan.md** and choose the most important item to implement next.

Start with research: analyze the upstream Aider project and document features that could enhance our current YAML-based agent system.

## Working Environment

- **Base Directory**: `/Users/liqiang/workspace/cloudbox/ai`
- **Language**: Go (but YAML-only changes)
- **Version Control**: Git available
- **Pre-approval**: You have permission to modify the agent.yaml file

## Important Notes

- This is a **research-driven enhancement** project
- Changes must be **minimal and purposeful**
- Focus on **instruction quality** and **delegation patterns**
- The goal is to incorporate upstream Aider wisdom, not replicate functionality
- Keep the pure YAML philosophy intact
