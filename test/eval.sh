#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

# built-in
BASE=$(pwd)
echo $BASE
# printenv

# core utils - must be full path
/bin/pwd
pwd

/bin/ls -al /tmp
ls -al /tmp

adapter="echo"

# actions='["ai:read_tool_config","sh:format"]'
# /flow:sequence --actions "$actions" --template "data:,{{toPrettyJson .config}}" --tool "fs:read_file" --format raw

# actions='["ai:read_agent_config","sh:format"]'
# /flow:sequence --actions "$actions" --template "data:,{{toPrettyJson .config}}\n\n agent: {{.agent}}\n kit: {{.kit}}\ nname: {{.name}}\n" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "sh:format"]'
# /flow:sequence --actions "$actions" --template "data:,{{toPrettyJson .agent}} \n kit: {{.kit}} \n name: {{.name}}" --agent "ed" --format raw

# actions='["ai:read_agent_config", "ai:new_agent", "ai:call_llm", "sh:format"]'
# /flow:sequence --actions "$actions" --template "data:,###\n****************{{toPrettyJson .result}}\n*************" --agent "ed" --message "correcto mine englise" --adapter "$adapter" --format raw

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

# /flow:sequence --actions "$actions" --template "$template" --agent "test" --adapter "$adapter" --output file:///tmp/eval.out


adapter="echo"
# actions='["ai:call_llm", "sh:format"]'
# actions='["ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm", "sh:format"]'
# actions='["ai:new_agent", "ai:build_query", "sh:format"]'
# actions='["ai:new_agent", "ai:build_agent", "ai:call_llm", "sh:format"]'
# actions='["ai:new_agent", "ai:build_agent", "sh:format"]'
# actions='["ai:new_agent", "sh:format"]'

# build_agent => build_query, build_prompt, build_context
# actions='["ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "sh:format"]'
# actions='["ai:new_agent", "ai:build_agent", "sh:format"]'

actions='["ai:new_agent", "ai:build_agent", "ai:call_llm", "sh:format"]'

# actions='["sh:format"]'
# actions='["ai:call_llm"]'

# {{.result.Value |fromJson |toPrettyJson}}

template='data:,
>>> Query:
{{.query}}

>>> Result:
{{.result |toPrettyJson}}

>>> Error:
{{.error}}

---
{{.agent}}
---
{{printenv}}
'
/flow:sequence --agent "joker" --message "what is new today" --actions "$actions" --template "$template" --adapter "$adapter" 


#  full flow:


# actions='["ai:read_agent_config", "ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm", "sh:format"]'

# template='data:,
# {{toPrettyJson .}}
# '

# /flow:sequence --actions "$actions" --template "$template" --agent "test" --adapter "$adapter" 

# command line in a terminal:
# test/eval.sh  --agent joker  --message "what is the weather in dublin ca in the next few days" --option tools='["web:fetch_content", "web:ddg_search"]'  --max-history 1 --max-turns 10

echo "$?"
echo "*** eval tests completed ***"
###