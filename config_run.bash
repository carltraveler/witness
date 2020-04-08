#!/usr/bin/env bash
set -ex

enterworkdir=$(pwd)
witnessdir="witness"
which git || apt install git
echo "install git done."
which go || apt install golang

[[ -d $witnessdir ]] || git clone https://github.com/carltraveler/witness
cd $witnessdir; go build config_server.go

./config_server
