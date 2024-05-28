package Rustc

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"forklift/FileManager/Models"
	"forklift/FileManager/Tar"
	"forklift/Lib/Config"
	"forklift/Lib/Diagnostic/Time"
	log "forklift/Lib/Logging/ConsoleLogger"
	"forklift/Rpc/Models/CacheUsage"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const CachePackageVersion = "3"

var cargoHashRegex = regexp.MustCompile("^metadata=([0-9a-f]{16})$")

type WrapperTool struct {
	rustcArgs               *[]string
	Logger                  *log.Logger
	CrateName               string
	CargoCrateHash          string
	OutDir                  string
	SourceFile              string
	RustCArgsHash           string
	workDir                 string
	CrateSourceChecksum     string
	osWorkDir               string
	CrateExternDepsChecksum string
	CrateNativeDepsChecksum string

	cachePackageName string
}

func NewWrapperToolFromArgs(workDir string, rustArgs *[]string) *WrapperTool {
	var wrapper = WrapperTool{}
	wrapper.rustcArgs = rustArgs

	wrapper.CrateName, wrapper.CargoCrateHash, wrapper.OutDir = wrapper.extractNameMetaHashDir(rustArgs)
	wrapper.workDir = workDir
	wrapper.OutDir = FileManager.GetTrueRelFilePath(wrapper.workDir, wrapper.OutDir)
	wrapper.RustCArgsHash = GetArgsHash(rustArgs, wrapper.workDir)

	wrapper.ExternDepsChecksum()

	var osWorkDir, _ = os.Getwd()
	wrapper.osWorkDir = osWorkDir

	return &wrapper
}

func NewWrapperToolFromCacheItem(workDir string, item Models.CacheItem) *WrapperTool {
	var wrapper = WrapperTool{}
	wrapper.CrateName = item.Name
	wrapper.CargoCrateHash = item.Hash
	wrapper.OutDir = item.OutDir
	wrapper.workDir = workDir
	wrapper.CrateSourceChecksum = item.CrateSourceChecksum
	wrapper.RustCArgsHash = item.RustCArgsHash

	wrapper.CrateExternDepsChecksum = item.CrateExternDepsChecksum
	wrapper.CrateNativeDepsChecksum = item.CrateNativeDepsChecksum

	return &wrapper
}

func GetArgsHash(args *[]string, toRemove string) string {
	var sha = sha1.New()
	for _, arg := range *args {
		var stripped = strings.Replace(arg, toRemove, "", -1)
		sha.Write([]byte(stripped))
	}

	return fmt.Sprintf("%x", sha.Sum(nil))
}

func (wrapperTool *WrapperTool) IsNeedProcessFromCache() bool {
	return wrapperTool.CrateName != "" &&
		wrapperTool.CrateName != "___" &&
		wrapperTool.CargoCrateHash != "" &&
		!strings.Contains(wrapperTool.OutDir, "/var/folders/") &&
		!strings.Contains(wrapperTool.OutDir, "/tmp")
}

