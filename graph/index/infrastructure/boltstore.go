package infrastructure

import (
	"bytes"
	"encoding/gob"
	"fmt"

	bolt "go.etcd.io/bbolt"

	"github.com/kdeps/kartographer/graph/index/domain"
)

//nolint:gochecknoglobals // bbolt bucket names, static byte slices
var (
	bucketFiles  = []byte("files")
	bucketTopics = []byte("topics")
	bucketMeta   = []byte("meta")
)

const defaultDBMode = 0o600

// BoltIndexStore is a bbolt-backed domain.IndexStore.
type BoltIndexStore struct {
	db *bolt.DB
}

// NewBoltIndexStore opens or creates a bbolt database at dbPath.
// The caller must call Close() when done.
func NewBoltIndexStore(dbPath string) (*BoltIndexStore, error) {
	db, err := bolt.Open(dbPath, defaultDBMode, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketFiles, bucketTopics, bucketMeta} {
			if _, createErr := tx.CreateBucketIfNotExists(b); createErr != nil {
				return createErr
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init bbolt buckets: %w", err)
	}

	return &BoltIndexStore{db: db}, nil
}

func (s *BoltIndexStore) Close() error {
	return s.db.Close()
}

// PutFile stores a file record and keeps the topic inverted index in sync.
func (s *BoltIndexStore) PutFile(rec *domain.FileRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		prevTopics, err := s.readPrevTopics(tx, rec.Path)
		if err != nil {
			return err
		}

		if putErr := putGob(tx.Bucket(bucketFiles), []byte(rec.Path), rec); putErr != nil {
			return putErr
		}

		return s.syncTopicIndex(tx, rec.Path, prevTopics, rec.Topics)
	})
}

func (s *BoltIndexStore) readPrevTopics(tx *bolt.Tx, path string) ([]string, error) {
	raw := tx.Bucket(bucketFiles).Get([]byte(path))
	if raw == nil {
		return nil, nil
	}
	var prev domain.FileRecord
	if err := decodeGob(raw, &prev); err != nil {
		return nil, err
	}
	return prev.Topics, nil
}

func (s *BoltIndexStore) syncTopicIndex(tx *bolt.Tx, path string, prevTopics, newTopics []string) error {
	topicsB := tx.Bucket(bucketTopics)

	for _, topic := range prevTopics {
		if !containsStr(newTopics, topic) {
			if err := removeFromTopic(topicsB, topic, path); err != nil {
				return err
			}
		}
	}
	for _, topic := range newTopics {
		if err := addToTopic(topicsB, topic, path); err != nil {
			return err
		}
	}
	return nil
}

func (s *BoltIndexStore) GetFile(path string) (*domain.FileRecord, bool, error) {
	var rec domain.FileRecord
	found := false
	err := s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(bucketFiles).Get([]byte(path))
		if raw == nil {
			return nil
		}
		found = true
		return decodeGob(raw, &rec)
	})
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &rec, true, nil
}

func (s *BoltIndexStore) AllFiles() ([]*domain.FileRecord, error) {
	var recs []*domain.FileRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketFiles).ForEach(func(_, raw []byte) error {
			var rec domain.FileRecord
			if err := decodeGob(raw, &rec); err != nil {
				return err
			}
			recs = append(recs, &rec)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return recs, nil
}

func (s *BoltIndexStore) FilesByTopic(topic string) ([]string, error) {
	var files []string
	err := s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(bucketTopics).Get([]byte(topic))
		if raw == nil {
			return nil
		}
		return decodeGob(raw, &files)
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func addToTopic(topicsB *bolt.Bucket, topic, path string) error {
	var files []string
	if raw := topicsB.Get([]byte(topic)); raw != nil {
		if err := decodeGob(raw, &files); err != nil {
			return err
		}
	}
	if containsStr(files, path) {
		return nil
	}
	files = append(files, path)
	return putGob(topicsB, []byte(topic), files)
}

func removeFromTopic(topicsB *bolt.Bucket, topic, path string) error {
	raw := topicsB.Get([]byte(topic))
	if raw == nil {
		return nil
	}
	var files []string
	if err := decodeGob(raw, &files); err != nil {
		return err
	}
	remaining := files[:0]
	for _, f := range files {
		if f != path {
			remaining = append(remaining, f)
		}
	}
	if len(remaining) == 0 {
		return topicsB.Delete([]byte(topic))
	}
	return putGob(topicsB, []byte(topic), remaining)
}

func containsStr(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func putGob(b *bolt.Bucket, key []byte, value interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return b.Put(key, buf.Bytes())
}

func decodeGob(raw []byte, value interface{}) error {
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(value); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}
