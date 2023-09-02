package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/anlaneg/m3u8/parse"
	"github.com/anlaneg/m3u8/dl"
)

var (
	url          string
)

func init() {
	flag.StringVar(&url, "u", "", "M3U8 URL, required")
}

func main() {
	flag.Parse()
	
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[error]", r)
			os.Exit(0)
		}
	}()

	if url == "" {
		panic("parameter '" + "u" + "' is required")
	}
	
	result, err := parse.FromURL(url)
	if err != nil {
		panic(err.Error())
	}
	
	fmt.Printf("[ok/%d] %s",len(result.M3u8.Segments),dl.GenFileName(url))
	return
}
