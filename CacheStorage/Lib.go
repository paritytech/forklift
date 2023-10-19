package CacheStorage

import (
	"crypto/sha1"
	"fmt"
)

type RustcArtifact struct {
	Artifact string
	Emit     string
}

func CreateCachePackageName(name string, hash string, outDir string, compressor string) string {
	var sha = sha1.New()
	sha.Write([]byte(outDir))
	return fmt.Sprintf("%s_%s_%x_%s", name, hash, string(sha.Sum(nil)), compressor)
}
