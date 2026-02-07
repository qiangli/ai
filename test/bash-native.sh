#!/bin/bash
# this must be run with a system bash
cd /tmp && echo "cd is okay"

/sh:exec || echo "agent/tool extended feature is not supported"

echo "done"