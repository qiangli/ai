#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

set -ue

BASE=$(pwd)
echo $BASE

/sh:set_envs --option envs="[OPENAI_API_KEY=invalid,GEMINI_API_KEY=invalid,ANTHROPIC_API_KEY=invalid,XAI_API_KEY=invalide}]"

##
# 
$BASE/test/bash-basic.sh
$BASE/test/bash-ext.sh

$BASE/test/flow.sh

##
adapter="--adapter echo --output none"
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

$BASE/test/std.sh || exit 1
$BASE/test/fs.sh

$BASE/test/env.sh

# ai bash examples
$BASE/test/example.sh

#
function test_failed() {
    echo "❌ test outputs differ"
    exit 1
}
$BASE/test/test.sh

# diff /tmp/test.txt ./test/data/test-expected.txt 
/sh:exec --command "diff /tmp/test.txt ./test/data/test-expected.txt || echo  '❌❌ test outputs differ'" 

echo "$?"
echo "*** All tests completed ***"
###

