package Wrapper

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var logger = log.WithFields(log.Fields{})

var WorkDir string

func Run(args []string) {

	/*os.MkdirAll("prof", os.ModePerm)
	var filename = fmt.Sprintf("prof/mem.prof.%d", os.Getpid())
	var profFile, _ = os.Create(filename)
	pprof.StartCPUProfile(profFile)
	*/

	var rustcArgsOnly = args[1:]

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		logger.Fatalln("No `FORKLIFT_WORK_DIR` specified!")
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
		log.Debugf("unknown log level (verbose) `%s`, using default `info`\n", Lib.AppConfig.General.LogLevel)
	}

	log.SetLevel(logLevel)

	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &rustcArgsOnly)

	var logger = wrapperTool.Logger

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	//var cachePackageName = CacheStorage.CreateCachePackageName(crateName, crateHash, outDir, compressor.GetKey())

	//logger.Tracef("wrapper args: %s\n", os.Args)
	var flClient = Rpc.NewForkliftRpcClient()

	//check deps
	var deps = Rustc.GetExternDeps(&rustcArgsOnly)
	var rebuiltDep = flClient.CheckExternDeps(deps)
	var gotRebuildDeps = true
	if rebuiltDep == "" {
		gotRebuildDeps = false
	}

	if gotRebuildDeps {
		logger.Debugf("Got rebuilt dep: %s", rebuiltDep)
	} else {
		logger.Debugf("No rebuilt deps")
	}

	var useCache = false

	// calc sources checksum
	if wrapperTool.IsNeedProcessFromCache() {
		useCache = calcChecksum2(wrapperTool)
	}

	// try get from cache
	if useCache && wrapperTool.IsNeedProcessFromCache() && !gotRebuildDeps {

		//var _, existsInStore = store.GetMetadata(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())

		/*
			var needDownload = true

			if !existsInStore {
				logger.Debugf("%s does not exist in storage\n", wrapperTool.GetCachePackageName())
				needDownload = false
			}*/

		//if needDownload {
		var f = store.Download(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())
		if f != nil {
			Tar.UnPack(WorkDir, compressor.Decompress(f))
			logger.Debugf("Downloaded artifacts for %s\n", wrapperTool.GetCachePackageName())

			io.Copy(os.Stderr, wrapperTool.ReadStderrFile())
			//io.Copy(os.Stdout, wrapperTool.ReadIOStreamFile("stdout"))

			//pprof.StopCPUProfile()
			//profFile.Close()
			return
		}
		//}
	} else {
		logger.Debugf("No need to use cache for %s", wrapperTool.OutDir)
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

	if !root && hasCargoToml(path) {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			checksum(filepath.Join(path, entry.Name()), hash, false)
		} else /*if strings.HasSuffix(entry.Name(), ".rs")*/ {
			var file, _ = os.Open(filepath.Join(path, entry.Name()))
			io.Copy(hash, file)
			file.Close()
		}
	}
}

func calcChecksum(wrapperTool *Rustc.WrapperTool) bool {
	var logger = wrapperTool.Logger
	var depInfoOnlyCommand = Rustc.CreateDepInfoCommand(&os.Args)

	logger.Debugf("depInfoOnlyCommand: %s", depInfoOnlyCommand)
	depInfoCmd := exec.Command(depInfoOnlyCommand[1], depInfoOnlyCommand[2:]...)
	//depInfoCmd.Env = []string{}

	var rustFlags, _ = os.LookupEnv("RUSTFLAGS")
	log.Infof("RUSTFLAGS %s", rustFlags)

	rustFlags, _ = os.LookupEnv("CARGO_ENCODED_RUSTFLAGS")
	log.Infof("CARGO_ENCODED_RUSTFLAGS %s", rustFlags)

	rustFlags, _ = os.LookupEnv("WASM_BUILD_RUSTFLAGS")
	log.Infof("WASM_BUILD_RUSTFLAGS %s", rustFlags)

	var depInfoStderr = bytes.Buffer{}
	//var multiWriter

	depInfoCmd.Stderr = &depInfoStderr
	err := depInfoCmd.Run()
	if err != nil {
		logger.Debugf("%s, %s, %s", err, string(depInfoStderr.Bytes()), depInfoOnlyCommand)
		return false
	}

	logger.Debugf("depInfoStderr: %s", depInfoStderr.String())

	artifact, err := Rustc.GetDepArtifact(&depInfoStderr)
	if err != nil {
		logger.Debugf("%s, %s, %s", err, string(depInfoStderr.Bytes()), depInfoOnlyCommand)
		return false
	}

	var files = Rustc.GetSourceFiles(artifact.Artifact)
	var checksum = FileManager.GetCheckSum(files, WorkDir)

	wrapperTool.CrateSourceChecksum = checksum
	logger.Debugf("Checksum: %s", checksum)
	return true
}
