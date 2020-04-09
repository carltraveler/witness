#!/usr/bin/env bash
#build first
preparedir="./images/"
mkdir -p $preparedir
mkdir -p $preparedir/contract/src
go build confighandle.go
cd witness_server; go build witness_server.go rpc.go; mv witness_server witness_server_daemon;cd -

cp confighandle $preparedir
cp witness_server/witness_server_daemon $preparedir

cp config.fixed.json $preparedir
cp newcontract.bash $preparedir
cp run_server.bash $preparedir
cp wallet.dat $preparedir
cp contract/Cargo.toml $preparedir/contract
cp contract/src/lib.rs $preparedir/contract/src
cp config.json $preparedir/
