package Rustc

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"forklift/CacheStorage"
	"forklift/FileManager/Models"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type WrapperTool struct {
	Logger    *log.Entry
	CrateName string
	CrateHash string
	OutDir    string
	//CachePackageName    string
	workDir             string
	CrateSourceChecksum string
}

func NewWrapperToolFromArgs(workDir string, rustArgs *[]string) *WrapperTool {
	var wrapper = WrapperTool{}
	wrapper.CrateName, wrapper.CrateHash, wrapper.OutDir = wrapper.extractNameMetaHashDir(rustArgs)
	wrapper.workDir = workDir
	wrapper.OutDir, _ = filepath.Rel(wrapper.workDir, wrapper.OutDir)
	//wrapper.CachePackageName = createCachePackageName(wrapper.CrateName, wrapper.CrateHash, wrapper.OutDir)
	wrapper.createLogger()

	return &wrapper
}

func NewWrapperToolFromCacheItem(workDir string, item Models.CacheItem) *WrapperTool {
	var wrapper = WrapperTool{}
	wrapper.CrateName = item.Name
	wrapper.CrateHash = item.Hash
	wrapper.OutDir = item.OutDir
	wrapper.workDir = workDir
	//wrapper.OutDir, _ = filepath.Rel(wrapper.workDir, wrapper.OutDir)
	//wrapper.CachePackageName = createCachePackageName(wrapper.CrateName, wrapper.CrateHash, wrapper.OutDir)
	wrapper.CrateSourceChecksum = item.CrateSourceChecksum
	wrapper.createLogger()

	return &wrapper
}

func (wrapperTool *WrapperTool) createLogger() {
	wrapperTool.Logger = log.WithFields(log.Fields{
		"crate": wrapperTool.CrateName,
		"hash":  wrapperTool.CrateHash,
	})
}

func (wrapperTool *WrapperTool) IsNeedProcessFromCache() bool {
	return wrapperTool.CrateName != "" &&
		wrapperTool.CrateHash != "" &&
		!strings.Contains(wrapperTool.OutDir, "/var/folders/") &&
		!strings.HasPrefix(wrapperTool.OutDir, "/tmp")
}

func (wrapperTool *WrapperTool) GetCachePackageName() string {

	var sha = sha1.New()
	sha.Write([]byte(wrapperTool.OutDir))

	if prefix, ok := os.LookupEnv("FORKLIFT_PACKAGE_SUFFIX"); ok {
		return fmt.Sprintf(
			"%s_%s_%s_%x_%s",
			wrapperTool.CrateName,
			wrapperTool.CrateHash,
			wrapperTool.CrateSourceChecksum,
			sha.Sum(nil),
			prefix)
	} else {
		return fmt.Sprintf(
			"%s_%s_%s_%x",
			wrapperTool.CrateName,
			wrapperTool.CrateHash,
			wrapperTool.CrateSourceChecksum,
			sha.Sum(nil))
	}

}

func (wrapperTool *WrapperTool) WriteToItemCacheFile() {

	var itemsCachePath = path.Join(wrapperTool.workDir, ".forklift", "items-cache")
	err := os.MkdirAll(itemsCachePath, 0755)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("item-%s", wrapperTool.GetCachePackageName())),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	_, err = itemFile.WriteString(fmt.Sprintf(
		"%s | | | %s | %s | %s | %s  \n",
		wrapperTool.CrateName,
		wrapperTool.CrateHash,
		wrapperTool.GetCachePackageName(),
		wrapperTool.OutDir,
		wrapperTool.CrateSourceChecksum))
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	err = itemFile.Close()
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}
}

func (wrapperTool *WrapperTool) ReadStderrFile() io.Reader {
	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")
	var file, _ = os.Open(path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", wrapperTool.GetCachePackageName())))

	var resultBuf = bytes.Buffer{}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		var artifact CacheStorage.RustcArtifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var absPath = filepath.Join(wrapperTool.workDir, artifact.Artifact)
			artifact.Artifact = absPath
			var newArtifactByte, _ = json.Marshal(artifact)
			wrapperTool.Logger.Tracef("read artifact as %s", absPath)
			resultBuf.Write(newArtifactByte)
		} else {
			resultBuf.WriteString(str)
		}
		resultBuf.WriteString("\n")
	}

	return &resultBuf
}

func (wrapperTool *WrapperTool) WriteStderrFile(reader io.Reader) *[]CacheStorage.RustcArtifact {
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-stderr", wrapperTool.GetCachePackageName())),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	var result []CacheStorage.RustcArtifact

	for fileScanner.Scan() {
		var artifact CacheStorage.RustcArtifact
		var str = fileScanner.Text()
		json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			var relpath, _ = filepath.Rel(wrapperTool.workDir, artifact.Artifact)
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
		wrapperTool.Logger.Errorln(err)
	}

	itemFile, err := os.OpenFile(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), suffix)),
		os.O_TRUNC|os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	_, err = io.Copy(itemFile, reader)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}

	err = itemFile.Close()
	if err != nil {
		wrapperTool.Logger.Errorln(err)
	}
}

func (wrapperTool *WrapperTool) ReadIOStreamFile(suffix string) io.Reader {

	var itemsCachePath = path.Join(wrapperTool.workDir, "target", "forklift")

	itemFile, err := os.Open(
		path.Join(itemsCachePath, fmt.Sprintf("%s-%s", wrapperTool.GetCachePackageName(), suffix)),
	)
	if err != nil {
		wrapperTool.Logger.Errorln(err)
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

func (wrapperTool *WrapperTool) extractNameMetaHashDir(args *[]string) (string, string, string) {

	var name, hash, outDir string

	var count = 0

	for i, arg := range *args {

		if name == "" && arg == "--crate-name" {
			name = (*args)[i+1]
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
			outDir = (*args)[i+1]
			count += 1
		}

		if count >= 3 {
			break
		}
	}

	return name, hash, outDir
}
