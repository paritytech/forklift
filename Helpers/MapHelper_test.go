package Helpers_test

import (
	"forklift/Helpers"
	"testing"
)

var testMap = map[string]interface{}{
	"bool":       true,
	"boolString": "true",
	"int":        int64(42),
	"intString":  "42",
	"str":        "Hello, World!",
}

func TestMapGet_bool(t *testing.T) {
	result := Helpers.MapGet(&testMap, "bool", false)
	if result != true {
		t.Error("Get existed failed")
	}

	result = Helpers.MapGet(&testMap, "boolString", false)
	if result != true {
		t.Error("Get existed string failed")
	}

	result = Helpers.MapGet(&testMap, "abc", true)
	if result != true {
		t.Error("Get non-existed failed")
	}
}

func TestMapGet_int(t *testing.T) {
	result := Helpers.MapGet(&testMap, "int", int64(0))
	if result != 42 {
		t.Error("Get existed failed")
	}

	result = Helpers.MapGet(&testMap, "intString", int64(0))
	if result != 42 {
		t.Error("Get existed string failed")
	}

	result = Helpers.MapGet(&testMap, "abc", int64(42))
	if result != 42 {
		t.Error("Get non-existed failed")
	}
}

func TestMapGet_str(t *testing.T) {
	result := Helpers.MapGet(&testMap, "str", "")
	if result != "Hello, World!" {
		t.Error("Get existed failed")
	}

	result = Helpers.MapGet(&testMap, "abc", "default")
	if result != "default" {
		t.Error("Get non-existed failed")
	}
}
