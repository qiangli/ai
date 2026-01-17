#!/usr/bin/env ai /sh:bash --format raw --script
# set -ue

message="tell me a joke"

# /sh:set_envs --option query="$message"
# fix me
# datetime=$(date)
datetime="Jan  1 08:08:08"

envs="
{
\"datetime\":\"$datetime\",
\"message\":\"$message\"
}
"
/sh:set_envs --option envs="${envs}"

#
env datetime2="${datetime}" message2="${message}"

echo "---"
printenv

echo "*** 🎉 env test completed ***"
###