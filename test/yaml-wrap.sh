#!/usr/bin/env ai /sh:bash --format raw --script

/sh:set_envs --option envs="[OPENAI_API_KEY=invalid}]"

./swarm/resource/template/atm.yaml --adapter echo $@