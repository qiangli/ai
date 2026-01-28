#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script
# set -ue

message="tell me a joke"

# /sh:set_envs --option query="$message"
# fix me
# datetime=$(date)
datetime="Jan  1 08:08:08"

# map
map="
{
\"datetime\":\"$datetime\",
\"message\":\"$message\"
}
"
/sh:set_envs --option envs="${map}"

# array
array="[HOME=$HOME,PATH=${PATH}]"
/sh:set_envs --option envs="${array}"

# name=value
env datetime2="${datetime}" message2="${message}"

echo "---"
printenv

echo "*** 🎉 env test completed ***"
###