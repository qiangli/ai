#!/usr/bin/env ai /sh:bash --format raw --script
set -ue

# adapter="chat"
adapter="echo"

# {{.result.Value |fromJson |toPrettyJson}}

# template='data:,
# >>> Query:
# {{.query}}

# >>> Result:
# {{.result |toPrettyJson}}

# >>> Error:
# {{.error}}

# ---
# {{.agent}}
# ---
# {{printenv}}
# '

# actions='["ai:read_agent_config","ai:new_agent","ai:build_context","ai:call_llm","sh:format"]'
# actions='["ai:read_agent_config","ai:new_agent","ai:build_context","sh:format"]'

actions='["ai:spawn_agent","sh:format"]'

template='data:,

{{.history |toPrettyJson}}

'

# agent="ask"
# agent="context/history"
# agent="context/summary"
# agent="memory"
# agent="kbase"
# agent="think"
# agent="eval"
agent="chat"

/flow:sequence --actions "$actions" --template "$template" --agent "$agent" --adapter "$adapter" --message "what is new"

# /agent:atm/hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --adapter echo --info

echo "$?"
echo "*** eval tests completed ***"
###