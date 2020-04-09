#!/usr/bin/env bash
go build config_server.go
docker build -f Dockerfile -t carltraveler/witness_config:$1 .
