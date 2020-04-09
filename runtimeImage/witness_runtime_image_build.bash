#!/usr/bin/env bash
./prepare.bash
docker build -f Dockerfile -t docker build -f Dockerfile -t carltraveler/witness_runtimev0:$1 .
