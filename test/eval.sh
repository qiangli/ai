#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue

adapter="echo"

# actions='["ai:read_tool_config","sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}" --tool "fs:read_file" --format raw

# actions='["ai:read_agent_config","sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}\n\n agent: {{.agent}}\n kit: {{.kit}}\ nname: {{.name}}\n" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "sh:format"]'
# /sh:flow --actions "$actions" --template "data:,{{toPrettyJson .agent}} \n kit: {{.kit}} \n name: {{.name}}" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "ai:call_llm", "sh:format"]'
# /sh:flow --actions "$actions" --template "data:,###\n****************{{toPrettyJson .result}}\n*************" --agent "ed" --message "correcto mine englise" --adapter "$adapter" --format raw

# actions='["ai:read_agent_config", "ai:new_agent",  "sh:format"]'
# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "sh:format"]'
# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "ai:build_prompt", "sh:format"]'
# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "sh:format"]'

# template='data:,
# >>> env:
# {{printenv}}

# >>> query:
# {{.query}}

# >>> instruction:
# {{.prompt}}

# >>> context:
# {{toPrettyJson .history}}
# '

# /sh:flow --actions "$actions" --template "$template" --agent "test" --adapter "$adapter" --output file:///tmp/eval.out


adapter=""
# actions='["ai:call_llm", "sh:format"]'
actions='["ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm", "sh:format"]'
# actions='["ai:new_agent", "ai:build_query", "sh:format"]'

# actions='["sh:format"]'
# actions='["ai:call_llm"]'

# {{.result.Value |fromJson |toPrettyJson}}

template='data:,
>>> Query:
{{.query}}

>>> Result:
{{.result.Value}}

>>> Error:
{{.error}}

---
{{printenv}}
'
/sh:flow --actions "$actions" --template "$template" --adapter "$adapter" 

#  full flow:


# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm", "sh:format"]'

# template='data:,
# {{toPrettyJson .}}
# '

# /sh:flow --actions "$actions" --template "$template" --agent "test" --adapter "$adapter" 


###