#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

echo "Bash basic extension tests"

set -xue

##
/bin/ls -al /tmp

echo ">>> Testing bash extnesions..."

#
/sh:help
/sh:pass

# system command
/sh:exec --command "ls -al /tmp"

# /sh:cd
/sh:pwd

/sh:get_envs
/sh:set_envs --option name=value
/sh:unset_envs --option keys="[name]"

# tool
/atm:hi --script "$(pwd)/swarm/resource/template/atm.yaml" \
     --option greeting="how are you today?" \
     --option names='["dragon", "horse"]'

# agent
# /agent:atm/hi --script "$(pwd)/swarm/resource/template/atm.yaml" \
#     --adapter echo

# alias
/alias:cmd --option cmd="ls -al /tmp"

# scripting
/sh:bash --script "data:,ls -al /tmp"
/sh:bash --script "test/sb.sh"
# TODO support subshell or print file content
# /sh:bash --script "data:,$(cat $BASE/test/sb.sh)"

# flow
/flow:sequence --actions '["sh:pass","sh:pwd"]'
/flow:choice --actions '[sh:pwd,sh:pass]'
/flow:parallel --actions '[sh:pwd,sh:pass]'
/flow:chain --option chain=[sh:timeout,sh:backoff,alias:cmd] \
    --option cmd="ls -al /tmp" \
    --option duration="60s"

echo "*** 🎉 Bash extention tests completed ***"
###