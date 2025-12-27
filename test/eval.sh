#!/usr/bin/env ai /sh:bash --format raw --script
set -ue

# adapter="chat"
adapter="echo"

# {{.result.Value |fromJson |toPrettyJson}}
# actions='["ai:read_agent_config","ai:new_agent","ai:build_context","ai:call_llm","sh:format"]'
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


# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="memory/memory"
# # agent="memory/long_term"
# # agent="memory/todo_list"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# >>>>>>>>>
# {{printenv}}
# '

# ##
# # adapter="chat"
# actions='[ai:spawn_agent,sh:format]'
# agent="context/lastn"
# # agent="context/summary"
# template='data:,
# >>>>>>>>>
# {{.history | toPrettyJson}}

# >>>>>>>>>
# {{printenv}}
# '

# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="kbase"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# {{.tools | toPrettyJson}}
# >>>>>>>>>
# {{printenv}}
# '


# ##
# actions='[ai:spawn_agent,sh:format]'
# agent="think"
# template='data:,
# >>>>>>>>>
# {{.prompt}}

# >>>>>>>>>
# {{.tools | toPrettyJson}}

# >>>>>>>>>
# {{.model | toPrettyJson}}

# >>>>>>>>>
# {{printenv}}

# '

##
actions='[ai:spawn_agent,sh:format]'
template='data:,

>>>>>>>>> Instruction/Prompt
{{.prompt}}

>>>>>>>>> Context/History
{{.history}}

>>>>>>>>> Message/Query
{{.query}}

>>>>>>>>> Tools
{{.tools }}

>>>>>>>>> Model
{{.model }}

>>>>>>>>> Environment
{{printenv}}

'
adapter="chat"

agent="aider/detect_lang"
extra="--message write a simple hello world"
##
/flow:sequence --actions "$actions" --template "$template" --agent "$agent" --adapter "$adapter" $extra

# /agent:atm/hi --script "./swarm/atm/resource/template/atm.yaml" \
#     --adapter echo --info

echo "$?"
echo "*** eval tests completed ***"
###