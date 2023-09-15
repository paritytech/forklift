package FileManager

import (
	"os"
	"testing"
)

func createTestFs(t *testing.T) {
	os.MkdirAll("./testDir", 0777)

	os.MkdirAll("./testDir/d1", 0777)
	os.MkdirAll("./testDir/d1/sd1", 0777)
	os.MkdirAll("./testDir/d1/ssd1", 0777)

	os.MkdirAll("./testDir/release/build/wasm-opt-cxx-sys-b6cbb26960b880b1/out/cxxbridge/crate/wasm-opt-cxx-sys", 0777)

	os.MkdirAll("./testDir/d1/sd2", 0777)

	os.MkdirAll("./testDir/d2", 0777)

	os.WriteFile("./testDir/d1/ssd1/file1", []byte("qwertyuiop"), 0777)
	os.WriteFile("./testDir/release/build/wasm-opt-cxx-sys-b6cbb26960b880b1/out/cxxbridge/crate/wasm-opt-cxx-sys/file2", []byte("qwertyuiop"), 777)
}

func removeTestFs() {
	os.RemoveAll("./testDir")
	os.RemoveAll("./untarDir")
}

func TestTar(t *testing.T) {
	createTestFs(t)
	var entries = FindOpt("./", "testDir")

	Tar(entries)

	t.Cleanup(func() {
		removeTestFs()
	})
}

func TestUnTar(t *testing.T) {

	createTestFs(t)
	t.Cleanup(func() {
		removeTestFs()
	})

	var entries = FindOpt("./testDir/release/build/", "b6cbb26960b880b1")
	var reader, _ = Tar(entries)
	UnTar("./untarDir/", reader)
}
