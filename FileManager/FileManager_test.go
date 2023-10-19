package FileManager

import (
	"fmt"
	"forklift/FileManager/Tar"
	"os"
	"testing"
)

func createTestFs(t *testing.T) {
	os.MkdirAll("./testDir", 0777)

	os.MkdirAll("./testDir/d1-KEY", 0777)
	os.MkdirAll("./testDir/d1-KEY/sd1", 0777)
	os.MkdirAll("./testDir/d1-KEY/ssd1", 0777)
	os.MkdirAll("./testDir/d2/ssd1-KEY", 0777)

	os.MkdirAll("./testDir/release/build/wasm-opt-cxx-sys-b6cbb26960b880b1/out/cxxbridge/crate/wasm-opt-cxx-sys", 0777)

	os.MkdirAll("./testDir/d1-KEY/sd2", 0777)

	os.MkdirAll("./testDir/d2", 0777)

	os.WriteFile("./testDir/d1-KEY/ssd1/file1", []byte("qwertyuiop"), 0777)
	os.WriteFile("./testDir/d1-KEY/ssd1/file2-KEY", []byte("qwertyuiop"), 0777)
	os.WriteFile("./testDir/release/build/wasm-opt-cxx-sys-b6cbb26960b880b1/out/cxxbridge/crate/wasm-opt-cxx-sys/file2", []byte("qwertyuiop"), 777)
}

func removeTestFs() {
	os.RemoveAll("./testDir")
	os.RemoveAll("./untarDir")
}

func TestFindRecursive(t *testing.T) {
	createTestFs(t)
	t.Cleanup(func() {
		removeTestFs()
	})

	var result = findRecursive("testDir", "KEY", false)
	for _, entry := range result {
		fmt.Println(entry.Path)
	}
}

func TestTar(t *testing.T) {
	createTestFs(t)
	var entries = Find("./", "testDir")

	Tar.Pack(entries)

	t.Cleanup(func() {
		removeTestFs()
	})
}

func TestUnTar(t *testing.T) {

	createTestFs(t)
	t.Cleanup(func() {
		removeTestFs()
	})

	var entries = Find("./testDir/release/build/", "b6cbb26960b880b1")
	var reader, _ = Tar.Pack(entries)
	Tar.UnPack("./untarDir/", reader)
}
