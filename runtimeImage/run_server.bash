#!/usr/bin/env bash
set -ex

enterworkdir=$(pwd)
prefixworkdir="/data/"
export PATH="$HOME/.cargo/bin:$PATH"
which rustup || {
	echo "rustup not install"
	exit 1
}
which ontio-wasm-build || {
	echo "ontio-wasm-build not install"
	exit 1
}

[[ -f $prefixworkdir/config.json ]] || { 
	echo "config.json should be set by." && exit 1 
}

./confighandle

[[ $? == 0 ]] && {
	cp $enterworkdir/wallet.dat $prefixworkdir/
	cd $prefixworkdir
	echo "123456" | ./witness_server -l 2 -c config.run.json
}
