#!/usr/bin/env bash
set -ex

[[ $1 == "" ]] && exit 1
dir=$(pwd)
cd contract

# save old file to restart.
cp src/lib.rs ../lib.rs.bake
sed -i "s/Ab1z3Sxy7ovn4AuScdmMh4PRMvcwCMzSNV/$1/g" src/lib.rs
echo "start build rust contract."
RUSTFLAGS="-C link-arg=-zstack-size=32768" cargo build --release --target wasm32-unknown-unknown
# restore old file.
cp ../lib.rs.bake src/lib.rs
echo "build rust contract success."
which ontio-wasm-build || cargo install --git=https://github.com/ontio/ontio-wasm-build

cd $dir
ontio-wasm-build contract/target/wasm32-unknown-unknown/release/ogq.wasm contract.wasm
echo "contract prepared over"
