package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/anlaneg/m3u8/dl"
)

var (
	url          string
	output       string
	chanSize     int
	continueFlag bool
	maxTries     int
)

func init() {
	flag.StringVar(&url, "u", "", "M3U8 URL, required")
	flag.IntVar(&chanSize, "c", 25, "Maximum number of occurrences")
	flag.StringVar(&output, "o", "", "Output folder, required")
	flag.BoolVar(&continueFlag, "C", true, "continue download")
	flag.IntVar(&maxTries, "m", -1, "Maximum number of try")
}

func main() {
	/*命令行解析*/
	flag.Parse()
	
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[error]", r)
			os.Exit(0)
		}
	}()

	/*参数检查*/
	if url == "" {
		panic("parameter '" + "u" + "' is required")
		os.Exit(0)
	}
	if output == "" {
		panic("parameter '" + "o" + "' is required")
		os.Exit(0)
	}
	if chanSize <= 0 {
		fmt.Println("parameter 'c' must be greater than 0")
		os.Exit(0)
	}
	if maxTries <= 0 {
		maxTries = -1
	}

	/*创建 downloader task*/
	downloader, err := dl.NewTask(output, url)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if downloader.IsExist() {
		fmt.Printf("*****%s****exists\n", downloader.GetFileName())
		os.Exit(0)
	}

	/*执行download task*/
	if err := downloader.Start(chanSize, continueFlag, maxTries); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println("Done!")
}

/*func panicParameter(name string) {
	panic("parameter '" + name + "' is required")
}*/
