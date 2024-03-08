package FileManager

import (
	"log"
	"os"
	"path/filepath"
)

func GetTrueRelFilePath(workDir string, path string) string {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	var absPath, err = filepath.Abs(path)
	if err != nil {
		log.Fatalln(err)
	}

	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		log.Fatalln(err)
	}

	relPath, err := filepath.Rel(workDir, absPath)
	if err != nil {
		log.Fatalln(err)
	}

	return relPath
}
