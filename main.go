package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/anlaneg/m3u8/dl"
)

var (
	url      string
	output   string
	chanSize int
)

func init() {
	flag.StringVar(&url, "u", "", "M3U8 URL, required")
	flag.IntVar(&chanSize, "c", 25, "Maximum number of occurrences")
	flag.StringVar(&output, "o", "", "Output folder, required")
}

func main() {
    /*命令行解析*/
	flag.Parse()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[error]", r)
			os.Exit(-1)
		}
	}()
	if url == "" {
		panicParameter("u")
	}
	if output == "" {
		panicParameter("o")
	}
	if chanSize <= 0 {
		panic("parameter 'c' must be greater than 0")
	}

	/*创建 downloader task*/
	downloader, err := dl.NewTask(output, url)
	if err != nil {
		panic(err)
	}
	
	/*执行download task*/
	if err := downloader.Start(chanSize); err != nil {
		panic(err)
	}
	fmt.Println("Done!")
}

func panicParameter(name string) {
	panic("parameter '" + name + "' is required")
}
