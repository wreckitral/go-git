package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func lsTree(sha string, nameOnly bool) error {
    dir, filename := sha[:2], sha[2:]

    path := fmt.Sprintf(".git/objects/%s/%s", dir, filename)

    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    zlibReader, err := zlib.NewReader(file)
    if err != nil {
        return err
    }
    defer zlibReader.Close()

    data, err := io.ReadAll(zlibReader)
    if err != nil {
        return err
    }

    if !bytes.HasPrefix(data, []byte("tree")) {
        return fmt.Errorf("object is not a tree")
    }

    nullIndex := bytes.IndexByte(data, 0)
    if nullIndex == -1 {
        return fmt.Errorf("invalid tree object")
    }

    treeData := data[nullIndex+1:]
    return parseTree(treeData, nameOnly)
}

func parseTree(data []byte, nameOnly bool) error {
    for len(data) > 0 {
        spaceIndex := bytes.IndexByte(data, ' ') // get the space character index
        if spaceIndex == -1 {
            return fmt.Errorf("invalid tree data")
        }

        mode := data[:spaceIndex] // get the content from before the space character index
        data = data[spaceIndex+1:] // get the content from after the space character index

        nullIndex := bytes.IndexByte(data, 0)
        if nullIndex == -1 {
            return fmt.Errorf("invalid tree data")
        }

        name := data[:nullIndex] // get data before the null index
        data = data[nullIndex+1:] // get data after the null index

        if len(data) < 20 {
            return fmt.Errorf("invalid tree data")
        }

        sha := data[:20] // get the sha value
        data = data[20:] // move data forward pass the sha value

        if nameOnly {
            fmt.Println(string(name))
        } else {
            fmt.Printf("%s %s %s\n", mode, fmt.Sprintf("%x", sha), name)
        }
    }

    return nil
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

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(content); err != nil {
		return err
	}
    w.Close()

	return os.WriteFile(file, buf.Bytes(), 0644)
}

func getTreeHash(dir string) (string, []byte) {
    files, _ := os.ReadDir(dir)

    type entry struct {
        name string
        con []byte
    }

    var entries []entry
    conSize := 0

    for _, file := range files {
        if file.Name() == ".git" {
            continue
        }

        if file.IsDir() {
            b, _ := getTreeHash(filepath.Join(dir, file.Name()))
            s := fmt.Sprintf("40000 %s\x00", file.Name())
            byts := append([]byte(s), b...)
            entries = append(entries, entry{file.Name(), byts})
            conSize += len(byts)
        } else {
            f, _ := os.Open(filepath.Join(dir, file.Name()))
            b, _ := io.ReadAll(f)
            s := fmt.Sprintf("blob %d\x00%s", len(b), string(b))

            sha1 := sha1.New()
            _, err := io.WriteString(sha1, s)
            if err != nil {
                return "", nil
            }

			s = fmt.Sprintf("100644 %s\u0000", file.Name())
			b = append([]byte(s), sha1.Sum(nil)...)
			entries = append(entries, entry{file.Name(), b})
			conSize += len(b)
        }
    }
    sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

    s := fmt.Sprintf("tree %d\u0000", conSize)
    b := []byte(s)
    for _, entry := range entries {
        b = append(b, entry.con...)
    }
    sha1 := sha1.New()
    _, err := io.WriteString(sha1, string(b))
    if err != nil {
        return "", nil
    }
    return hex.EncodeToString(sha1.Sum(nil)), b
}
