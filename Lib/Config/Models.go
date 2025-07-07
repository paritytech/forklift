package Config

type ForkliftConfig struct {
	Storage     ForkliftStorage
	Compression ForkliftCompression
	General     ForkliftGeneral
	Metrics     ForkliftMetrics
	Cache       ForkliftCache
}

type ForkliftStorage struct {
	Type     string
	ReadOnly bool
}

type ForkliftCache struct {
	ExtraEnv      []string
	ExtraMetadata map[string]string
}

type ForkliftCompression struct {
	Type string
}

type ForkliftGeneral struct {
	LogLevel        string
	ThreadsCount    int
	JobNameVariable string
	JobsBlacklist   []string
	PackageSuffix   string
	Quiet           bool
}

type ForkliftMetrics struct {
	Enabled      bool
	PushEndpoint string
	ExtraLabels  map[string]string
}
