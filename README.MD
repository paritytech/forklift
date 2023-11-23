# Forklift
Cargo caching solution

"Forklift" acts as a `rustc wrapper` and decides whether to use cache based on `rustc` arguments it receives.


---
## Usage

1. Create `.forklift/config.toml` in cargo project root (see config-example.toml)
2. Run forklift server

- manually:
```shell
# required vars 
export RUSTC_WRAPPER="forklift"
export FORKLIFT_WORK_DIR="$(pwd)"
# start server
forklift start
cargo build --release ......
# stop server
forklift stop
# push artifacts to remote storage
forklift push
```
- or by all-in-one command (push included):
```shell 
forklift cargo build --release ......
```

---
## How it works

1. cargo invokes wrapper with rustc arguments for each crate which it
   intends to build (crates are not presented in target dir)
   ```shell
   forklift rustc --crate-name libssh2_sys /Users/parity/.cargo/registry/src/index.crates.io-6f17d22bba15001f/libssh2-sys-0.3.0/lib.rs 
      --error-format=json 
      --json=diagnostic-rendered-ansi,artifacts,future-incompat 
      --diagnostic-width=214 
      --crate-type lib 
      --emit=dep-info,metadata,link 
      -C opt-level=3 
      -C embed-bitcode=no 
      -C metadata=1c2087c8e4d22ed5 
      -C extra-filename=-1c2087c8e4d22ed5 
      --out-dir /Users/parity/sources/repos/cargo/target/release/deps 
      -L dependency=/Users/parity/sources/repos/cargo/target/release/deps 
      --extern libc=/Users/parity/sources/repos/cargo/target/release/deps/liblibc-d5976263134a2741.rmeta 
      --extern libz_sys=/Users/parity/sources/repos/cargo/target/release/deps/liblibz_sys-2c4d1c8ae9635bb1.rmeta 
      --extern openssl_sys=/Users/parity/sources/repos/cargo/target/release/deps/libopenssl_sys-4646fa0b007fe98d.rmeta 
      --cap-lints allow -L native=/Users/parity/sources/repos/cargo/target/release/build/libssh2-sys-700b9978bf455f50/out/build 
      -l static=ssh2 
      -L native=/opt/homebrew/opt/openssl@3/lib
   ```
2. wrapper calculates cache key from cargo hash (`-C metadata=1c2087c8e4d22ed5`),
   output directory hash and source files hash. Crate source files are
   collected by executing same rustc command but with `--emit=dep-info` only,
   which generates .d file with all source files that will be used
   for build.
3. Wrapper collects dependencies (`--extern name=path`) and sends them to the
   server to check if any of these dependencies have already been rebuilt in this "session".
4. - if there are any rebuilt deps or no cache package in the remote storage,

     forklift will run given rust c command, collect its output and save new cache
     package description.
   - if there are no rebuilt dependencies and cache package is presented in the storage,
   
     forklift will download package and emit output saved from original rustc run imitating
     successful rustc execution.
5. `forklift push` gather new cache descriptions, pack all necessary files and upload
   package to the storage

---
## Storage

You may specify storage with `-s, --storage` flag. Additional parameters for storage could be provided with `-p`

### S3 (`-s s3`, default)

S3 looks for parameters in params (`-p`), then in environment variables

Available parameters:
 - S3_ACCESS_KEY_ID
 - S3_SECRET_ACCESS_KEY
 - S3_USE_SSL (default `true`)
 - S3_ENDPOINT_URL
 - BUCKET_NAME, S3_BUCKET_NAME (default `forklift`)

example:

`forklift pull -s s3 -p S3_ACCESS_KEY_ID=<key_id> -p S3_SECRET_ACCESS_KEY=<>secret> -p S3_ENDPOINT_URL=192.168.1.2:9000 -p S3_BUCKET_NAME=forklift ../rust-simple`

### Google cloud storage (`-s gcs`)

Gcs looks for parameters in params (`-p`), then in environment variables

Available parameters:
- GCP_APPLICATION_CREDENTIALS - path to application credentials file
- GCP_CREDENTIALS_JSON_BASE64 - base64 decoded application credentials json data
- BUCKET_NAME, S3_BUCKET_NAME (default `forklift`)

example:

`forklift pull -s gcs -p GCP_APPLICATION_CREDENTIALS="/home/user/application_default_credentials.json" -p S3_BUCKET_NAME=forklift ../rust-simple`

