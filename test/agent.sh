#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

# agent="${1:-ask}"
agent="memory"
adapter="echo"
template='data:,
>>> History
{{toPrettyJson .history}}
'

# actions="[ai:call_llm,sh:format]"
actions="[ai:read_agent_config,ai:new_agent,ai:build_model,ai:build_query,ai:build_prompt,ai:build_context,ai:call_llm,sh:format]"

/flow:sequence  --actions "$actions" --agent "$agent" --adapter "$adapter" --template "$template" $@

echo "$?"
echo "*** agent tests completed ***"
###