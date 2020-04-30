#!/usr/bin/env bash
set -ex

prefixworkdir="/data/"
appconfigdir="/appconfig/"
contractdir="/wasm/"
[[ $1 != "" ]] && prefixworkdir=$1
[[ $2 != "" ]] && appconfigdir=$2
[[ $2 != "" ]] && contractdir=$3

[[ -f $appconfigdir/config.json ]] || { 
	echo "config.json should be set by." 
	exit 1 
}

[[ -f $contractdir/contract.wasm ]] || { 
	echo "wasm file not found" 
	exit 1 
}

[[ -f config.fixed.json ]] || { 
	echo "config.fixed.json should be set by." 
	exit 1 
}

[[ -w $prefixworkdir ]] || {
	echo "$prefixworkdir no write access."
	exit 1
}

echo "depoy. init. and generate config.run.json."
echo "123456" | ./confighandle -runPath $prefixworkdir -configPath $appconfigdir -contractPath $contractdir

[[ $? == 0 ]] && {
	echo "config success. start witness_server_daemon"
	# all save in docker
	cp ./wallet.dat $prefixworkdir/
	cp ./witness_server_daemon $prefixworkdir
	cd $prefixworkdir
	echo "123456" | ./witness_server_daemon -l 2 --correctdatabase 2 -c config.run.json
}

echo "config failed. or server exit"
exit 1
