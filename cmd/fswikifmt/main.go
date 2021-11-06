package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/entooone/go-fswiki"
)

var (
	write = flag.Bool("w", false, "write result to source file instead of stdout")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: fswikifmt [-w] FSWIKI_FILE\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	doc, err := fswiki.FormatDocument(f)
	if err != nil {
		return err
	}

	if *write {
		nf, err := os.Create(args[0])
		if err != nil {
			return err
		}
		defer nf.Close()

		nf.Write(doc)
	} else {
		os.Stdout.Write(doc)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
