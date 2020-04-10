#!/usr/bin/env bash
set -ex

[[ $1 == "" ]] && exit 1
dir=$(pwd)
cd contract

sed -i "s/APHNPLz2u1JUXyD8rhryLaoQrW46J3P6y2/$1/g" src/lib.rs
echo "start build rust contract."
RUSTFLAGS="-C link-arg=-zstack-size=32768" cargo build --release --target wasm32-unknown-unknown
echo "build rust contract success."
which ontio-wasm-build || cargo install --git=https://github.com/ontio/ontio-wasm-build

cd $dir
ontio-wasm-build contract/target/wasm32-unknown-unknown/release/ogq.wasm contract.wasm
echo "contract prepared over"
