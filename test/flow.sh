#!/usr/bin/env ai /sh:bash --verbose --script
set -xue

echo "# Flow Test"
DRY=""

#
BASE=$(pwd)
echo $BASE
printenv

echo "## Sequence"

# $DRY /sh:flow --actions '["sh:parse", "ai:get_envs", "sh:format"]'
$DRY /sh:flow --actions '["sh:parse", "ai:get_envs", "sh:format"]'

#
printenv
echo $?
exit 0
#
