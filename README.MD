# Forklift – caching solution for Cargo

![License](https://img.shields.io/badge/license-MIT-green)
![Tests](https://github.com/paritytech/forklift/actions/workflows/test.yml/badge.svg)
![Release](https://img.shields.io/github/v/release/paritytech/forklift)

![Forklift](https://i.postimg.cc/MTCyBBLv/forklift-paritytech.png)

Forklift acts as a `rustc` wrapper and decides whether to use the cache based on the `rustc` arguments it receives.

## Usage

1. Create a `.forklift/config.toml` file in the Cargo project root or in the home folder. Refer to `config-example.toml` for an example.

2. Run Cargo through Forklift by executing the following command:
   ```shell
   forklift cargo build --release ...
   ```


## How it works

1. Forklift operates its own "server", preparing the Cargo command, passing all necessary parameters, and executing it. The "server" assists in determining whether dependencies have been rebuilt for this Forklift+Cargo invocation and uploads cache packages to the storage.

2. Cargo invokes the wrapper with `rustc` arguments for each crate it intends to build (if artifacts are not present in the target directory).

3. The wrapper gathers dependencies (`--extern name=path`) and forwards them to the server to verify if any of these dependencies have been rebuilt during this "session".

4. The wrapper computes the cache key from the `rustc` arguments (which already include the cargo hash), the output path, the hash of the source files, hash of the rustc binary, and the hash of the 'dependencies' artifacts.

5. **If there are any rebuilt dependencies** or the cache package is not present in the storage, Forklift will execute the specified `rustc` command, collect its output, and upload it to the server.

   **If there are no rebuilt dependencies** and the cache package is available in the storage, Forklift will download the package and emit the output saved from the original `rustc` run, imitating a successful `rustc` execution.


## Configuration

### Config file

Forklift utilizes TOML configuration files and searches for `.forklift/config.toml` in the user's home folder and the current working directory.
If both configuration files are present, they will be merged, with the local file taking precedence.

### Environment variables

Configuration settings can be overridden with environment variables formatted as `FORKLIFT_<full.key.path>`.

For example, if you want to override `storage.s3.bucketName` from the config:

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

You can also set the `FORKLIFT_storage_s3_bucketName` variable here. Note that the key path is case sensitive.

### CLI

You can override any configuration value using commands through CLI invocation:

```shell
% forklift config get storage.fs.directory
/forklift_storage/cargo

% forklift config set metrics.enabled true bool
previous metrics.enabled: false
new metrics.enabled: true
```

See `forklift config --help` for more info.

### Structure

#### general
`[general]`

| Key           | Type    | Description                                                                 |
|---------------|---------|-----------------------------------------------------------------------------|
| logLevel      | `string` | Specifies the log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal`. Default: `info`. |
| packageSuffix | `string` | Defines a suffix for all cache package names. Default: `""`.                |
| threadsCount  | `int`    | Sets the number of threads for parallel uploads. Default: `2`.              |
| quiet         | `bool`   | Suppresses all output. Setting this to `true` sets `logLevel` to `fatal`. Default: `false`. |

#### cache
`[cache]`

| key           | type              | description                                                      |
|---------------|-------------------|------------------------------------------------------------------|
| extraEnv      | []string          | Additional environment variable names for cache key calculations |
| extraMetadata | map[string]string | Additional fields to include in artifact metadata                |

#### storage
`[storage]`

| Key  | Type   | Description                                      |
|------|--------|--------------------------------------------------|
| type | `string` | Storage type: `s3`, `fs`, `null`. Default: `null`. |

- **s3** - Amazon S3 via github.com/aws/aws-sdk-go.
- **fs** - Stores cache on the local file system.
- **null** - Performs no action, intended for testing.

`[storage.s3]`

| Key             | Type   | Description                             |
|-----------------|--------|-----------------------------------------|
| accessKeyId     | `string` | S3 access key ID.                        |
| bucketName      | `string` | S3 bucket name.                          |
| endpointUrl     | `string` | S3 endpoint URL.                         |
| secretAccessKey | `string` | S3 secret access key.                    |
| useSSL          | `bool`   | Use SSL, default `true`.                 |


`[storage.fs]`

| Key       | Type   | Description         |
|-----------|--------|---------------------|
| directory | `string` | Absolute directory path |

### compression
`[compression]`

| Key  | Type   | Description                                               |
|------|--------|-----------------------------------------------------------|
| type | `string` | Compression type: `none`, `lzma2`, `zstd`. Default: `none`. |

- **none** - Do not use compression.
- **lzma2** - Use the [jamespfennell/xz](https://github.com/jamespfennell/xz).
- **zstd** - Use the [klauspost/compress/zstd](https://github.com/klauspost/compress/tree/master/zstd).

`[compression.zstd]`

| Key              | Type | Description                                |
|------------------|------|--------------------------------------------|
| compressionLevel | `int`  | Compression level: `1` or `3`, default `3`. |

`[compression.lzma2]`

| Key              | Type | Description                           |
|------------------|------|---------------------------------------|
| compressionLevel | `int`  | Compression level, `0-9`, default `6` |

### metrics
`[metrics]`

| Key          | Type                | Description                              |
|--------------|---------------------|------------------------------------------|
| enabled      | `bool`              | Enable metrics, default `false`.        |
| pushEndpoint | `string`            | Prometheus remote write endpoint.       |
| extraLabels  | `map[string]string` | Extra labels for metrics, default `{}`. |

## Example

The example was made with the following Forklift options:

`none` compression, `fs` storage, `10` uploader threads

##### First run (no cache)

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

##### Second run (cache is being reused)

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

## Metrics

Forklift provides the set of Prometheus-compatible metrics to monitor caching efficiency and performance. These metrics are available when metrics collection is enabled in the configuration.

Available metrics include:

| Name                                                 | Description                                                                | Labels                                                                                                                                                                                          |
|------------------------------------------------------|----------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `forklift_uploader_uploading_network_avg_speed`      | Average cache upload speed in bytes per second across all uploader threads |                                                                                                                                                                                                 |
| `forklift_uploader_uploading_network_uploaded_bytes` | Total bytes downloaded during cache upload                                 |                                                                                                                                                                                                 |
| `forklift_uploader_uploading_status`                 | Number of cache packages uploads (with status label)                       | `status`:<br/> -`success`: cache package uploaded successfully<br/> -`warning`: uploaded, but with retries <br/> -`fail`: failed to upload in 3 tries)                                          |
| `forklift_uploader_uploading_time_task`              | Time spent on specific tasks during cache upload process                   | `task`:<br/> -`pack`<br/> -`compress`<br/> -`upload`)                                                                                                                                           |
| `forklift_uploader_uploading_time_total`             | Total time spent by Forklift (includes Cargo execution)                    |                                                                                                                                                                                                 |
| `forklift_wrapper_caching_cache_hit`                 | Number of cache hits (with status label)                                   | `status`:<br/> -`hit`: cache was used  <br/> -`warning`: cache was used, but with retries                                                                                                       |
| `forklift_wrapper_caching_cache_miss`                | Number of cache misses (with status label)                                 | `status`:<br/> -`miss`: no cache artifacts were found in storage  <br/> -`fail`: failed to retrieve artifact in 3 tries<br/> -`dep_rebuilt`: crate had a rebuilt dependency, no cache were used |
| `forklift_wrapper_caching_network_avg_speed`         | Average cache download speed in bytes per second across all rustc_wrappers |                                                                                                                                                                                                 |
| `forklift_wrapper_caching_network_downloaded_bytes`  | Total bytes downloaded during cache restore                                |                                                                                                                                                                                                 |
| `forklift_wrapper_caching_time_task`                 | Time spent on specific tasks during cache restore process                  | `task`:<br/> -`download`,<br/> -`decompress`,<br/> -`unpack`,<br/> -`rustc`                                                                                                                     |
| `forklift_wrapper_caching_time_total`                | Total time spent by rustc_wrappers                                         |                                                                                                                                                                                                 |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
