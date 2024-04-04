package Rustc_test

import (
	"bytes"
	"fmt"
	"forklift/Lib/Rustc"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestCmdTool_GetExternDeps(t *testing.T) {

	var input = []string{"qwerty", "asdfgh", "--extern", "a=a/b/c", "--extern", "d=d/e/f", "--extern", "g=g/h/i", "-extern", "j=j/k/l"}

	var nonBasePathResult = Rustc.GetExternDeps(&input, false)
	if !reflect.DeepEqual(*nonBasePathResult, []string{"a/b/c", "d/e/f", "g/h/i"}) {
		t.Error("Test failed")
	}

	var onlyBasePathResult = Rustc.GetExternDeps(&input, true)
	if !reflect.DeepEqual(*onlyBasePathResult, []string{"c", "f", "i"}) {
		t.Error("Test failed")
	}
}

func TestWrapperTool_WriteStderrFile(t *testing.T) {
	var wd, _ = os.Getwd()
	var wrapper = Rustc.NewWrapperToolFromArgs(wd, &[]string{})

	var data = "{\"artifact\":\"deps/base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}"
	var expectedData = "{\"artifact\":\"base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}\n"

	var reader = bytes.NewReader([]byte(data))

	var artifacts = wrapper.WriteStderrFile(reader)

	if len(*artifacts) != 1 {
		t.Error("No artifact")
	}

	if (*artifacts)[0].Artifact != "base64-a62ed92405ecbfa1.d" {
		t.Error("Wrong artifact")
	}

	var fileData, _ = os.ReadFile("target/forklift/" + wrapper.GetCachePackageName() + "-stderr")

	var actualData = string(fileData)
	if actualData != expectedData {
		t.Error("Data mismatch")
	}

}

func TestWrapperTool_ReadStderrFile(t *testing.T) {
	var wd, _ = os.Getwd()
	var wrapper = Rustc.NewWrapperToolFromArgs(wd, &[]string{"-a", "b"})

	var data = "{\"artifact\":\"target/deps/base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}"

	var itemsCachePath = path.Join(wd, "target", "forklift")
	os.MkdirAll(itemsCachePath, 0755)
	os.WriteFile("target/forklift/"+wrapper.GetCachePackageName()+"-stderr", []byte(data), 0755)

	var expectedData = "{\"artifact\":\"" + wd + "/target/deps/base64-a62ed92405ecbfa1.d\",\"emit\":\"dep-info\"}\n"

	var reader = wrapper.ReadStderrFile()
	var buf = bytes.Buffer{}
	buf.ReadFrom(reader)

	var actualData = buf.String()
	if actualData != expectedData {
		fmt.Printf("Expected: %s\n", expectedData)
		fmt.Printf("Actual  : %s\n", actualData)
		t.Error("Data mismatch")
	}
}
