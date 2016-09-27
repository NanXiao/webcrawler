package main

import (
	"fmt"
	"github.com/NanXiao/webcrawler"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please specify the URL.")
		os.Exit(1)
	}

	for _, v := range os.Args[1:] {
		fmt.Printf("\nThe site map of %s is:\n", v)
		fmt.Println(webcrawler.Crawl(v))
	}

}
