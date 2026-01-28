#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

/sh:set_envs --option envs="[OPENAI_API_KEY=invalid}]"

./swarm/resource/template/atm.yaml --base ./test/data/ --adapter echo $@