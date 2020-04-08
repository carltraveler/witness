#!/usr/bin/env bash
set -ex

enterworkdir=$(pwd)
prefixworkdir="/data/"
witnessdir="witness"
which git || apt install git
echo "install git done."
which go || apt install golang
echo "install go done."
if ! which rustup ; then
	curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly 
fi
source $HOME/.cargo/env
echo "install rust done."
rustup target add wasm32-unknown-unknown
echo "add target wasm32 done."
which ontio-wasm-build || cargo install --git=https://github.com/ontio/ontio-wasm-build
echo "install ontio-wasm-build done."

[[ -d $witnessdir ]] || git clone https://github.com/carltraveler/witness
echo "install witness repo done."
cd $witnessdir/runtimeImage/witness_server; go build witness_server.go rpc.go; cp witness_server $prefixworkdir; cd -
echo "build witness_server done."
cd $witnessdir/runtimeImage; go build confighandle.go
echo "build witness confighandle done."

[[ -f $prefixworkdir/config.json ]] || { 
	echo "config.json should be set by." && exit 1 
}

./confighandle

[[ $? == 0 ]] && {
	cp $enterworkdir/wallet.dat $prefixworkdir/
	cd $prefixworkdir
	echo "123456" | ./witness_server -l 2 -c config.run.json
}
