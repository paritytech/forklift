package Config_test

import (
	"fmt"
	"forklift/Lib/Config"
	"testing"
)

func TestForkliftConfig_Get(t *testing.T) {
	var path = "../../config-example.toml"
	Config.Init(&path)

	var actualValue = Config.AppConfig.Get("general.logLevel")
	var expectedValue = "info"

	if actualValue != expectedValue {
		fmt.Printf("Expected: %v\n", expectedValue)
		fmt.Printf("Actual  : %v\n", actualValue)
		t.Error("Test failed")
	}
}

func TestForkliftConfig_GetInt(t *testing.T) {
	var path = "../../config-example.toml"
	Config.Init(&path)

	var actualValue = Config.AppConfig.GetInt("general.threadscount")
	var expectedValue = int64(10)

	if actualValue != expectedValue {
		fmt.Printf("Expected: %v\n", expectedValue)
		fmt.Printf("Actual  : %v\n", actualValue)
		t.Error("Test failed")
	}
}

func TestForkliftConfig_GetString(t *testing.T) {
	var path = "../../config-example.toml"
	Config.Init(&path)

	var actualValue = Config.AppConfig.GetString("general.logLevel")
	var expectedValue = "info"

	if actualValue != expectedValue {
		fmt.Printf("Expected: %v\n", expectedValue)
		fmt.Printf("Actual  : %v\n", actualValue)
		t.Error("Test failed")
	}
}

func TestForkliftConfig_GetBool(t *testing.T) {
	var path = "../../config-example.toml"
	Config.Init(&path)

	var actualValue = Config.AppConfig.GetBool("storage.s3.useSsl")
	var expectedValue = true

	if actualValue != expectedValue {
		fmt.Printf("Expected: %v\n", expectedValue)
		fmt.Printf("Actual  : %v\n", actualValue)
		t.Error("Test failed")
	}
}

func TestForkliftConfig_Set(t *testing.T) {
	var path = "../../config-example.toml"
	Config.Init(&path)

	Config.AppConfig.Set("general.logLevel", "debug")

	var actualValue = Config.AppConfig.Get("general.logLevel")
	var expectedValue = "debug"

	if actualValue != expectedValue {
		fmt.Printf("Expected: %v\n", expectedValue)
		fmt.Printf("Actual  : %v\n", actualValue)
		t.Error("Test failed")
	}
}
