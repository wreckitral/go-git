package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Entry struct {
	mode string
	name string
	hash []byte
}

func collectEntries(dir string) ([]Entry, error) {
	var entries []Entry
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.Name() == ".git" {
			continue
		}

		mode := "100644"
		if file.IsDir() {
			mode = "40000"
			subEntries, err := writeTree(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			sha, _ := hex.DecodeString(subEntries)
			entries = append(entries, Entry{mode, file.Name(), sha})
		} else {
			content, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			blobHash := hashObject("blob", content)
			sha, _ := hex.DecodeString(blobHash)
			entries = append(entries, Entry{mode, file.Name(), sha})
		}
	}

    sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

func formatTree(entries []Entry) []byte {
	var buf bytes.Buffer
	for _, entry := range entries {
		fmt.Fprintf(&buf, "%s %s\x00", entry.mode, entry.name)
		buf.Write(entry.hash)
	}
	return buf.Bytes()
}

func hashObject(objectType string, content []byte) string {
	header := fmt.Sprintf("%s %d\x00", objectType, len(content))
	store := append([]byte(header), content...)

	hash := sha1.New()
	hash.Write(store)
	return hex.EncodeToString(hash.Sum(nil))
}

func writeObject(hash string, content []byte) error {
	dir := filepath.Join(".git", "objects", hash[:2])
	file := filepath.Join(dir, hash[2:])

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var buf bytes.Buffer
	zlibWriter := zlib.NewWriter(&buf)
	defer zlibWriter.Close()
	if _, err := zlibWriter.Write(content); err != nil {
		return err
	}

	return os.WriteFile(file, buf.Bytes(), 0644)
}
