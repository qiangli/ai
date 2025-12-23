#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue


echo ">>> Testing chain..."
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

# /sh:backoff --command '/flow:choice --actions "[\"sh:pass\",\"invalid_action\", \"sh:pwd\", \"kit:invalid_kit\"]"'  --option duration="15s"
# /sh:backoff --option action="/alias:choose" --option choose='/flow:choice --actions "[\"sh:pass\",\"no_such_cmd\", \"sh:pwd\", \"kit:invalid_kit\"]"' --option duration="15s"

# /sh:timeout --command "/sh:exec --command 'pwd'" --option duration="10s"
# /sh:timeout --option action="/alias:list_roots" --option list_roots="/sh:exec --command 'ls /'" --option duration="3s"

set -x

# actions="[\"sh:pass\",\"invalid_action\",\"sh:pwd\",\"kit:invalid_kit\",\"sh:pass\"]"
# choose="/flow:choice --actions "$actions""
# $choose

/flow:chain --actions '[\"sh:timeout\",\"sh:backoff\",\"alias:flow\"]' --option duration="10s" --option flow="/sh:pass"

exit 0
###
