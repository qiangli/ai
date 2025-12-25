#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

# agent="${1:-ask}"
# agents="memory/memory" "memory/long_term" "memory/todo_list"

agents=("memory/memory" "memory/long_term" "memory/todo_list")

adapter="echo"
# template='data:,
# >>> History
# {{toPrettyJson .history}}

# >>> Environment
# {{printenv}}
# '
template='data:,
>>> Environment
{{printenv}}
'


# actions="[ai:call_llm,sh:format]"
actions="[ai:read_agent_config,ai:new_agent,ai:build_model,ai:build_query,ai:build_prompt,ai:build_context,ai:call_llm,sh:format]"

for agent in "${agents[@]}"
do
  echo "$agent"
  /flow:sequence  --actions "$actions" --agent "$agent" --adapter "$adapter" --template "$template" --message "hello"  $@
done

echo "$?"
echo "*** agent tests completed ***"
###