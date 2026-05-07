package Rpc_test

import (
	"encoding/json"
	"forklift/CacheStorage/Compressors"
	"forklift/CacheStorage/Storages"
	"forklift/FileManager/Models"
	"forklift/Lib/Rustc"
	"forklift/Rpc"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// spyStorage records Upload calls and otherwise behaves like NullStorage.
type spyStorage struct {
	Storages.IStorage
	mu          sync.Mutex
	uploadCount int
}

func (s *spyStorage) Upload(_ string, _ io.Reader, _ map[string]*string) (*Storages.UploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploadCount++
	return &Storages.UploadResult{}, nil
}

func (s *spyStorage) Download(string) (*Storages.DownloadResult, error) {
	return nil, nil
}

func (s *spyStorage) GetMetadata(_ string) (map[string]*string, bool) {
	return nil, false
}

func (s *spyStorage) UploadCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uploadCount
}

// TestUploader_TempArtifactSkipsItemAndDrains is the regression for the
// `return`->`continue` fix in upload(). The old code returned from the
// worker on the first temp-artifact, leaking WaitGroup.Done() and
// deadlocking Wait(); the new code skips just the current item and
// continues to drain the channel.
//
// The skip predicate (strings.Contains "tmp/" or "/var/folders/") is
// path-separator sensitive and only fires on Unix-likes, so this test
// is Unix-only by design.
func TestUploader_TempArtifactSkipsItemAndDrains(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("upload skip predicate uses Unix-style path separators; not exercised on Windows")
	}

	workDir := t.TempDir()
	forkliftDir := filepath.Join(workDir, "target", "forklift")
	if err := os.MkdirAll(forkliftDir, 0755); err != nil {
		t.Fatal(err)
	}

	const cachePackageName = "skipme_deadbeef"
	artifactBytes, _ := json.Marshal(Rustc.Artifact{Artifact: "tmp/foo.rlib"})
	stderrPath := filepath.Join(forkliftDir, cachePackageName+"-stderr")
	if err := os.WriteFile(stderrPath, append(artifactBytes, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	spy := &spyStorage{}
	uploader := Rpc.NewUploader(workDir, spy, &Compressors.NoneCompressor{})
	queue := make(chan Models.CacheItem, 1)
	queue <- Models.CacheItem{
		Name:                    "skipme",
		CachePackageName:        cachePackageName,
		CrateExternDepsChecksum: "0",
	}
	close(queue)

	uploader.Start(queue, 1)

	done := make(chan struct{})
	go func() {
		uploader.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("uploader.Wait() deadlocked — Done() was not called for the skipped item (WaitGroup leak)")
	}

	if got := spy.UploadCount(); got != 0 {
		t.Errorf("expected no uploads for skipped item, got %d", got)
	}
}
