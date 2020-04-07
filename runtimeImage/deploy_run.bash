#!/usr/bin/env bash
set -ex

./confighandle

[[ $? == 0 ]] && ./witness_server -l 2 -c config.run.json
