package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"flag"
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

    // flags
    catfileFlag := flag.NewFlagSet("cat-file", flag.ExitOnError)
    prettyPrint := catfileFlag.Bool("p", false, "pretty-print <object> content")

    hashobjectFlag := flag.NewFlagSet("hash-object", flag.ExitOnError)
    writeObject := hashobjectFlag.Bool("w", false, "write the object into the object database")

    lstreeFlag := flag.NewFlagSet("ls-tree", flag.ExitOnError)
    nameOnly := lstreeFlag.Bool("name-only", false, "list only filenames")


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
        if err := catfileFlag.Parse(os.Args[2:]); err != nil {
            fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object>\n")
            os.Exit(1)
        }

        if !*prettyPrint || catfileFlag.NArg() < 1 {
            fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object>\n")
            os.Exit(1)
        }

        sha := catfileFlag.Arg(0)

        path := fmt.Sprintf(".git/objects/%s/%s", sha[:2], sha[2:])

        file, _ := os.Open(path)
        r, _ := zlib.NewReader(io.Reader(file))
        s, _ := io.ReadAll(r)

        parts := strings.Split(string(s), "\x00")
        fmt.Print(parts[1])
        r.Close()

    case "hash-object":
        if err := hashobjectFlag.Parse(os.Args[2:]); err != nil {
            fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <object>\n")
            os.Exit(1)
        }

        if !*writeObject || hashobjectFlag.NArg() < 1 {
            fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <object>\n")
            os.Exit(1)
        }

        file, _ := os.ReadFile(hashobjectFlag.Arg(0))
        stats, _ := os.Stat(hashobjectFlag.Arg(0))

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

    case "ls-tree":
        if err := lstreeFlag.Parse(os.Args[2:]); err != nil {
            fmt.Fprintf(os.Stderr, "usage: mygit ls-tree <object>\n")
            os.Exit(1)
        }

        if lstreeFlag.NArg() < 1 {
            fmt.Fprintf(os.Stderr, "usage: mygit ls-tree <object>\n")
            os.Exit(1)
        }

        sha := lstreeFlag.Arg(0)

        if err := lsTree(sha, *nameOnly); err != nil {
            fmt.Fprintf(os.Stderr, "ls-tree error: %v\n", err)
            os.Exit(1)
        }

    case "write-tree":
        hash, err := writeTree(".")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error writing tree:", err)
			os.Exit(1)
		}
		fmt.Println(hash)

    default:
	    fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
	    os.Exit(1)
	}
}

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

func writeTree(dir string) (string, error) {
	entries, err := collectEntries(dir)
	if err != nil {
		return "", err
	}

	treeContent := formatTree(entries)
	treeHash := hashObject("tree", treeContent)

	err = writeObject(treeHash, treeContent)
	if err != nil {
		return "", err
	}

	return treeHash, nil
}
