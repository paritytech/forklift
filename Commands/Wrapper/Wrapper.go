package Wrapper

import (
	"bytes"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var logger = log.WithFields(log.Fields{})

var WorkDir string

func Run(args []string) {
	//var outBuf = bytes.Buffer{}
	//log.SetOutput(&outBuf)

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
		log.Infof("unknown log level (verbose) `%s`, using default `info`\n", Lib.AppConfig.General.LogLevel)
	}

	log.SetLevel(logLevel)

	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &args)

	var logger = wrapperTool.Logger

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	//var cachePackageName = CacheStorage.CreateCachePackageName(crateName, crateHash, outDir, compressor.GetKey())

	logger.Tracef("wrapper args: %s\n", os.Args)
	var flClient = Rpc.NewForkliftRpcClient()

	//check deps
	var deps = Rustc.GetExternDeps(&args)
	var gotRebuildDeps = flClient.CheckExternDeps(deps)

	if gotRebuildDeps {
		logger.Debugf("Got rebuilt deps")
	} else {
		logger.Debugf("No rebuilt deps")
	}

	// calc sources checksum
	if wrapperTool.CrateName != "___" && !gotRebuildDeps {
		var depInfoOnlyCommand = Rustc.CreateDepInfoCommand(&os.Args)

		depInfoCmd := exec.Command(depInfoOnlyCommand[1], depInfoOnlyCommand[2:]...)
		var depInfoStderr = bytes.Buffer{}

		depInfoCmd.Stderr = &depInfoStderr
		err := depInfoCmd.Run()
		if err != nil {
			logger.Fatalf("%s, %s, %s", err, string(depInfoStderr.Bytes()), depInfoOnlyCommand)
		}

		artifact, err := Rustc.GetDepArtifact(&depInfoStderr)
		if err != nil {
			logger.Fatalf("%s, %s, %s", err, string(depInfoStderr.Bytes()), depInfoOnlyCommand)
		}

		var files = Rustc.GetSourceFiles(artifact.Artifact)
		var checksum = FileManager.GetCheckSum(files, WorkDir)

		wrapperTool.CrateSourceChecksum = checksum
		logger.Debugf("Checksum: %s", checksum)
	}

	// try get from cache
	if wrapperTool.IsNeedProcessFromCache() && !gotRebuildDeps {

		var meta, existsInStore = store.GetMetadata(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())

		var needDownload = true

		if !existsInStore {
			logger.Debugf("%s does not exist in storage\n", wrapperTool.GetCachePackageName())
			needDownload = false
		} else if meta == nil {
			logger.Debugf("no metadata for %s, downloading...\n", wrapperTool.GetCachePackageName())
			needDownload = true
		} else if _, ok := meta["sha-1-content"]; !ok {
			logger.Debugf("no metadata header for %s, downloading...\n", wrapperTool.GetCachePackageName())
			needDownload = true
		} else {
			//var searchPath = filepath.Join("target", config.General.Dir)
			//var files = FileManager.Find(searchPath, crateHash, true)

			needDownload = true

			//TODO: check local files
		}

		if needDownload {
			var f = store.Download(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())
			if f != nil {
				Tar.UnPack(WorkDir, compressor.Decompress(f))
				logger.Infof("Downloaded artifacts for %s\n", wrapperTool.GetCachePackageName())

				var smth = bytes.Buffer{}
				var mw = io.MultiWriter(os.Stderr, &smth)
				io.Copy(mw, wrapperTool.ReadStderrFile())

				io.Copy(os.Stdout, wrapperTool.ReadIOStreamFile("stdout"))

				os.Exit(0)
			}
		}
	}

	// execute rustc
	logger.Infof("executing rustc")

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
		var abs = filepath.Join(WorkDir, artifact.Artifact)

		artifactsPaths = append(artifactsPaths, abs)
	}
	flClient.RegisterExternDeps(&artifactsPaths)

	if runErr != nil {
		if serr, ok := err.(*exec.ExitError); ok {
			os.Exit(serr.ExitCode())
		}
		os.Exit(1)
	}

	if wrapperTool.CrateName != "___" {
		wrapperTool.WriteToItemCacheFile()
	}
}
