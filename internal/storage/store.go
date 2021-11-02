package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/flipkart-incubator/dkv/internal/stats"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flipkart-incubator/dkv/pkg/serverpb"
)

type Stat struct {
	RequestLatency *prometheus.SummaryVec
	ResponseError  *prometheus.CounterVec
}

func NewStat(registry prometheus.Registerer, engine string) *Stat {
	RequestLatency := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  stats.Namespace,
		Name:       fmt.Sprintf("storage_latency_%s", engine),
		Help:       fmt.Sprintf("Latency statistics for %s storage operations", engine),
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		MaxAge:     10 * time.Second,
	}, []string{stats.Ops})
	ResponseError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: stats.Namespace,
		Name:      fmt.Sprintf("storage_error_%s", engine),
		Help:      fmt.Sprintf("Error count for %s storage operations", engine),
	}, []string{stats.Ops})
	registry.MustRegister(RequestLatency, ResponseError)
	return &Stat{RequestLatency, ResponseError}
}

type MemoryUsage interface{}

// A KVStore represents the key value store that provides
// the underlying storage implementation for the various
// DKV operations.
type KVStore interface {
	io.Closer
	// Put stores the association between the given key and value and
	// optionally sets the expireTS of the key to the provided epoch in seconds
	Put(pairs ...*serverpb.KVPair) error
	// Get bulk fetches the associated values for the given keys.
	// Note that during partial failures, any successful results
	// are discarded and an error is returned instead.
	Get(keys ...[]byte) ([]*serverpb.KVPair, error)
	// Delete deletes the given key.
	Delete(key []byte) error
	// GetSnapshot retrieves the entire keyspace representation
	// with latest value against every key.
	GetSnapshot() (io.ReadCloser, error)
	// PutSnapshot ingests the given keyspace representation wholly
	// into the current state. Any existing state will be discarded
	// or replaced with the given state.
	PutSnapshot(io.ReadCloser) error
	// Iterate iterates through the entire keyspace in no particular
	// order. IterationOptions can be used to control where to begin
	// iteration as well as what keys are iterated by their prefix.
	Iterate(IterationOptions) Iterator
	// CompareAndSet compares the current value of the given key with
	// that of the given value. In case of a match, it updates that
	// key with the new value and returns true. Else, it returns false.
	// All this is done atomically from the caller's point of view and
	// hence is safe from a concurrency perspective.
	// If the expected value is `nil`, then the key is created and
	// initialized with the given value, atomically.
	CompareAndSet(key, expect, update []byte) (bool, error)

	GetMemoryUsage() (MemoryUsage, error)
}

// A Backupable represents the capability of the underlying store
// to be backed up and restored using filesystem as the medium.
type Backupable interface {
	// BackupTo backs up the entire state of the underlying store
	// as one or more files into the given `path`.
	// Note that it is upto the implementation to interpret the
	// provided path as a file or a folder.
	BackupTo(path string) error
	// RestoreFrom restores the entire state of the underlying store
	// from one or more files belonging to the given `path`. Typically
	// these files must have been generated by a previous invocation
	// of the `BackupTo` method using this `path` by the same
	// implementation of this interface.
	// Note that it is upto the implementation to interpret the
	// provided path as a file or a folder.
	// Returns the various traits of the newly restored store upon
	// successful restoration, else an error
	RestoreFrom(path string) (KVStore, Backupable, ChangePropagator, ChangeApplier, error)
}

// A ChangePropagator represents the capability of the underlying
// store from which committed changes can be retrieved for replication
// purposes. The implementor of this interface assumes the role of a
// master node in a typical master-slave setup.
type ChangePropagator interface {
	// GetLatestCommittedChangeNumber retrieves the change number of
	// the latest committed change. Returns an error if unable to
	// load this number.
	GetLatestCommittedChangeNumber() (uint64, error)
	// LoadChanges retrieves all the changes committed since the given
	// `fromChangeNumber`. Also, `maxChanges` can be used to limit the
	// number of changes returned in the response.
	LoadChanges(fromChangeNumber uint64, maxChanges int) ([]*serverpb.ChangeRecord, error)
}

// A ChangeApplier represents the capability of the underlying store
// to apply changes directly onto its key space. This is typically
// used for replication purposes to indicate that the implementor
// assumes the role of a slave node in a master-slave setup.
type ChangeApplier interface {
	// GetLatestAppliedChangeNumber retrieves the change number of
	// the latest committed change applied. Returns an error if unable
	// to load this number.
	GetLatestAppliedChangeNumber() (uint64, error)
	// SaveChanges commits to local key space the given changes and
	// returns the change number of the last committed change along
	// with an error that might have occurred during the process.
	// Note that implementors must treat every change on its own and
	// return the first error that occurs during the process. Remaining
	// changes if any must NOT be applied in order to ensure sequential
	// consistency.
	SaveChanges(changes []*serverpb.ChangeRecord) (uint64, error)
}

// TODO: Following functions should be moved to a util layer ?

const timeFormatTempPath = "20060102150405"

// CreateTempFile creates a temporary folder with the given prefix.
// It attempts to also appends a timestamp to the given prefix so as
// to better avoid collisions. Under the hood, it delegates to the
// GoLang API for temporary folder creation.
func CreateTempFile(dir string, prefix string) (*os.File, error) {
	tempFilePrefix := time.Now().AppendFormat([]byte(prefix), timeFormatTempPath)
	tempFile, err := ioutil.TempFile(dir, string(tempFilePrefix))
	if err != nil {
		return nil, err
	}
	return tempFile, nil
}

// CreateTempFolder creates a temporary folder with the given prefix.
// It attempts to also appends a timestamp to the given prefix so as
// to better avoid collisions. Under the hood, it delegates to the
// GoLang API for temporary folder creation.
func CreateTempFolder(dir string, prefix string) (string, error) {
	tempFolderPrefix := time.Now().AppendFormat([]byte(prefix), timeFormatTempPath)
	return ioutil.TempDir(dir, string(tempFolderPrefix))
}

// RenameFolder moves the given src path onto the given dst path by
// first removing the dst path and then performing the actual movement.
func RenameFolder(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return os.Rename(src, dst)
}
