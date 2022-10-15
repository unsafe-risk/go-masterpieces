package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dgraph-io/badger/v3"
)

const CurrentDBVersion = 100

//go:generate go run github.com/tinylib/msgp

type DBConfig struct {
	Version uint64
}

func GetConfig() (DBConfig, error) {
	var config DBConfig
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(CONFIG_PREFIX + "database"))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			_, err := config.UnmarshalMsg(val)
			return err
		})
		return err
	})
	return config, err
}

func migrate() {
	config, err := GetConfig()
	if err != nil {
		log.Println("Failed to get config:", err)
		var yn string
		log.Println("Want to migrate from 0? (y/n)")
		fmt.Scanln(&yn)
		if yn != "y" {
			log.Println("Abort")
			os.Exit(1)
		}
	}
	if config.Version < CurrentDBVersion {
		// migrate
		for i := range migrations {
			if config.Version < migrations[i].FromVersion {
				// Skip Old migrations
				continue
			}

			if config.Version == migrations[i].FromVersion {
				// Run this migration
				log.Println("Running migration", migrations[i].FromVersion, "to", migrations[i].ToVersion)
				migrations[i].Func()
				config.Version = migrations[i].ToVersion

				// Update config
				err = db.Update(func(txn *badger.Txn) error {
					data, err := config.MarshalMsg(nil)
					if err != nil {
						return err
					}

					return txn.Set([]byte(CONFIG_PREFIX+"database"), data)
				})
				if err != nil {
					log.Println("Failed to update config:", err)
					os.Exit(1)
				}
			}
		}
	}
}

type Migration struct {
	FromVersion uint64
	ToVersion   uint64
	Func        func() error
}

func migrate_0_100() error {
	err := db.Update(func(txn *badger.Txn) error {
		// Get all libraries
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(LIBRARY_PREFIX)); it.ValidForPrefix([]byte(LIBRARY_PREFIX)); it.Next() {
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

			// Update library Tags
			if len(library.Tags) == 0 {
				library.Tags = []string{"Go"}

				// Update tag:Go
				var Go Tag
				tagdata, err := txn.Get([]byte(TAG_PREFIX + "Go"))
				if err != nil {
					return err
				}
				err = tagdata.Value(func(val []byte) error {
					_, err := Go.UnmarshalMsg(val)
					return err
				})
				if err != nil {
					return err
				}

				// Check if library is already in tag
				found := false
				for _, lib := range Go.Libraries {
					if lib == library.ID {
						found = true
						break
					}
				}
				if !found {
					Go.Libraries = append(Go.Libraries, library.ID)
				}

				// Update tag
				data, err := Go.MarshalMsg(nil)
				if err != nil {
					return err
				}
				err = txn.Set([]byte(TAG_PREFIX+"Go"), data)
				if err != nil {
					return err
				}
			}

			// Update library
			data, err := library.MarshalMsg(nil)
			if err != nil {
				return err
			}
			err = txn.Set(item.Key(), data)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

var migrations = []Migration{
	{0, 100, migrate_0_100},
}