// ExecuteRustc - execute rustc and process output
func (wrapperTool *WrapperTool) ExecuteRustc() (*[]Artifact, error) {

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

// TryUseCache - try to use cache, return false if failed
func (wrapperTool *WrapperTool) TryUseCache(cacheUsageReport *CacheUsage.StatusReport) bool {

	var timer = Time.NewForkliftTimer()

	store, _ := Storages.GetStorageDriver(Config.AppConfig)
	compressor, _ := Compressors.GetCompressor(Config.AppConfig)

	var retries = 3
	for retries > 0 {
		// try download
		timer.Start("download")
		downloadResult, err := store.Download(wrapperTool.GetCachePackageName() + "_" + compressor.GetKey())
		cacheUsageReport.DownloadTime += timer.Stop("download")
		if downloadResult == nil && err == nil {
			wrapperTool.Logger.Debugf("%s does not exist in storage", wrapperTool.GetCachePackageName())
			cacheUsageReport.Status = CacheUsage.CacheMiss
			return false
		}
		if err != nil {
			wrapperTool.Logger.Warningf("download error: %s", err)
			retries--
			continue
		}
		cacheUsageReport.DownloadSize += downloadResult.BytesCount
		cacheUsageReport.DownloadSpeedBps += downloadResult.SpeedBps

		// try decompress
		timer.Start("decompress")
		decompressed, err := compressor.Decompress(downloadResult.Data)
		cacheUsageReport.DecompressTime += timer.Stop("decompress")
		if err != nil {
			wrapperTool.Logger.Warningf("decompression error: %s", err)
			retries--
			continue
		}

		// try unpack
		timer.Start("unpack")
		err = Tar.UnPack(wrapperTool.workDir, decompressed)
		cacheUsageReport.UnpackTime += timer.Stop("unpack")
		if err != nil {
			wrapperTool.Logger.Warningf("unpack error: %s", err)
			retries--
			continue
		} else {

			io.Copy(os.Stderr, wrapperTool.ReadStderrFile())
			wrapperTool.Logger.Infof("Downloaded and unpacked artifacts for %s", wrapperTool.GetCachePackageName())

			if retries == 3 {
				cacheUsageReport.Status = CacheUsage.CacheHit
			} else {
				cacheUsageReport.Status = CacheUsage.CacheHitWithRetry
			}

			return true
		}
	}
	if retries <= 0 {
		wrapperTool.Logger.Errorf("Failed to pull artifacts for %s", wrapperTool.GetCachePackageName())
		cacheUsageReport.Status = CacheUsage.CacheFetchFailed
	}
	return false
}

// ExternDepsChecksum Calculates checksum of extern deps artifacts (sha1 of all files data)
func (wrapperTool *WrapperTool) ExternDepsChecksum() string {

	if wrapperTool.CrateExternDepsChecksum != "" {
		return wrapperTool.CrateExternDepsChecksum
	}

	var deps = GetExternDeps(wrapperTool.rustcArgs, false)
	var sha = sha1.New()
	for _, dep := range *deps {
		var data, err = os.Open(dep)
		if err != nil {
			//wrapperTool.Logger.Errorf("%s", err)
		}
		io.Copy(sha, data)
	}

	wrapperTool.CrateExternDepsChecksum = fmt.Sprintf("%x", sha.Sum(nil))

	return wrapperTool.CrateExternDepsChecksum
}

// ExtraEnvVarsChecksum - calculates checksum of environment variables
func (wrapperTool *WrapperTool) ExtraEnvVarsChecksum() []byte {
	var sha = sha1.New()
	for i, varName := range Config.AppConfig.Cache.ExtraEnv {
		if varValue, ok := os.LookupEnv(varName); ok {
			sha.Write([]byte(fmt.Sprintf("%d:%s:%s", i, varName, varValue)))
		}
	}
	return sha.Sum(nil)
}

// GetCachePackageName Calculates unique name (cache key) for cache package like `base64_abcdef012345...`.
// Suffix is sha1 of crate name, crate source checksum, crate hash, out dir, rustc args hash, extern deps checksum.
func (wrapperTool *WrapperTool) GetCachePackageName() string {

	if wrapperTool.cachePackageName != "" {
		return wrapperTool.cachePackageName
	}

	var sha = sha1.New()

	sha.Write([]byte(CachePackageVersion))
	sha.Write([]byte(wrapperTool.CrateSourceChecksum))
	sha.Write([]byte(wrapperTool.OutDir))
	sha.Write([]byte(wrapperTool.RustCArgsHash))
	sha.Write([]byte(wrapperTool.ExternDepsChecksum()))

	sha.Write(wrapperTool.ExtraEnvVarsChecksum())

	var result = fmt.Sprintf(
		"%s_%x",
		wrapperTool.CrateName,
		sha.Sum(nil))

	if Config.AppConfig.General.PackageSuffix != "" {
		result += "_" + Config.AppConfig.General.PackageSuffix
	}

	wrapperTool.cachePackageName = result
	return result
}

func (wrapperTool *WrapperTool) ToCacheItem() Models.CacheItem {
	var item = Models.CacheItem{
		Name:                wrapperTool.CrateName,
		Hash:                wrapperTool.CargoCrateHash,
		CachePackageName:    wrapperTool.GetCachePackageName(),
		OutDir:              wrapperTool.OutDir,
		CrateSourceChecksum: wrapperTool.CrateSourceChecksum,
		RustCArgsHash:       wrapperTool.RustCArgsHash,

		CrateExternDepsChecksum: wrapperTool.CrateExternDepsChecksum,
		CrateNativeDepsChecksum: wrapperTool.CrateNativeDepsChecksum,
	}

	return item
}

func (wrapperTool *WrapperTool) ReadStderrFile() io.Reader {
	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")
	var file, _ = os.Open(path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", wrapperTool.GetCachePackageName())))

	var resultBuf = bytes.Buffer{}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		var artifact Artifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var absPath = filepath.Join(wrapperTool.workDir, wrapperTool.OutDir, artifact.Artifact)
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

func (wrapperTool *WrapperTool) WriteStderrFile(reader io.Reader) *[]Artifact {
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")
	err := os.MkdirAll(itemsCachePath, 0755)

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", wrapperTool.GetCachePackageName())),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}

	var result []Artifact

	for fileScanner.Scan() {
		var artifact Artifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var relpath = filepath.Base(artifact.Artifact)
			artifact.Artifact = relpath
			result = append(result, artifact)

			var newArtifactByte, _ = json.Marshal(artifact)
			itemFile.Write(newArtifactByte)
		} else {
			itemFile.WriteString(str)
		}
		itemFile.WriteString("\n")
	}

	return &result
}

func (wrapperTool *WrapperTool) WriteIOStreamFile(reader io.Reader, suffix string) {

	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")
	err := os.MkdirAll(itemsCachePath, 0755)
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), suffix)),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}

	_, err = io.Copy(itemFile, reader)
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}

	err = itemFile.Close()
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}
}

