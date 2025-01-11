#!/bin/bash
set -x
docker compose run  --rm -it --user "$(id -u):$(id -g)" --volume "/tmp/workspace":/app aider --chat-mode code --message "${1:-"tell me a joke"}"
##