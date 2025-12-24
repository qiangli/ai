#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue


echo ">>> Testing chain actions..."

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

# /alias:ls  --option ls="ls -al /tmp"
# /flow:sequence --option actions='["fs:list_roots","agent:ed","alias:cmd","sh:format"]'  --option adapter="echo" --option query="what is unix" --option cmd="ls -al /tmp" --option template=$template

# 

# cmd=/ai:spawn_agent --agent ed --adapter echo -- correct me
/flow:chain --option chain='["sh:timeout","sh:backoff","alias:cmd"]' \
    --option cmd="/ai:spawn_agent --agent ed --adapter echo -- correct me" \
    --option duration="10s" 

exit 0
###
