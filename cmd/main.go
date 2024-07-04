package main

import (
	"bytes"
	"compress/zlib"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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
    writeObjFlag := hashobjectFlag.Bool("w", false, "write the object into the object database")

    lstreeFlag := flag.NewFlagSet("ls-tree", flag.ExitOnError)
    nameOnly := lstreeFlag.Bool("name-only", false, "list only filenames")

    commitTreeFlag := flag.NewFlagSet("commit-tree", flag.ExitOnError)

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

        path := filepath.Join(".git", "objects", sha[:2], sha[2:])

        b, err := os.ReadFile(path)
        if err != nil {
            fmt.Fprintf(os.Stderr, "error on cat-file: %s", err.Error())
            os.Exit(1)
        }

        buf := bytes.NewBuffer(b)

        r, err := zlib.NewReader(buf)
        if err != nil {
            fmt.Fprintf(os.Stderr, "error on cat-file: %s", err.Error())
            os.Exit(1)
        }
        defer r.Close()

        names, err := io.ReadAll(r)
        if err != nil {
            fmt.Fprintf(os.Stderr, "error on cat-file: %s", err.Error())
            os.Exit(1)
        }

        found := false
        for _, name := range names {
            if found {
                fmt.Print(string(name))
            } else {
                found = name == 0
            }
        }

    case "hash-object":
        if err := hashobjectFlag.Parse(os.Args[2:]); err != nil {
            fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <object>\n")
            os.Exit(1)
        }

        if !*writeObjFlag || hashobjectFlag.NArg() < 1 {
            fmt.Fprintf(os.Stderr, "usage: mygit hash-object -w <object>\n")
            os.Exit(1)
        }

        file, _ := os.ReadFile(hashobjectFlag.Arg(0))

        hash := hashObject("blob", file)

        if err := writeObject(hash, file); err != nil {
            fmt.Fprintf(os.Stderr, "failed writing object: %s", err.Error())
            os.Exit(1)
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
        currentDir, _ := os.Getwd()

        treeHash, c := getTreeHash(currentDir)

        if err := writeObject(treeHash, c); err != nil {
            fmt.Fprintf(os.Stderr, "ls-tree error: %v\n", err)
            os.Exit(1)
        }

        fmt.Println(treeHash)

    case "commit-tree":
        if err := commitTreeFlag.Parse(os.Args[2:]); err != nil {
            fmt.Fprintf(os.Stderr, "usage: mygit commit-tree <object> -p <object> -m \"commit message\"\n")
            os.Exit(1)
        }

        if commitTreeFlag.NArg() < 4 {
            fmt.Fprintf(os.Stderr, "usage: mygit commit-tree <object> -p <object> -m \"commit message\"\n")
            os.Exit(1)
        }

        treeSha := commitTreeFlag.Arg(0)
        parent := commitTreeFlag.Arg(2)
        msg := commitTreeFlag.Arg(3)

        author := "wreckitral"
        email := "defhanayasofhiea@gmail.com"
        currentTime := time.Now().Unix()
        timezone, _ := time.Now().Local().Zone()

        commitData := fmt.Sprintf("tree %s\nparent %s\nauthor %s <%s> %s %s\ncommitter %s <%s> %s %s\n\n%s\n",
            treeSha, parent, author, email,
            fmt.Sprint(currentTime), timezone, author, email,
            fmt.Sprint(currentTime), timezone, msg)

        hash := hashObject("commit", []byte(commitData))

        if err := writeObject(hash, []byte(commitData)); err != nil {
            fmt.Println("error cuk")
        }

        fmt.Println(hash)

    default:
	    fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
	    os.Exit(1)
	}
}

