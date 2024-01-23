package Wrapper

import (
	"bytes"
	"errors"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Logging"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	"forklift/Rpc/Models"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var WorkDir string

func Run(args []string) {

	var rustcArgsOnly = args[1:]

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		log.Fatalln("No `FORKLIFT_WORK_DIR` specified!")
		return
	}

	WorkDir = wd

	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &rustcArgsOnly)

	logger := Logging.CreateLogger("Wrapper", 1, log.Fields{
		"crate": wrapperTool.CrateName,
		"hash":  wrapperTool.CrateHash,
	})
	wrapperTool.Logger = logger

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	var flClient = Rpc.NewForkliftRpcClient()

	//check deps
	var deps = Rustc.GetExternDeps(&rustcArgsOnly, true)
	var rebuiltDep = flClient.CheckExternDeps(deps)
	var gotRebuildDeps = true
	if rebuiltDep == "" {
		gotRebuildDeps = false
	}

	if gotRebuildDeps {
		logger.Debugf("Got rebuilt dep: %s", rebuiltDep)
		flClient.ReportStatus(wrapperTool.CrateName, Models.DependencyRebuilt)
	} else {
		logger.Debugf("No rebuilt deps")
	}

	// calc sources checksum
	if wrapperTool.IsNeedProcessFromCache() {
		calcChecksum2(wrapperTool)
	}

	// try get from cache
	if wrapperTool.IsNeedProcessFromCache() && !gotRebuildDeps {

		var retries = 3

		for retries > 0 {
			f, err := store.Download(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())
			if f == nil && err == nil {
				logger.Debugf("%s does not exist in storage", wrapperTool.GetCachePackageName())
				flClient.ReportStatus(wrapperTool.CrateName, Models.CacheMiss)
				break
			}
			if err != nil {
				logger.Warningf("download error: %s", err)
				retries--
				continue
			}

			decompressed, err := compressor.Decompress(f)
			if err != nil {
				logger.Warningf("decompression error: %s", err)
				retries--
				continue
			}

			err = Tar.UnPack(WorkDir, decompressed)
			if err != nil {
				logger.Warningf("unpack error: %s", err)
				retries--
				continue
			} else {
				logger.Infof("Downloaded and unpacked artifacts for %s", wrapperTool.GetCachePackageName())

				io.Copy(os.Stderr, wrapperTool.ReadStderrFile())

				if retries == 3 {
					flClient.ReportStatus(wrapperTool.CrateName, Models.CacheUsed)
				} else {
					flClient.ReportStatus(wrapperTool.CrateName, Models.CacheUsedWithRetry)
				}

				return
			}
		}
		if retries <= 0 {
			logger.Errorf("Failed to pull artifacts for %s", wrapperTool.GetCachePackageName())
			flClient.ReportStatus(wrapperTool.CrateName, Models.CacheFetchFailed)
		}

	} else {
		logger.Debugf("No need to use cache for %s", wrapperTool.CrateName)
	}

	// execute rustc
	logger.Infof("Executing rustc")
	var artifacts, rustcError = ExecuteRustc(wrapperTool)

	if rustcError != nil {
		logger.Errorf("Rustc finished with error: %s", rustcError)
		var exitError *exec.ExitError
		if errors.As(rustcError, &exitError) {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
	logger.Debugf("Finished rustc")

	// register rebuilt artifacts
	RegisterRebuiltArtifacts(artifacts, flClient)

	if wrapperTool.IsNeedProcessFromCache() {
		flClient.AddUpload(wrapperTool.ToCacheItem())
	}
}

// ExecuteRustc - execute rustc and process output
func ExecuteRustc(wrapperTool *Rustc.WrapperTool) (*[]CacheStorage.RustcArtifact, error) {

	cmd := exec.Command(os.Args[1], os.Args[2:]...)

	var (
		rustcStdout = bytes.Buffer{}
		rustcStderr = bytes.Buffer{}
		rustcStdin  = bytes.Buffer{}
	)

	cmd.Stdout = io.MultiWriter(os.Stdout, &rustcStdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &rustcStderr)

	rustcStdin2 := bytes.Buffer{}
	stdinWriter := io.MultiWriter(&rustcStdin2, &rustcStdin)
	cmd.Stdin = &rustcStdin
	io.Copy(stdinWriter, os.Stdin)

	var runErr = cmd.Run()
	if runErr != nil {
		return nil, runErr
	}

	wrapperTool.WriteIOStreamFile(&rustcStdout, "stdout")
	artifacts := wrapperTool.WriteStderrFile(&rustcStderr)
	wrapperTool.WriteIOStreamFile(&rustcStdin2, "stdin")

	return artifacts, nil
}

// RegisterRebuiltArtifacts - register rebuilt artifacts
func RegisterRebuiltArtifacts(artifacts *[]CacheStorage.RustcArtifact, flClient *Rpc.ForkliftRpcClient) {
	var artifactsPaths = make([]string, 0)
	for _, artifact := range *artifacts {
		var abs = filepath.Base(artifact.Artifact)

		artifactsPaths = append(artifactsPaths, abs)
	}
	flClient.RegisterExternDeps(&artifactsPaths)
}
