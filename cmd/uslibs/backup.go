package main

import (
	"log"
	"os"
	"time"
)

func FormatTime(t time.Time) string {
	return t.Format("Mon_Jan02_15_04_05_MST_2006")
}

func BackupDB() error {
	os.MkdirAll("backup", 0755)
	f, err := os.Create("backup/" + FormatTime(time.Now().UTC()) + ".backup")
	if err != nil {
		return err
	}
	defer f.Close()
	v, err := db.Backup(f, 0)
	if err != nil {
		return err
	}
	log.Println("Backup done, version:", v)
	return nil
}
