package main

import (
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v3"
)

const MB = 1 << 20

var db *badger.DB
var IDSeq *badger.Sequence

func initDB(path string) {
	var err error
	db, err = badger.Open(
		badger.DefaultOptions(path).WithValueLogFileSize(1 * MB),
	)
	if err != nil {
		panic(err)
	}
	IDSeq, err = db.GetSequence([]byte("id"), 2000)
	if err != nil {
		panic(err)
	}

	migrate()
}

func closeDB() {
	err := IDSeq.Release()
	if err != nil {
		log.Println(err)
	}
	err = db.Flatten(runtime.GOMAXPROCS(-1))
	if err != nil {
		log.Println(err)
	}
	err = db.Sync()
	if err != nil {
		log.Println(err)
	}
	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
}

//go:generate go run github.com/tinylib/msgp
type Library struct {
	ID          uint64
	Name        string
	URL         string
	Description string
	Tags        []string
}

type Tag struct {
	Name      string
	Libraries []uint64
}

var _ = func() int {
	return 0
}()

const LIBRARY_PREFIX = "library:" // library:<id>  -> Library
const URL_INDEX_PREFIX = "url:"   // url:<url>     -> <id>
const NAME_INDEX_PREFIX = "name:" // name:<name>   -> <id>
const TAG_PREFIX = "tag:"         // tag:<name>    -> Tag
const CONFIG_PREFIX = "config:"   // config:<key> -> <value>

func UintToStr(u uint64) string {
	return string(strconv.FormatUint(u, 36))
}

func AddLibrary(name, url, description string, tags ...string) error {
	id, err := IDSeq.Next()
	if err != nil {
		return err
	}

	// Trim tags
	for i, tag := range tags {
		tags[i] = strings.Trim(tag, " ")
		tags[i] = strings.ToTitle(tags[i])
	}

	// Unique tags
	uniqueTags := make(map[string]bool)
	var uniqueTagsSlice []string
	for _, tag := range tags {
		if _, ok := uniqueTags[tag]; !ok {
			uniqueTags[tag] = true
			uniqueTagsSlice = append(uniqueTagsSlice, tag)
		}
	}
	tags = uniqueTagsSlice

	// If tags == []string{""} then tags = []string{"Go"}
	if len(tags) == 1 && tags[0] == "" {
		tags = []string{"Go"}
	}

	library := Library{
		ID:          id,
		Name:        name,
		URL:         url,
		Description: description,
		Tags:        tags,
	}
	data, err := library.MarshalMsg(nil)
	if err != nil {
		return err
	}
	err = db.Update(func(txn *badger.Txn) error {
		// ID index
		err = txn.Set([]byte(LIBRARY_PREFIX+UintToStr(id)), data)
		if err != nil {
			return err
		}
		// URL index
		err = txn.Set([]byte(URL_INDEX_PREFIX+url), []byte(UintToStr(id)))
		if err != nil {
			return err
		}
		// Name index
		err = txn.Set([]byte(NAME_INDEX_PREFIX+name), []byte(UintToStr(id)))
		if err != nil {
			return err
		}
		// Tag index
		for _, tag := range tags {
			t, err := txn.Get([]byte(TAG_PREFIX + tag))
			var tagData Tag
			if err == nil {
				err = t.Value(func(val []byte) error {
					_, err := tagData.UnmarshalMsg(val)
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
			tagData.Name = tag
			tagData.Libraries = append(tagData.Libraries, id)
			data, err := tagData.MarshalMsg(nil)
			if err != nil {
				return err
			}

			err = txn.Set([]byte(TAG_PREFIX+tag), data)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Println("Added library", id)
	return nil
}

func ListLibraries(limit int) ([]Library, error) {
	var libraries []Library
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(LIBRARY_PREFIX)); it.ValidForPrefix([]byte(LIBRARY_PREFIX)); it.Next() {
			if limit > 0 && len(libraries) >= limit {
				break
			}
			item := it.Item()
			var library Library
			err := item.Value(func(val []byte) error {
				_, err := library.UnmarshalMsg(val)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
			libraries = append(libraries, library)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return libraries, nil
}

func DeleteLibrary(id uint64) error {
	err := db.Update(func(txn *badger.Txn) error {
		// Get library
		item, err := txn.Get([]byte(LIBRARY_PREFIX + UintToStr(id)))
		if err != nil {
			return err
		}
		var library Library
		err = item.Value(func(val []byte) error {
			_, err := library.UnmarshalMsg(val)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Delete ID index
		err = txn.Delete([]byte(LIBRARY_PREFIX + UintToStr(id)))
		if err != nil {
			return err
		}
		// Delete URL index
		err = txn.Delete([]byte(URL_INDEX_PREFIX + library.URL))
		if err != nil {
			return err
		}
		// Delete Name index
		err = txn.Delete([]byte(NAME_INDEX_PREFIX + library.Name))
		if err != nil {
			return err
		}

		// Delete Tag index
		for _, tag := range library.Tags {
			t, err := txn.Get([]byte(TAG_PREFIX + tag))
			var tagData Tag
			if err == nil {
				err = t.Value(func(val []byte) error {
					_, err := tagData.UnmarshalMsg(val)
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
			// Remove library from tag
			for i, libraryID := range tagData.Libraries {
				if libraryID == id {
					tagData.Libraries = append(tagData.Libraries[:i], tagData.Libraries[i+1:]...)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
