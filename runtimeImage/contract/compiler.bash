rm target/wasm32-unknown-unknown/release/ogq.wasm
RUSTFLAGS="-C link-arg=-zstack-size=32768" cargo build --release --target wasm32-unknown-unknown
cp target/wasm32-unknown-unknown/release/ogq.wasm .
ontio-wasm-build ogq.wasm ogq.wasm
python3 wasm2binary.py
