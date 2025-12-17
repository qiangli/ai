#!/usr/bin/env ai /sh:bash --verbose --format raw --script
set -xue

actions='["ai:read_tool_config","sh:format"]'

/sh:flow --actions "$actions" --template "data:,{{toPrettyJson .config}}" --tool "fs:read_file" --format raw