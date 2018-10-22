package db

import (
	"encoding/binary"
)

// The Schema will define how to store and retrieve data from the db.
// Currently we store blocks by prefixing `block` to their hash and
// using that as the key to store blocks.
// `block` + hash -> block
//
// We store the crystallized state using the crystallized state lookup key, and
// also the genesis block using the genesis lookup key.
// The canonical head is stored using the canonical head lookup key.

// The fields below define the suffix of keys in the db.
var (
	mainBucket = []byte("main-bucket")

	blockPrefix         = []byte("b-")
	aStatePrefix        = []byte("a-")
	cStatePrefix        = []byte("c-")
	attestationPrefix   = []byte("x-")
	blockByHeightPrefix = []byte("h-")

	chainHeightKey = []byte("chain-height")
)

func blockKey(key []byte) []byte {
	return append(blockPrefix, key...)
}

func aStateKey(key []byte) []byte {
	return append(aStatePrefix, key...)
}

func cStateKey(key []byte) []byte {
	return append(cStatePrefix, key...)
}

func attestationKey(key []byte) []byte {
	return append(attestationPrefix, key...)
}

func blockByHeightKey(key []byte) []byte {
	return append(blockByHeightPrefix, key...)
}

// encodeSlotNumber encodes a slot number as big endian uint64.
func encodeSlotNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
