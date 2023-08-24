## Forklift
Rust cache solution

### Usage
To operate forklift needs `items.cache` file in project root directory 

```
...
anyhow | 1.0.70 | 3972830907337473613 | 372254a9488afe4d
base64 | 0.21.0 | 11577549004696459789 | a0abb77ab29c1a0d
bytesize | 1.2.0 | 3307315851809781629 | 2de5f23e8b23f77d
cfg-if | 1.0.0 | 10051630247118670300 | 8b7e9074351489dc
...
```

You can generate this file with [modified cargo](https://gitlab.parity.io/parity/infrastructure/ci_cd/forklift/cargo-modified)

```shell
rust-simple % ../cargo/target/release/cargo prebuild > items.cache 
```

then you may use these commands
- `forklift push` - upload artifacts with hashes from `items.cache` to storage. Will search in `deps`, `build` adn `.fingerptint` folders
- `forklift pull` - download artifacts

Rignt now forklift pull/push artifacts **only from target/debug folder**

### Storage

Currently forklift uses only S3 as cache storage. You must provide this environment variables:

- S3_ACCESS_KEY_ID
- S3_SECRET_ACCESS_KEY
- S3_ENDPOINT_URL
- S3_BUCKET_NAME


### Example

```shell
parity@Evgenys-MacBook-Pro rust-simple % export S3_ENDPOINT_URL=https://storage.googleapis.com
parity@Evgenys-MacBook-Pro rust-simple % export S3_BUCKET_NAME=forklift-cache-parity-build-stg
parity@Evgenys-MacBook-Pro rust-simple % ../cargo/target/release/cargo prebuild > items.cache
parity@Evgenys-MacBook-Pro rust-simple % cargo build                                         
   Compiling base64 v0.21.2
   Compiling rust-simple v0.1.0 (/Users/parity/sources/repos/rust-simple)
    Finished dev [unoptimized + debuginfo] target(s) in 0.65s
parity@Evgenys-MacBook-Pro rust-simple % ./forklift push
No entries from 'build' for base64 6d04f85381ff6fe7
No entries from 'build' for rust-simple 1160ba424447b3ee
Uploaded 19 entries from 'deps' for base64 6d04f85381ff6fe7
Uploaded 8 entries from 'deps' for rust-simple 1160ba424447b3ee
Uploaded 1 entries from '.fingerprint' for base64 6d04f85381ff6fe7
Uploaded 1 entries from '.fingerprint' for rust-simple 1160ba424447b3ee
parity@Evgenys-MacBook-Pro rust-simple % rm -rf target/debug 
parity@Evgenys-MacBook-Pro rust-simple % ./forklift pull    
Downloaded artifacts for base64 6d04f85381ff6fe7 1 / 2
Downloaded artifacts for rust-simple 1160ba424447b3ee 2 / 2
parity@Evgenys-MacBook-Pro rust-simple % cargo build
    Finished dev [unoptimized + debuginfo] target(s) in 0.03s
parity@Evgenys-MacBook-Pro rust-simple % 
```

### TODO
- compress uploads
- optimize uploads
- normal cli
- internal crates caching
- logging and exception handling