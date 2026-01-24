#!/usr/bin/env ai /sh:bash --format raw --script
set -ue


echo ">>> Testing backoff..."

# backoff tests
# system command action
/sh:backoff --command "/sh:exec --command 'printenv'"  --option duration="10s"
# tool action
/sh:backoff --command "/fs:list_roots"  --option duration="10s"
# agent action
/sh:backoff --command "/agent:ed/ed --adapter echo --output none"  --option duration="10s"
# flow:choice
/sh:backoff --command "/flow:choice --actions '[sh:pass,invalid_action,kit:invalid_tool]'"  --duration "15s"
#
/sh:backoff --action "/alias:choose" --option choose="/flow:choice --actions '[sh:printenv,no_such_cmd,sh:pwd,kit:invalid_tool]'" --duration "15s"

echo "$?"
echo "*** 🎉 backoff tests completed ***"
###