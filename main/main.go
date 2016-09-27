package main

import (
	"fmt"
	"github.com/NanXiao/webcrawler"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please specify the URL")
		os.Exit(1)
	}
	fmt.Println(webcrawler.Crawl(os.Args[1]))
}
