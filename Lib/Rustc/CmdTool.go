package Rustc

import (
	"bufio"
	"encoding/json"
	"errors"
	"forklift/CacheStorage"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CmdTool struct {
}

// CreateDepInfoCommand Create same command? but with --emit=dep-info only
func CreateDepInfoCommand(args *[]string) []string {
	var result = make([]string, len(*args)+2)

	for i, arg := range *args {
		if len(arg) > 7 && arg[:7] == "--emit=" {
			result[i] = "--emit=dep-info"
			/*} else if arg == "debug-assertions=off" {
			result[i] = "debug-assertions=on"*/
		} else {
			result[i] = arg
		}
	}

	result[len(result)-2] = "-C"
	result[len(result)-1] = "debuginfo=2" +
		""

	return result
}

func GetExternDeps(args *[]string, basePathOnly bool) *[]string {
	var result []string

	for i := 0; i < len(*args); i++ {
		if (*args)[i] == "--extern" {

			var parts = strings.Split((*args)[i+1], "=")

			if len(parts) < 2 && basePathOnly {
				result = append(result, parts[0])
			} else if len(parts) < 2 && !basePathOnly {
			} else if basePathOnly {
				result = append(result, filepath.Base(parts[1]))
			} else {
				result = append(result, parts[1])
			}

			i++
		}
	}

	return &result
}

func GetNativeDeps(args *[]string, basePathOnly bool) *[]string {
	var result []string

	for i := 0; i < len(*args); i++ {
		if (*args)[i] == "-L" {

			var parts = strings.Split((*args)[i+1], "=")

			if len(parts) < 2 {
				continue
			}

			if parts[0] != "native" {
				continue
			}
			result = append(result, parts[1])

			i++
		}
	}

	return &result
}

func GetDepArtifact(reader io.Reader) (CacheStorage.RustcArtifact, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		var artifact CacheStorage.RustcArtifact
		var str = scanner.Text()
		_ = json.Unmarshal([]byte(str), &artifact)
		if artifact.Artifact != "" {
			return artifact, nil
		}
	}

	return CacheStorage.RustcArtifact{}, errors.New("no .d artifact in output")
}

func GetSourceFiles(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	scanner.Scan()
	var depsString = scanner.Text()

	return strings.Split(depsString, " ")[1:]
}
