#!/usr/bin/env ai /sh:bash --format raw --script
set -xue

echo ">> Testing timer..."

time /sh:exec --command "sleep 2"
time /sh:bash --script "data:,sleep 2"
time /agent:joker --message "explain timeout in a unix system" --adapter "echo" --output none

echo ">>> Testing timeout..."

/sh:timeout --command "/sh:exec --command 'pwd'" --option duration="10s"
/sh:timeout --option action="/alias:list_roots" --option list_roots="/sh:exec --command 'ls /tmp'" --option duration="3s"

echo "*** 🎉 timeout test completed ***"
###
