package main

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
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
        if len(os.Args) < 3 {
            fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p [<args>...]\n")
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

    default:
	    fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
	    os.Exit(1)
	}
}
