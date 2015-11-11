// Package store is a content addressable file store
// TODO(tcm): Abstract out file operations to allow alternate backing stores
package store

import (
	"crypto/sha1"
	"errors"
	"hash"
	"io"
)

// Walker is used to enumerate a store. Given a StoreID as an
// argument, it returns all the StoreIDs that the contents of the
// input store item may refer to
type Walker func(ID) []ID

// Storer is an interface to a content addressable blob store. Files
// can be written to it, and then accesed and referred to based on an
// ID representing the content of the written item.
type Storer interface {
	Store() (WriteCloser, error) // Write something to the store

	Open(ID) (io.ReadCloser, error) // Open a file by id
	Size(ID) (int64, error)         // Open a file by id
	Link(ID, ...string) error       // Link a file id to a given location
	UnLink(ID) error                // UnLink a blob
	ForEach(f func(ID))             // Call function for each ID

	EmptyFileID() ID       // Return the StoreID for an 0 byte object
	IsEmptyFileID(ID) bool // Compares the ID to the stores empty ID

	SetRef(name string, id ID) error // Set a reference
	GetRef(name string) (ID, error)  // Get a reference
	DeleteRef(name string) error     // Delete a reference
	ListRefs() map[string]ID         // Get a reference
}

type hashStore struct {
	Hasher
	tempDir     string
	baseDir     string
	prefixDepth int
}

func (t *hashStore) Store() (WriteCloser, error) {
	file, err := TempFile(t.tempDir, "blob")
	if err != nil {
		return nil, err
	}

	h := t.NewHash()
	mwriter := io.MultiWriter(file, h)

	writer := &hashedStoreWriter{
		file:   file,
		hasher: h,
		writer: mwriter,
	}

	return writer, nil
}

func CopyToStore(d Storer, r io.ReadCloser) (id ID, err error) {
	w, err := d.Store()
	if err != nil {
		return nil, errors.New("during copy to store, " + err.Error())
	}

	if _, err = io.Copy(w, r); err != nil {
		return nil, errors.New("during copy to store, " + err.Error())
	}

	if r.Close() != nil {
		return nil, errors.New("during copy to store, " + err.Error())
	}

	if w.Close() != nil {
		return nil, errors.New("during copy to store, " + err.Error())
	}

	return w.Identity()
}

// New creates a blob store that uses  hex encoded hash strings of
// ingested blobs for IDs, using the provided hash function
func New(
	baseDir string, // Base directory of the persistant store
	tempDir string, // Temporary directory for ingesting files
	prefixDepth int, // How many chars to use for directory prefixes
	hf func() hash.Hash,
) Storer {
	store := &hashStore{
		Hasher:      NewHasher(hf),
		tempDir:     tempDir,
		baseDir:     baseDir,
		prefixDepth: prefixDepth,
	}

	return store
}

// Sha1Store creates a blob store that uses  hex encoded sha1 strings of
// ingested blobs for IDs
func Sha1Store(
	baseDir string, // Base directory of the persistant store
	tempDir string, // Temporary directory for ingesting files
	prefixDepth int, // How many chars to use for directory prefixes
) Storer {
	return New(
		tempDir,
		baseDir,
		prefixDepth,
		sha1.New)
}
