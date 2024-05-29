package Wrapper

import (
	"errors"
	"forklift/Lib/Diagnostic/Time"
	"forklift/Lib/Logging"
	log "forklift/Lib/Logging/ConsoleLogger"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	"forklift/Rpc/Models/CacheUsage"
	"os"
	"os/exec"
	"path/filepath"
)

var WorkDir string

func Run(args []string) {

	var rustcArgsOnly = args[1:]

	var timer = Time.NewForkliftTimer()

	wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR")

	if !ok || wd == "" {
		log.Fatalf("No `FORKLIFT_WORK_DIR` specified!")
		return
	}

	WorkDir = wd

	var wrapperTool = Rustc.NewWrapperToolFromArgs(WorkDir, &rustcArgsOnly)

	logger := Logging.CreateLogger("Wrapper", 4, log.Fields{
		"crate": wrapperTool.CrateName,
		"hash":  wrapperTool.CargoCrateHash,
	})
	wrapperTool.Logger = logger

	var flClient = Rpc.NewForkliftRpcClient()

	err := flClient.Connect()
	if err != nil {
		log.Errorf("Failed to connect to forklift server: %s", err)
		return
	}

	var cacheUsageReport = CacheUsage.StatusReport{
		CrateName: wrapperTool.CrateName,
	}

	// Real work starts here

	// check deps
	var deps = Rustc.GetExternDeps(&rustcArgsOnly, true)
	var rebuiltDep = flClient.CheckExternDeps(deps)
	var gotRebuildDeps = true
	if rebuiltDep == "" {
		gotRebuildDeps = false
	}

	if !wrapperTool.IsNeedProcessFromCache() {
		logger.Debugf("No need to use cache for %s", wrapperTool.CrateName)
		var rustcError = BypassRustc()
		if rustcError != nil {
			var exitError *exec.ExitError
			if errors.As(rustcError, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			os.Exit(1)
		}
		return
	}

	if gotRebuildDeps {
		logger.Debugf("Got rebuilt dep: %s", rebuiltDep)
		cacheUsageReport.Status = CacheUsage.DependencyRebuilt
		//flClient.ReportStatus(wrapperTool.CrateName, CacheUsage.DependencyRebuilt)
	} else {
		logger.Debugf("No rebuilt deps")
	}

	calcChecksum(wrapperTool)

	var cacheHit = false
	// try get from cache
	if !gotRebuildDeps {
		cacheHit = wrapperTool.TryUseCache(&cacheUsageReport)
	}

	if !cacheHit {
		// execute rustc
		timer.Start("rustc")
		logger.Infof("Executing rustc")
		logger.Tracef("Rustc args: %s", rustcArgsOnly)
		var artifacts, rustcError = wrapperTool.ExecuteRustc()
		cacheUsageReport.RustcTime += timer.Stop("rustc")

		if rustcError != nil {
			logger.Errorf("Rustc finished with error: %s", rustcError)
			var exitError *exec.ExitError
			if errors.As(rustcError, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			os.Exit(1)
		}
		logger.Debugf("Finished rustc")

		// register rebuilt artifacts
		RegisterRebuiltArtifacts(artifacts, flClient)

		flClient.AddUpload(wrapperTool.ToCacheItem())
	}

	flClient.ReportStatusObject(cacheUsageReport)
}

// BypassRustc - execute rustc as is, without any caching
func BypassRustc() error {
	var rustcArgsOnly = os.Args[2:]
	var cmd = exec.Command(os.Args[1], rustcArgsOnly...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	var rustcError = cmd.Run()

	return rustcError
}

// RegisterRebuiltArtifacts - register rebuilt artifacts
func RegisterRebuiltArtifacts(artifacts *[]Rustc.Artifact, flClient *Rpc.ForkliftRpcClient) {
	var artifactsPaths = make([]string, 0)
	for _, artifact := range *artifacts {
		var abs = filepath.Base(artifact.Artifact)

		artifactsPaths = append(artifactsPaths, abs)
	}
	flClient.RegisterExternDeps(&artifactsPaths)
}
