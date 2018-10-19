package db

import (
	"os"
	"path"

	"github.com/boltdb/bolt"
)

// BeaconDB manages the data layer of the beacon chain implementation.
// The exposed methods do not have an opinion of the underlying data engine,
// but instead reflect the beacon chain logic.
// For example, instead of defining get, put, remove
// This defines methods such as getBlock, saveBlocksAndAttestations, etc.
type BeaconDB struct {
	db *bolt.DB
}

// Close closes the underlying leveldb database.
func (db *BeaconDB) Close() error {
	return db.db.Close()
}

func (db *BeaconDB) update(fn func(*bolt.Tx) error) error {
	return db.db.Update(fn)
}

func (db *BeaconDB) view(fn func(*bolt.Tx) error) error {
	return db.db.View(fn)
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucket(bucket); err != nil {
			return err
		}
	}

	return nil
}

// NewDB initializes a new DB. If the genesis block and states do not exist, this method creates it.
func NewDB(dirPath string) (*BeaconDB, error) {
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	datafile := path.Join(dirPath, "beaconchain.db")
	boltDB, err := bolt.Open(datafile, 0600, nil)
	if err != nil {
		return nil, err
	}

	db := &BeaconDB{db: boltDB}

	db.update(func(tx *bolt.Tx) error {
		return createBuckets(tx, mainBucket)
	})

	return db, nil
}
