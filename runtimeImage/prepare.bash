#!/usr/bin/env bash
#build first
preparedir="./images/"
mkdir -p $preparedir
go build confighandle.go
cd witness_server; go build witness_server.go rpc.go; mv witness_server witness_server_daemon;cd -

cp confighandle $preparedir
cp witness_server/witness_server_daemon $preparedir

cp config.fixed.json $preparedir
cp run_server.bash $preparedir
cp wallet.dat $preparedir
cp config.json $preparedir/

## for test
#cd $preparedir
#mkdir appconfig wasm data
#cp ../contract.wasm wasm/
#cp ../config.json appconfig/

