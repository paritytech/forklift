# Forklift
Rust caching solution

---
## Usage
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
- `forklift push --storage s3 ... [cargo_project_dir]` - upload artifacts with hashes from `items.cache` to storage.
Will search in `deps`, `build` and `.fingerptint` folders
- `forklift pull --storage s3 ... [cargo_project_dir]` - download artifacts

for more info see `forklift help`

---
## Storage

You may specify storage with `-s` or `--storage` flag. Additional parameters fo storage could be provided with `-p`

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

**!Important, right now you must use the same compression as for push and pull**

Supported options:

- None (`--compression none`) default, do not use compression 
- LZMA (`--compression xz`) use [ulikunitz/xz go library](https://github.com/ulikunitz/xz)
- LZMA2C (`--compression lzma2`) use [jamespfennell/xz go library](https://github.com/jamespfennell/xz), accepts `-p LZMA2_COMPRESSION_LEVEL=<0-9>`, default 6  
  **!!! Due to CGO requirements, `-c lzma2` is supported only in `linux_*` artifact from release page !!!**
- zstandart (`--compression zstd`) use [github.com/klauspost/compress/zstd](https://github.com/klauspost/compress/tree/master/zstd), accepts `-p ZSTD_COMPRESSION_LEVEL=<1|3>`, default 3

---
## Example

```shell
parity@Evgenys-MacBook-Pro rust-simple % ../cargo/target/release/cargo prebuild > items.cache
parity@Evgenys-MacBook-Pro rust-simple % cargo build                                         
   Compiling base64 v0.21.2
   Compiling rust-simple v0.1.0 (/Users/parity/sources/repos/rust-simple)
    Finished dev [unoptimized + debuginfo] target(s) in 0.65s
parity@Evgenys-MacBook-Pro rust-simple % forklift pull \
      -p S3_ACCESS_KEY_ID=... \
      -p S3_SECRET_ACCESS_KEY=... \
      -p S3_ENDPOINT_URL=192.168.1.2:9000 \
      -p S3_BUCKET_NAME=forklift \
      -c xz
No entries from 'build' for base64 6d04f85381ff6fe7
No entries from 'build' for rust-simple 1160ba424447b3ee
Uploaded 19 entries from 'deps' for base64 6d04f85381ff6fe7
Uploaded 8 entries from 'deps' for rust-simple 1160ba424447b3ee
Uploaded 1 entries from '.fingerprint' for base64 6d04f85381ff6fe7
Uploaded 1 entries from '.fingerprint' for rust-simple 1160ba424447b3ee
parity@Evgenys-MacBook-Pro rust-simple % rm -rf target/debug 
parity@Evgenys-MacBook-Pro rust-simple % forklift pull \
      -p S3_ACCESS_KEY_ID=... \
      -p S3_SECRET_ACCESS_KEY=... \
      -p S3_ENDPOINT_URL=192.168.1.2:9000 \
      -p S3_BUCKET_NAME=forklift \
      -c xz   
Downloaded artifacts for base64 6d04f85381ff6fe7 1 / 2
Downloaded artifacts for rust-simple 1160ba424447b3ee 2 / 2
parity@Evgenys-MacBook-Pro rust-simple % cargo build
    Finished dev [unoptimized + debuginfo] target(s) in 0.03s
parity@Evgenys-MacBook-Pro rust-simple % 
```

---
## TODO
- optimize uploads
- internal crates caching