#!/bin/bash
set -xue

# this script can be run as system bash

# test/backoff.sh
# test/chain.sh
test/eval.sh
test/script.sh

# TODO fix concurrent writes for parallel
# test/flow.sh

# test/sb.json.sh
# test/sb.md
# test/sb.sh
# test/sb.txt
# test/sb.yaml
test/timout.sh
# test/yaml-wrap.sh

echo "Test completed"
