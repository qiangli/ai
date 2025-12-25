#!/bin/bash
# this script runs as regular system bash

set -ue

BASE=$(pwd)
echo $BASE

##
# $BASE/test/eval.sh
$BASE/test/script.sh
$BASE/test/flow.sh

##
adapter=echo
#$BASE/sb.json.sh
$BASE/test/sb.md --adapter $adapter
$BASE/test/sb.sh --adapter $adapter
$BASE/test/sb.txt --adapter $adapter
$BASE/test/sb.yaml --adapter $adapter
$BASE/test/yaml-wrap.sh --adapter $adapter

##
$BASE/test/timeout.sh
$BASE/test/backoff.sh
$BASE/test/chain.sh

echo "$?"
echo "*** All tests completed successfully ***"
###

