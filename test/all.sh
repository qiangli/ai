#!/bin/bash
set -xue

# this script can be run as system bash

test/eval.sh
test/script.sh

# TODO fix concurrent writes for parallel
# test/flow.sh

# test/sb.json.sh
# test/sb.md
# test/sb.sh
# test/sb.txt
# test/sb.yaml
# test/yaml-wrap.sh

test/timeout.sh

test/backoff.sh

# test/chain.sh
echo ""
echo "*** Test completed ***"
