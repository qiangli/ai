#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue


echo "# Flow Test"
DRY=""

#
# BASE=$(pwd)
# echo $BASE
# printenv

# {{.result.Value |fromJson |toPrettyJson}}
template='data:,
*** Query:
{{.query}}

*** Result:
{{.result}}

*** Error:
{{.error}}

*** Environment:
{{printenv}}
'

# $DRY /flow:chain --actions '["sh:pwd", "fs:list_roots", "agent:ed", "sh:format"]'  --option adapter="echo" --option query="what is unix" --option template=$template
# $DRY /flow:chain --actions '["sh:pwd", "fs:list_roots", "agent:ed", "sh:format"]'  --option adapter="echo" --option query="what is unix" --option template='data:,{{.query}}'
# $DRY /flow:chain --actions '["sh:parse", "ai:call_llm", "sh:format"]' --option command="/ai:pass --option query='tell me a joke' --option template='data:,>>>this is a test\n {{.result}} error: {{.error}}'" --output console

# /sh:timeout --command "/sh:exec --command 'sleep 10'"  --option duration="3s"
# /sh:backoff --command "/flow:choice --actions '[\"sh:pass\",\"no_such_cmd\",\"invalid_action\", \"kit:invalid_kit\"]'"  --option duration="10s"

# $DRY /flow:chain --actions '["sh:timeout", "sh:backoff", "ai:call_llm", "sh:format"]' --option command="/ai:pass --option query='tell me a joke' --option template='data:,>>>this is a test\n {{.result}} error: {{.error}}'" --output console

# alias cmd='/flow:sequence --actions ["sh:parse", "ai:call_llm", "sh:format"]'
#  /flow:chain --actions '["sh:timeout", "sh:backoff", "alias:cmd"]'
#
# printenv
# echo $?
exit 0
#