### File system (-s fs)

Available parameters:
- FS_DIR absolute path

example:

`forklift push --storage fs -p FS_DIR=/Users/parity/sources/workbench/forklift_storage/cargo-none -c none ../crust-simple`

---
## Compression

Specify compression with `-c` or `--compression` flag

**!Important, right now you must use the same compression for push and pull**

Supported options:

- None (`--compression none`) default, do not use compression 
- LZMA (`--compression xz`) use [ulikunitz/xz go library](https://github.com/ulikunitz/xz)
- LZMA2C (`--compression lzma2`) use [jamespfennell/xz go library](https://github.com/jamespfennell/xz), accepts `-p LZMA2_COMPRESSION_LEVEL=<0-9>`, default 6  
  **!!! Due to CGO requirements, `-c lzma2` is supported only in `linux_*` artifact from release page !!!**
- zstandart (`--compression zstd`) use [github.com/klauspost/compress/zstd](https://github.com/klauspost/compress/tree/master/zstd), accepts `-p ZSTD_COMPRESSION_LEVEL=<1|3>`, default 3

---
## Example

First run
```shell
parity@MacBook-Pro rust-simple % ../forklift/forklift cargo build --release
   Compiling base64 v0.21.2
time="2023-11-23T14:59:44+04:00" level=info msg="No rebuilt deps" crate=base64 hash=f1da19b6179a58d5
time="2023-11-23T14:59:44+04:00" level=info msg="base64_f1da19b6179a58d5_da39a3ee5e6b4b0d3255bfef95601890afd80709_7704e991860a53554f1b814e05e8bd6a9c0d9fa7 does not exist in storage\n" crate=base64 hash=f1da19b6179a58d5
time="2023-11-23T14:59:44+04:00" level=info msg="executing rustc" crate=base64 hash=f1da19b6179a58d5
   Compiling rust-simple v0.1.0 (/Users/parity/sources/repos/rust-simple)
time="2023-11-23T14:59:44+04:00" level=info msg="Got rebuilt dep: /Users/parity/sources/repos/rust-simple/target/release/deps/libbase64-f1da19b6179a58d5.rlib" crate=rust_simple hash=0beaf3a97b4d073c
time="2023-11-23T14:59:44+04:00" level=info msg="No need to use cache for target/release/deps" crate=rust_simple hash=0beaf3a97b4d073c
time="2023-11-23T14:59:44+04:00" level=info msg="executing rustc" crate=rust_simple hash=0beaf3a97b4d073c
    Finished release [optimized] target(s) in 0.41s
INFO[0000] cacheItems 2                                 
INFO[0000] Uploaded rust_simple-0beaf3a97b4d073c, 4de709f84a1f8fc1df833563eb4484b99e145334 
INFO[0000] Uploaded base64-f1da19b6179a58d5, d5c717fb4cf0ac98fca640ec3bedddf826023fe4 
```

Second run
```shell
parity@MacBook-Pro rust-simple % ../forklift/forklift cargo build --release                          
   Compiling base64 v0.21.2
time="2023-11-23T15:02:14+04:00" level=info msg="No rebuilt deps" crate=base64 hash=f1da19b6179a58d5
time="2023-11-23T15:02:14+04:00" level=info msg="Downloaded artifacts for base64_f1da19b6179a58d5_da39a3ee5e6b4b0d3255bfef95601890afd80709_7704e991860a53554f1b814e05e8bd6a9c0d9fa7\n" crate=base64 hash=f1da19b6179a58d5
   Compiling rust-simple v0.1.0 (/Users/parity/sources/repos/rust-simple)
time="2023-11-23T15:02:14+04:00" level=info msg="No rebuilt deps" crate=rust_simple hash=0beaf3a97b4d073c
time="2023-11-23T15:02:14+04:00" level=info msg="Downloaded artifacts for rust_simple_0beaf3a97b4d073c_b1713adaf16c3f6c66a004116b83cd15d88d2fd6_7704e991860a53554f1b814e05e8bd6a9c0d9fa7\n" crate=rust_simple hash=0beaf3a97b4d073c
    Finished release [optimized] target(s) in 0.09s
INFO[0000] cacheItems 0                             
```

---
## TODO
- Optimize uploads
- Revise readme
- Refactoring