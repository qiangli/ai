#!/usr/bin/env ai /sh:bash --format raw --script
# set -xue

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

choose="/flow:choice --option actions=[sh:pass,no_such_cmd,sh:pwd,kit:invalid_tool]"

/flow:chain --option chain=[sh:timeout,sh:backoff,alias:cmd] \
    --option cmd="${choose}" \
    --option duration="60s"

echo "*** 🎉 chain test completed ***"
exit 0
###
