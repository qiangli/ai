#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue
set -ue

echo ">>> Testing chain actions..."

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
# # cmd=/ai:spawn_agent --agent ed --adapter echo -- correct me
# /flow:chain --option chain='["sh:timeout","sh:backoff","alias:cmd"]' \
#     --option cmd="/ai:spawn_agent --agent ed --adapter echo -- correct me" \
#     --option duration="10s" 

# cmd="/ai:spawn_agent --agent ed --adapter echo -- correct me"
# /flow:chain --option chain='["sh:timeout","sh:backoff","alias:cmd"]' \
#     --option cmd="$cmd" \
#     --option duration="10s" 

# function backoff() {
#     /sh:backoff --option action=/alias:choose --option choose='/flow:choice --option actions=[\"sh:pass\",\"no_such_cmd\",\"sh:pwd\",\"kit:invalid_tool\"]' --option duration=15s
# }

# backoff 
# $(backoff)
# alias bko=backoff; bko
# export cmd="ls -al /opt"
# /sh:bash --option script="data:,$cmd"
# #TODO expandenv for scripts: /sh:bash --option script='data:,$cmd'

# choose='/flow:choice --option actions=["sh:pass","no_such_cmd","sh:pwd","kit:invalid_tool"]'
choose="/flow:choice --option actions=[sh:pass,no_such_cmd,sh:pwd,kit:invalid_tool]"
# $choose
# cmd="/sh:backoff --option action=/alias:choose --option choose=${choose} --option duration=15s"
# $cmd
/flow:chain --option chain=[sh:timeout,sh:backoff,alias:cmd] \
    --option cmd="${choose}" \
    --option duration="60s"

exit 0
###
