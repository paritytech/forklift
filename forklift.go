package main

import (
	"encoding/binary"
	"fmt"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager"
	"os"
	"strings"
)

func main() {

	//os.Setenv("S3_ACCESS_KEY_ID", "pLwDdUcVhCKNqXDEH4ze")
	//os.Setenv("S3_SECRET_ACCESS_KEY", "qvtkoH5m1hA3ZCHuTMD1RlbOeuVGdvsSlD8bE8lZ")
	//os.Setenv("S3_ENDPOINT_URL", "192.168.1.2:9000")
	//os.Setenv("S3_BUCKET_NAME", "forklift")

	if len(os.Args) <= 1 {
		fmt.Println("use pull or push")
		return
	}

	if os.Args[1] == "check" {
		var cacheItems = parseCacheRequest()
		fmt.Println(cacheItems)
	}

	if os.Args[1] == "push" {
		store := Storages.NewS3Storage()
		var cacheItems = parseCacheRequest()

		for _, item := range cacheItems {
			var files = FileManager.Find("./target/debug/build", item.hash)

			if len(files) > 0 {
				var reader = FileManager.Tar(files)
				store.Upload(item.hash+"-build", reader)
				fmt.Println("Uploaded", len(files), "entries from 'build' for", item.name, item.hash)
			} else {
				fmt.Println("No entries from 'build' for", item.name, item.hash)
			}
		}

		for _, item := range cacheItems {
			var files = FileManager.Find("./target/debug/deps", item.hash)

			if len(files) > 0 {
				var reader = FileManager.Tar(files)
				store.Upload(item.hash+"-deps", reader)
				fmt.Println("Uploaded", len(files), "entries from 'deps' for", item.name, item.hash)
			} else {
				fmt.Println("No entries from 'deps' for", item.name, item.hash)
			}
		}

		for _, item := range cacheItems {
			var files = FileManager.Find("./target/debug/.fingerprint", item.hash)

			if len(files) > 0 {
				var reader = FileManager.Tar(files)
				store.Upload(item.hash+"-fp", reader)
				fmt.Println("Uploaded", len(files), "entries from '.fingerprint' for", item.name, item.hash)
			} else {
				fmt.Println("No entries from '.fingerprint' for", item.name, item.hash)
			}
		}
	}

	if os.Args[1] == "pull" {
		store2 := Storages.NewS3Storage()
		var cacheItems2 = parseCacheRequest()

		for i, item := range cacheItems2 {
			FileManager.UnTar("./target/debug/build", store2.Download(item.hash+"-build"))

			FileManager.UnTar("./target/debug/deps", store2.Download(item.hash+"-deps"))

			FileManager.UnTar("./target/debug/.fingerprint", store2.Download(item.hash+"-fp"))

			fmt.Println("Downloaded artifacts for", item.name, item.hash, i+1, "/", len(cacheItems2))
		}
	}

	return
}

func Int64ToByteArray(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)

	return b
}

func Int32ToByteArray(num uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, num)

	return b
}

type CacheItem struct {
	name    string
	version string
	hashInt string
	hash    string
}

func parseCacheRequest() []CacheItem {

	var b, _ = os.ReadFile("./items.cache")
	str := string(b)
	var splitStrings = strings.Split(strings.ReplaceAll(str, "\r\n", "\n"), "\n")

	var result []CacheItem

	for i := range splitStrings {
		var itemParts = strings.Split(splitStrings[i], "|")
		if len(itemParts) < 4 {
			continue
		}
		result = append(result, CacheItem{
			name:    strings.TrimSpace(itemParts[0]),
			version: strings.TrimSpace(itemParts[1]),
			hashInt: strings.TrimSpace(itemParts[2]),
			hash:    strings.TrimSpace(itemParts[3]),
		})
	}

	return result
}
