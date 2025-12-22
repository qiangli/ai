#!/usr/bin/env ai /sh:bash --format raw --script
set -xue


echo "testing..."

# backoff tests
# /sh:backoff --command "/sh:exec --command 'ls -al'"  --option duration="10s"

# /sh:exec --command "no_such_cmd"
# /sh:backoff --command "/sh:exec --command 'no_such_cmd'"  --option duration="10s"

/sh:backoff --command "/flow:choice --actions '[\"sh:pass\",\"no_such_cmd\",\"invalid_action\", \"kit:invalid_kit\"]'"  --option duration="10s"

#