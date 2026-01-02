#!/bin/bash
# this script runs as regular system bash

set -ue

BASE=$(pwd)
echo $BASE

##
# $BASE/test/script.sh
$BASE/test/bash-basic.sh
$BASE/test/bash-ext.sh

$BASE/test/flow.sh

##
adapter="--adapter echo"
#$BASE/sb.json.sh
$BASE/test/sb.md $adapter
$BASE/test/sb.sh $adapter
$BASE/test/sb.txt $adapter
$BASE/test/sb.yaml $adapter
$BASE/test/yaml-wrap.sh $adapter

##
$BASE/test/timeout.sh
$BASE/test/backoff.sh
$BASE/test/chain.sh

$BASE/test/agent.sh

$BASE/test/std.sh

echo "$?"
echo "*** All tests completed successfully ***"
###

