# Forklift
Cargo caching solution

Forklift acts as a `rustc wrapper` and decides whether to use cache based on `rustc` arguments it receives.


## Usage

1. Create `.forklift/config.toml` in cargo project root
   or in home folder (see config-example.toml)
2. Run cargo through forklift
   ```shell
   forklift cargo build --release ...
   ```


## How it works
1. Forklift runs its own "server", prepare cargo command, passes all necessary
   parameters and executes it.
   "Server" helps to calculate if dependencies have been rebuilt
   (for this forklift/cargo invocation) and uploads cache packages to the storage.

2. cargo calls wrapper with rustc arguments for each crate which it
   wants to build (artifacts are not presented in target dir), e.g.:

3. Wrapper collects dependencies (`--extern name=path`) and sends them to the
   server to check if any of these dependencies have already been rebuilt in this "session".

4. Wrapper calculates cache key from rustc args (already includes cargo hash),
   output path, source files hash, and dependencies artifacts hash

5. **if there are any rebuilt deps** or cache package is not presented in the storage,
   forklift will run the given rust c command, collect its output and send 
   it to the server for upload

   **if there are no rebuilt deps** and cache package is presented in the storage,
   forklift will download package and emit output saved from original rustc run imitating
   successful rustc execution.


## Configuration

### Config file

Forklift uses toml configuration files and will look for `.forklift/config.toml`
in user home folder and in the current working directory.
If both config files exist, they will be merged, with the local file
taking precedence

### Environment variables

Config may be overridden with environment variables like
`FORKLIFT_<full.key.path>`

For example, if you want to override `storage.s3.bucketName`
from config
```toml
#config.toml
[storage]
type = "s3"

[storage.s3]
accessKeyId = "ABCDEF1234567890"
bucketName = "forklift"
endpointUrl = "https://storage.googleapis.com"
secretAccessKey = "very_secret_key"
useSSL = true
```
you can set the `FORKLIFT_storage_s3_bucketName` variable
(Note that key path is case-sensitive)

### CLI

You can override any config value with commands:
```shell
% forklift config get storage.fs.directory
/forklift_storage/cargo

% forklift config set metrics.enabled true bool
previous metrics.enabled: false
new metrics.enabled: true
```

See `forklift config --help` for more info

### Structure

#### general
`[general]`

| key           | type   | description                                                                 |
|---------------|--------|-----------------------------------------------------------------------------|
| logLevel      | string | log level, `trace`,`debug`,`info`,`warn`,`error`, `fatal`, default `info`   |
| packageSuffix | string | suffix for all cache package names, default `""`                            |
| threadsCount  | int    | number of threads for parallel uploads, default `2`                         |
| quiet         | bool   | do not print any output, `logLevel` will be set to `fatal`, default `false` |

#### cache
`[cache]`

| key      | type     | description                                                      |
|----------|----------|------------------------------------------------------------------|
| extraEnv | []string | Additional environment variable names for cache key calculations |

#### storage
`[storage]`

| key  | type   | description                                      |
|------|--------|--------------------------------------------------|
| type | string | storage type, `s3`, `fs`, `null`, default `null` |

- **s3** - Amazon S3 with github.com/aws/aws-sdk-go
- **fs** - store cache on local file system
- **null** - doing nothing, just for testing

`[storage.s3]`

| key             | type   | description                             |
|-----------------|--------|-----------------------------------------|
| accessKeyId     | string | s3 access key id                        |
| bucketName      | string | s3 bucket name                          |
| endpointUrl     | string | s3 endpoint url                         |
| secretAccessKey | string | s3 secret access key                    |
| useSSL          | bool   | use ssl, default `true`                 |


`[storage.fs]`

| key       | type   | description       |
|-----------|--------|-------------------|
| directory | string | absolute dir path |

### compression
`[compression]`

| key  | type   | description                                               |
|------|--------|-----------------------------------------------------------|
| type | string | compression type, `none`, `lzma2`, `zstd`, default `none` |

- **none** - do not use compression
- **lzma2** - use [jamespfennell/xz go library](https://github.com/jamespfennell/xz)
- **zstd** - zstandart, use [github.com/klauspost/compress/zstd](https://github.com/klauspost/compress/tree/master/zstd)

`[compression.zstd]`

| key              | type | description                                |
|------------------|------|--------------------------------------------|
| compressionLevel | int  | compression level, `1` or `3`, default `3` |

`[compression.lzma2]`

| key              | type | description                           |
|------------------|------|---------------------------------------|
| compressionLevel | int  | compression level, `0-9`, default `6` |

### metrics
`[metrics]`

| key          | type              | description                            |
|--------------|-------------------|----------------------------------------|
| enabled      | bool              | enable metrics, default `false`        |
| pushEndpoint | string            | prometheus remote write endpoint       |
| extraLabels  | map[string]string | extra labels for metrics, default `{}` |

## Example

`none` compression, `fs` storage, `10` uploader threads


First run
```shell
% forklift cargo build --release
   Server info: Uploader threads: 10 
   Compiling libc v0.2.150
   Compiling proc-macro2 v1.0.70
   Compiling unicode-ident v1.0.12
   ............
   Wrapper info: Executing rustc, crate: 'cargo' hash: 'b778cdb1a5889e85'
    Finished release [optimized] target(s) in 58.41s
```

Second run
```shell
% forklift cargo build --release
   Server info: Uploader threads: 10
   Compiling libc v0.2.150
   Compiling proc-macro2 v1.0.70
   Compiling unicode-ident v1.0.12
   Compiling cfg-if v1.0.0
   Compiling autocfg v1.1.0
   Compiling pkg-config v0.3.27
   Compiling vcpkg v0.2.15
   Compiling memchr v2.6.4
   Compiling thiserror v1.0.50
   Wrapper info: Downloaded and unpacked artifacts for autocfg_c15112a65e15df34352675fecb8a9b766f7b414a, crate: 'autocfg' hash: 'd9aedd06ecfbfc27'
   ............      
   Wrapper info: Downloaded and unpacked artifacts for cargo_e722267ee4ebd91be7690a276f4a08fb857c5cf4, crate: 'cargo' hash: 'b778cdb1a5889e85'
    Finished release [optimized] target(s) in 11.46s                     
```