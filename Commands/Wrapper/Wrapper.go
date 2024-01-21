package Wrapper

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Logging"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	"forklift/Rpc/Models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"hash"
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

	err := viper.Unmarshal(&Lib.AppConfig)
	if err != nil {
		log.Errorln(err)
	}

	logLevel, err := log.ParseLevel(Lib.AppConfig.General.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		log.Debugf("unknown log level (verbose) `%s`, using default `info`", Lib.AppConfig.General.LogLevel)
	}
	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &rustcArgsOnly)

	var l = log.Logger{
		Out:       os.Stderr,
		Formatter: &Logging.ForkliftTextFormatter{Indentation: 1, TaskPrefix: "Wrapper"},
		Level:     logLevel,
	}

	var logger = l.WithFields(log.Fields{
		"crate": wrapperTool.CrateName,
		"hash":  wrapperTool.CrateHash,
	})
	wrapperTool.Logger = logger

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	//var cachePackageName = CacheStorage.CreateCachePackageName(crateName, crateHash, outDir, compressor.GetKey())

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
				flClient.ReportStatus(wrapperTool.CrateName, Models.CacheUsed)

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
	logger.Debugf("executing rustc")

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

	wrapperTool.WriteIOStreamFile(&rustcStdout, "stdout")
	artifacts := wrapperTool.WriteStderrFile(&rustcStderr)
	wrapperTool.WriteIOStreamFile(&rustcStdin2, "stdin")

	// register rebuilt artifacts path
	var artifactsPaths = make([]string, 0)
	for _, artifact := range *artifacts {
		var abs = filepath.Base(artifact.Artifact)

		artifactsPaths = append(artifactsPaths, abs)
	}
	flClient.RegisterExternDeps(&artifactsPaths)

	if runErr != nil {
		if serr, ok := err.(*exec.ExitError); ok {
			os.Exit(serr.ExitCode())
		}
		os.Exit(1)
	}

	logger.Debugf("Finished rustc")

	if wrapperTool.IsNeedProcessFromCache() {
		flClient.AddUpload(wrapperTool.ToCacheItem())
		//wrapperTool.WriteToItemCacheFile()
	}
}

func hasCargoToml(path string) bool {
	var cargoTomls, err = filepath.Glob(filepath.Join(path, "Cargo.toml"))
	if err != nil {
		log.Panicf("Error: %s", err)
	}
	//log.Debugf("cargo tomls: %d", len(cargoTomls))
	return len(cargoTomls) > 0
}

func calcChecksum2(wrapperTool *Rustc.WrapperTool) bool {

	var path = wrapperTool.SourceFile

	path = filepath.Dir(path)

	for {
		if hasCargoToml(path) {
			break
		} else {
			path = filepath.Dir(path)
		}
	}

	wrapperTool.Logger.Tracef("Cargo.toml found in %s", path)

	var sha = sha1.New()
	checksum(path, sha, true)
	wrapperTool.CrateSourceChecksum = fmt.Sprintf("%x", sha.Sum(nil))
	return true
}

func checksum(path string, hash hash.Hash, root bool) {
	var entries, _ = os.ReadDir(path)

	/*if !root && hasCargoToml(path) {
		return
	}*/

	for _, entry := range entries {

		if root && needIgnore(entry.Name()) {
			continue
		}

		if entry.IsDir() {
			checksum(filepath.Join(path, entry.Name()), hash, false)
		} else {
			var file, _ = os.Open(filepath.Join(path, entry.Name()))
			io.Copy(hash, file)
			file.Close()
		}
	}
}

// needIgnore returns true if entryName should be ignored
func needIgnore(entryName string) bool {
	var ignorePatterns = []string{
		".git",
		".idea",
		".vscode",
		".cargo",
		"target",
		".forklift",
	}

	for _, pattern := range ignorePatterns {
		if pattern == entryName {
			return true
		}
	}

	return false
}
