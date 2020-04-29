#!/usr/bin/env bash
set -ex

prefixworkdir="/data/"
appconfigdir="/appconfig/"
[[ $1 != "" ]] && prefixworkdir=$1
[[ $2 != "" ]] && appconfigdir=$2
export PATH="$HOME/.cargo/bin:$PATH"
which rustup || {
	echo "rustup not install"
	exit 1
}
which ontio-wasm-build || {
	echo "ontio-wasm-build not install"
	exit 1
}

[[ -f $appconfigdir/config.json ]] || { 
	echo "config.json should be set by." 
	exit 1 
}

[[ -f config.fixed.json ]] || { 
	echo "config.fixed.json should be set by." 
	exit 1 
}

echo "generate config.run.json."
./confighandle -runPath $prefixworkdir -configPath $appconfigdir

[[ $? == 0 ]] && {
	echo "config success. start witness_server_daemon"
	# all save in docker
	cp ./wallet.dat $prefixworkdir/
	cp ./witness_server_daemon $prefixworkdir
	cd $prefixworkdir
	echo "123456" | ./witness_server_daemon -l 2 --correctdatabase 2 -c config.run.json
}

echo "config failed. or server exit"
