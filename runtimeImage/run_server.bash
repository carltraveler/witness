#!/usr/bin/env bash
set -ex

prefixworkdir="./run/"
witnessdir="witness" 
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

[[ -d $witnessdir ]] || git clone https://github.com/carltraveler/witness

{
	cd $witnessdir/runtimeImage/;
	cp witness_server_daemon $prefixworkdir;
	echo "prepare witness_server_daemon done."
	[[ -f $prefixworkdir/config.json ]] || { 
		echo "config.json should be set by." && exit 1 
	}
	./confighandle -runPath $prefixworkdir
	cd -
}

[[ $? == 0 ]] && {
	cp ./wallet.dat $prefixworkdir/
	cp $witnessdir/runtimeImage/witness_server_daemon $prefixworkdir
	cd $prefixworkdir
	echo "123456" | ./witness_server_daemon -l 2 -c config.run.json
}

echo "config failed."
