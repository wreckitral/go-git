package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	 if len(os.Args) < 2 {
	 	fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
	 	os.Exit(1)
	 }

	switch command := os.Args[1]; command {
    case "init":
	    for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
	        if err := os.MkdirAll(dir, 0755); err != nil {
	            fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
	        }
	    }

        headFileContents := []byte("ref: refs/heads/main\n")
        if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
            fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
	    }

	    fmt.Println("Initialized git directory")

    case "cat-file":
        if len(os.Args) < 4 {
            fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object>\n")
            os.Exit(1)
        }

        sha := os.Args[3]

        path := fmt.Sprintf(".git/objects/%s/%s", sha[0:2], sha[2:])

        file, _ := os.Open(path)
        r, _ := zlib.NewReader(io.Reader(file))
        s, _ := io.ReadAll(r)

        parts := strings.Split(string(s), "\x00")
        fmt.Print(parts[1])
        r.Close()

    case "hash-object":
        if len(os.Args) < 4 {
            fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <file>\n")
            os.Exit(1)
        }

        file, _ := os.ReadFile(os.Args[3])
        stats, _ := os.Stat(os.Args[3])

        content := string(file)
        contentAndHeader := fmt.Sprintf("blob %d\x00%s", stats.Size(), content)

        sha := sha1.Sum([]byte(contentAndHeader))
        hash := fmt.Sprintf("%x", sha) // to base 16, with lower-case letters a-f

        blobPath := fmt.Sprintf(".git/objects/%s/%s", hash[:2], hash[2:])

        // write the bytes to this buffer variable
        var buffer bytes.Buffer

        z := zlib.NewWriter(&buffer)

        _, err := z.Write([]byte(contentAndHeader))
        if err != nil {
            fmt.Printf("failed on file compression: %s", err.Error())
        }
        z.Close()

        if err := os.MkdirAll(filepath.Dir(blobPath), os.ModePerm); err != nil {
            fmt.Printf("failed on creating directory: %s", err.Error())
        }

        f, err := os.Create(blobPath)
        if err != nil {
            fmt.Printf("failed on creating file: %s", err.Error())
        }
        defer f.Close()

        // write buffer to the file
        _, err = f.Write(buffer.Bytes())
        if err != nil {
            fmt.Printf("failed on writing file: %s", err.Error())
        }

        fmt.Print(hash)

    default:
	    fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
	    os.Exit(1)
	}
}
