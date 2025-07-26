#!/bin/bash

# sk-secret
token=${1:-"sk-secret"}
addr=${2:-"4000"}

curl --location --request POST "http://127.0.0.1:$addr/v1/chat/completions" \
--header "Authorization: $token" \
--header "Content-Type: application/json" \
--data-raw '{
    "messages": [
        {
            "role": "system",
            "content": "Hi"
        }
    ],
    "model": "gpt-4.1-mini",
    "stream": true
}'