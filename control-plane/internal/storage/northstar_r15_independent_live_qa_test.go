package storage

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Agent-Field/agentfield/control-plane/pkg/types"
)

type r15IndependentIdempotencyAPI interface {
	LookupAsyncExecutionIdempotency(context.Context, string, string, string, string) (*AsyncExecutionIdempotencyResult, error)
	CreateAsyncExecutionIdempotent(context.Context, string, string, string, string, *types.Execution) (*AsyncExecutionIdempotencyResult, error)
}

var _ r15IndependentIdempotencyAPI = (*LocalStorage)(nil)
var _ r15IndependentIdempotencyAPI = (AsyncExecutionIdempotencyStore)(nil)

func TestR15NilExecutionRejectedBeforeDatabaseAccess(t *testing.T) {
	var store *LocalStorage
	result, err := store.CreateAsyncExecutionIdempotent(context.Background(), "owner\x00target", "target", "request", "digest", nil)
	if err == nil {
		t.Fatal("nil execution must be rejected before database access")
	}
	if result != nil {
		t.Fatalf("rejected execution returned a binding: %+v", result)
	}
}

func TestR15AuthorizedSourceFingerprints(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the QA source for fingerprinting")
	}
	storageDirectory := filepath.Dir(testFile)
	projectRoot := filepath.Clean(filepath.Join(storageDirectory, "..", ".."))
	paths := []string{
		filepath.Join(storageDirectory, "execution_idempotency.go"),
		testFile,
		filepath.Join(projectRoot, "docs", "architecture", "basefive-r14-contracts.json"),
	}
	for _, sourcePath := range paths {
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read authorized fingerprint target %s: %v", sourcePath, err)
		}
		if len(source) == 0 {
			t.Fatalf("authorized fingerprint target is empty: %s", sourcePath)
		}
		relativePath, err := filepath.Rel(projectRoot, sourcePath)
		if err != nil {
			t.Fatalf("relativize fingerprint target: %v", err)
		}
		t.Logf("R15 authorized-file SHA256 %s %x", filepath.ToSlash(relativePath), sha256.Sum256(source))
	}
}
