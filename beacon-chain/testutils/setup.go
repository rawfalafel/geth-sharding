package testutils

import (
	"os"
	"path"
	"testing"

	"github.com/boltdb/bolt"
)

func SetupDB(t *testing.T) *bolt.DB {
	// TODO: Fix so that db is always created in the same location
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to open dir: %v", err)
	}

	datadir := path.Join(dir, "test")
	if err := os.RemoveAll(datadir); err != nil {
		t.Fatalf("failed to clean dir: %v", err)
	}

	if err := os.MkdirAll(datadir, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	datafile := path.Join(datadir, "test.db")
	boltDB, err := bolt.Open(datafile, 0600, nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	boltDB.NoSync = true

	return boltDB
}