package main

import (
	"flag"
	"fmt"
	"github.com/anlaneg/m3u8/parse"
	"os"
	//"github.com/anlaneg/m3u8/dl"
	"github.com/anlaneg/m3u8/tool"
)

var (
	//url          string
	file string
)

func init() {
	//flag.StringVar(&url, "u", "", "M3U8 URL, required")
	flag.StringVar(&file, "f", "", "M3U8 URL files, required")
}

type URLTask struct {
	tool.ConcurrencyRun
}

func (t *URLTask) GetConcurrency() int {
	return 25
}

func (t *URLTask) DoTask(data interface{}) error {
	url, ok := data.(string)
	if !ok {
		return fmt.Errorf("type error")
	}

	result, err := parse.FromURL(url)
	if err != nil {
		return fmt.Errorf("%s,error=%s", url,err.Error())
	}

	fmt.Printf("[ok/%d] %s\n", len(result.M3u8.Segments) /*dl.GenFileName(url)*/, url)
	return nil
}

func main() {
	flag.Parse()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[error]:%s", r)
			os.Exit(0)
		}
	}()

	if file == "" {
		panic("parameter '" + "f" + "' is required")
	}

	urls, err := tool.ReadLines(file)
	if err != nil {
		panic(err.Error())
	}

	urlTask := &URLTask{}
	data := make([]interface{}, len(urls))
	for i, v := range urls {
		data[i] = v
	}
	tool.ConcurrencyTaskRun(urlTask, data)
	return
}
