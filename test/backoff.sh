#!/usr/bin/env ai /sh:bash --format raw --script
set -xue


echo ">>> Testing backoff..."

# backoff tests
/sh:backoff --command "/sh:exec --command 'pwd'"  --option duration="10s"

# /sh:exec --command "no_such_cmd"
# /sh:backoff --command "/sh:exec --command 'no_such_cmd'"  --option duration="10s"

/sh:backoff --command '/flow:choice --actions "[\"sh:pass\",\"invalid_action\",\"kit:invalid_tool\"]"'  --option duration="15s"
/sh:backoff --option action="/alias:choose" --option choose='/flow:choice --actions "[\"sh:pass\",\"no_such_cmd\",\"sh:pwd\",\"kit:invalid_tool\"]"' --option duration="15s"

echo "$?"
echo "*** backoff tests completed ***"
###