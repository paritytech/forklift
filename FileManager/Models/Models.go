package Models

import "io/fs"

type TargetFsEntry struct {
	Path     string
	BasePath string
	Info     fs.FileInfo
}

type CacheItem struct {
	Name    string
	Version string
	HashInt string
	Hash    string
}