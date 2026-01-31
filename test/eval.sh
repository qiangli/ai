#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

# set -x

# BASE=$(pwd)
# /bin/ls /x
# ls /x
# cd /
# exec test.sh
# /sh:fail
# /fs:list_roots
# ./test/env.sh
# /bin/bash ./test/bash-legacy.sh
# echo "status $?"

# map
map="
{
\"datetime\":\"$(date)\",
\"message\":\"test\"
}
"
/sh:set_envs --option envs="${map}"

echo "datetime: $datetime"
echo "message: $message"

# export name="charles"
# set -a; name=charles; nick=lee; set +a;
# echo "name: ${name} nick: $nick"
###