func (wrapperTool *WrapperTool) ReadIOStreamFile(suffix string) io.Reader {

	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")

	itemFile, err := os.Open(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), suffix)),
	)
	if err != nil {
		wrapperTool.Logger.Errorf(err.Error())
	}

	var result = bytes.Buffer{}
	io.Copy(&result, itemFile)
	itemFile.Close()

	return &result
}

func (wrapperTool *WrapperTool) CreateMetadata() map[string]*string {
	var metaMap = map[string]*string{
		"cargo-hash":        &wrapperTool.CargoCrateHash,
		"sha1-source-files": &wrapperTool.CrateSourceChecksum,
		"sha1-rustc-args":   &wrapperTool.RustCArgsHash,
		"sha1-extern-deps":  &wrapperTool.CrateExternDepsChecksum,
		"sha1-native-deps":  &wrapperTool.CrateNativeDepsChecksum,
	}
	return metaMap
}

// extractNameMetaHashDir - extract crate name, cargo hash and output dir from rustc args
func (wrapperTool *WrapperTool) extractNameMetaHashDir(args *[]string) (string, string, string) {

	var name, hash, outDir string

	var count = 0

	for i, arg := range *args {

		if wrapperTool.SourceFile == "" && strings.HasPrefix(arg, "--edition") {
			wrapperTool.SourceFile = (*args)[i+1]
			count += 1
		}

		if name == "" && arg == "--crate-name" {
			name = (*args)[i+1]
			count += 1
		}

		if hash == "" {
			var match = cargoHashRegex.FindAllStringSubmatch(arg, 1)
			if len(match) > 0 {
				hash = match[0][1]
				count += 1
			}
		}

		if outDir == "" && arg == "--out-dir" {
			outDir = (*args)[i+1]
			count += 1
		}

		if count >= 4 {
			break
		}
	}

	return name, hash, outDir
}
