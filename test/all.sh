#!/bin/bash
# this script runs as regular system bash

set -ue

BASE=$(pwd)
echo $BASE

$BASE/test/eval.sh

$BASE/test/script.sh

#
$BASE/test/flow.sh

#$BASE/sb.json.sh
#$BASE/sb.md
#$BASE/sb.sh
#$BASE/sb.txt
#$BASE/sb.yaml
#$BASE/yaml-wrap.sh

$BASE/test/timeout.sh

$BASE/test/backoff.sh

#$BASE/chain.sh

echo "$?"
echo "*** All tests completed successfully ***"
###

