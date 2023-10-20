package Wrapper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Tar"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var logger = log.WithFields(log.Fields{})

var WorkDir string

func Run(args []string) {
	//var outBuf = bytes.Buffer{}
	//log.SetOutput(&outBuf)

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		logger.Fatalln("NO `FORKLIFT_WORK_DIR` specified!")
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

	var crateName, crateHash, outDir = extractNameMetaHashDir(args)

	outDir, _ = filepath.Rel(WorkDir, outDir)

	var logger = log.WithFields(log.Fields{
		"crate": crateName,
		"hash":  crateHash,
	})

	store, _ := Storages.GetStorageDriver(Lib.AppConfig)
	compressor, _ := Compressors.GetCompressor(Lib.AppConfig)

	var cachePackageName = CacheStorage.CreateCachePackageName(crateName, crateHash, outDir, compressor.GetKey())

	logger.Tracef("wrapper args: %s\n", os.Args)

	if crateName != "" &&
		crateHash != "" &&
		!strings.Contains(outDir, "/var/folders/") &&
		!strings.HasPrefix(outDir, "/tmp") {

		//var cachePackageName = CacheStorage.CreateCachePackageName(crateName, crateHash, outDir, compressor.GetKey())
		writeToItemCacheFile(crateName, crateHash, cachePackageName, outDir)

		fmt.Sprintf("%s-%s-%s", crateName, crateHash, compressor.GetKey())
		var meta, existsInStore = store.GetMetadata(cachePackageName)

		var needDownload = true

		if !existsInStore {
			logger.Debugf("%s does not exist in storage\n", cachePackageName)
			needDownload = false
		} else if meta == nil {
			logger.Debugf("no metadata for %s, downloading...\n", cachePackageName)
			needDownload = true
		} else if _, ok := meta["sha-1-content"]; !ok {
			logger.Debugf("no metadata header for %s, downloading...\n", cachePackageName)
			needDownload = true
		} else {
			//var searchPath = filepath.Join("target", config.General.Dir)
			//var files = FileManager.Find(searchPath, crateHash, true)

			needDownload = true

			/*
				if len(files) <= 0 {
					log.Debugf("%s no local files, downloading...\n", cachePackageName)
					needDownload = true
				} else {
					var _, sha = Tar.Pack(files)
					var shaLocal = fmt.Sprintf("%x", sha.Sum(nil))

					var shaRemote = *shaRemotePtr

					if shaRemote != shaLocal {
						logger.Debugf("%s checksum mismatch, remote: %s local: %s, downloading...\n", cachePackageName, shaRemote, shaLocal)
						needDownload = true
					} else {
						logger.Tracef("%s checksum match , remote: %s local: %s\n", cachePackageName, shaRemote, shaLocal)
						needDownload = false
					}
				}*/
		}

		if needDownload {
			var f = store.Download(cachePackageName)
			if f != nil {
				Tar.UnPack(WorkDir, compressor.Decompress(f))
				logger.Infof("Downloaded artifacts for %s\n", cachePackageName)

				io.Copy(os.Stderr, readStderrFile(cachePackageName))
				io.Copy(os.Stdout, ReadIOStreamFile(cachePackageName, "stdout"))

				os.Exit(0)
			}
		}
	}

	// execute rustc
	logger.Debug("executing rustc")

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

	writeIOStreamFile(&rustcStdout, cachePackageName, "stdout")
	writeStderrFIle(&rustcStderr, cachePackageName)
	writeIOStreamFile(&rustcStdin2, cachePackageName, "stdin")

	if runErr != nil {
		if serr, ok := err.(*exec.ExitError); ok {
			os.Exit(serr.ExitCode())
		}
		os.Exit(1)
	}
}

func writeToItemCacheFile(crateName string, crateHash string, cachePackageName string, outDir string) {

	var itemsCachePath = path.Join(WorkDir, ".forklift", "items-cache")
	err := os.MkdirAll(itemsCachePath, 0755)
	if err != nil {
		logger.Errorln(err)
	}

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("item-%s", cachePackageName)),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		logger.Errorln(err)
	}

	_, err = itemFile.WriteString(fmt.Sprintf("%s | | | %s | %s | %s \n", crateName, crateHash, cachePackageName, outDir))
	if err != nil {
		logger.Errorln(err)
	}

	err = itemFile.Close()
	if err != nil {
		logger.Errorln(err)
	}
}

func readStderrFile(cachePackageName string) io.Reader {
	var itemsCachePath = path.Join(WorkDir, "target", Lib.AppConfig.General.Dir, "forklift")
	var file, _ = os.Open(path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", cachePackageName)))

	var resultBuf = bytes.Buffer{}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		var artifact CacheStorage.RustcArtifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var absPath, _ = filepath.Abs(artifact.Artifact)
			artifact.Artifact = absPath
			var newArtifactByte, _ = json.Marshal(artifact)
			resultBuf.Write(newArtifactByte)
		} else {
			resultBuf.WriteString(str)
		}
		resultBuf.WriteString("\n")
	}

	return &resultBuf
}

func writeStderrFIle(reader io.Reader, cachePackageName string) {
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	var itemsCachePath = path.Join(WorkDir, "target", Lib.AppConfig.General.Dir, "forklift")

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", cachePackageName)),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		logger.Errorln(err)
	}

	for fileScanner.Scan() {
		var artifact CacheStorage.RustcArtifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var relpath, _ = filepath.Rel(WorkDir, artifact.Artifact)
			artifact.Artifact = relpath

			var newArtifactByte, _ = json.Marshal(artifact)
			itemFile.Write(newArtifactByte)
		} else {
			itemFile.WriteString(str)
		}
		itemFile.WriteString("\n")
	}
}

func writeIOStreamFile(reader io.Reader, cachePackageName string, suffix string) {

	var itemsCachePath = path.Join(WorkDir, "target", Lib.AppConfig.General.Dir, "forklift")
	err := os.MkdirAll(itemsCachePath, 0755)
	if err != nil {
		logger.Errorln(err)
	}

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", cachePackageName, suffix)),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		logger.Errorln(err)
	}

	_, err = io.Copy(itemFile, reader)
	if err != nil {
		logger.Errorln(err)
	}

	err = itemFile.Close()
	if err != nil {
		logger.Errorln(err)
	}
}

func ReadIOStreamFile(cachePackageName string, suffix string) io.Reader {

	var itemsCachePath = path.Join(WorkDir, "target", Lib.AppConfig.General.Dir, "forklift")

	itemFile, err := os.Open(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", cachePackageName, suffix)),
	)
	if err != nil {
		logger.Errorln(err)
	}

	var result = bytes.Buffer{}
	io.Copy(&result, itemFile)
	itemFile.Close()

	return &result
}

// /
// /
// /
var regex = regexp.MustCompile("^metadata=([0-9a-f]{16})$")

func extractNameMetaHashDir(args []string) (string, string, string) {

	var name, hash, outDir string

	var count = 0

	for i, arg := range args {

		if name == "" && arg == "--crate-name" {
			name = args[i+1]
			count += 1
		}

		if hash == "" {
			var match = regex.FindAllStringSubmatch(arg, 1)
			if len(match) > 0 {
				hash = match[0][1]
				count += 1
			}
		}

		if outDir == "" && arg == "--out-dir" {
			outDir = args[i+1]
			count += 1
		}

		if count >= 3 {
			break
		}
	}

	return name, hash, outDir
}
