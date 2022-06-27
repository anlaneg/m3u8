package tool

import (
	"fmt"
	"os"
	"testing"
)

const (
	DB_PATH = "abc.db"
)

func openDB(t *testing.T) *UrlDB {
	url_db, err := OpenUrlDB(DB_PATH)
	if err != nil {
		t.Error(err)
	}
	return url_db
}

func TestOpenUrlDB(t *testing.T) {
	url_db := openDB(t)

	_, err := os.Stat(DB_PATH)
	if err != nil {
		t.Error(err)
	}

	url_db.Close()

	_, err = os.Stat(DB_PATH)
	if err != nil {
		t.Error(err)
	}
}

func testAdd(t *testing.T, key string,
	url string) {
	url_db := openDB(t)
	defer func() {
		url_db.Close()
		os.Remove(DB_PATH)
	}()

	type funcStruct struct {
		addUrl   func(url string) error
		listUrls func() ([]string, error)
		has      func(url string) (bool, error)
		delete   func(url string) error
	}
	funcMap := map[string]funcStruct{
		"url": funcStruct{
			url_db.AddUrl,
			url_db.ListUrls,
			url_db.IsHaveUrl,
			url_db.DeleteUrl,
		},
		"success": funcStruct{
			url_db.AddSuccessUrl,
			url_db.ListSuccessUrls,
			url_db.IsSuccessUrl,
			url_db.DeleteSuccessUrl,
		},
		"failed": funcStruct{
			url_db.AddFailedUrl,
			url_db.ListFailedUrls,
			url_db.IsFailedUrl,
			url_db.DeleteFailedUrl,
		},
	}

	funcMap[key].addUrl(url)

	urls, err := funcMap[key].listUrls()
	if err != nil {
		t.Fatalf("%s list failed,err=%s", key, err)
	}

	if len(urls) != 1 {
		for idx, elem := range urls {
			fmt.Printf("urls[%d]=%s\n", idx, elem)
		}
		t.Fatalf("%s list failed,result length =%d", key, len(urls))
	}

	if urls[0] != url {
		t.Fatalf("%s list failed,result miss(%s)", key, urls[0])
	}

	has, err := funcMap[key].has(url)
	if err != nil {
		t.Fatalf("%s looup failed,err=%s", key, err)
	}

	if !has {
		t.Fatalf("%s looup failed,not exist", key)
	}

	err = funcMap[key].delete(url)
	if err != nil {
		t.Fatalf("%s delete url failed,err=%s", key, err)
	}

	has, err = funcMap[key].has(url)
	if err != nil {
		t.Fatalf("%s looup failed,err=%s", key, err)
	}

	if has {
		t.Fatalf("%s looup failed,exist", key)
	}

	//url_db.Close()
}

func TestAddUrl(t *testing.T) {
	url := "https://www.baidu.com"
	testAdd(t, "url", url)
}

func TestAddFailedUrl(t *testing.T) {
	url := "https://www.google.com"
	testAdd(t, "failed", url)
}

func TestAddSuccessUrl(t *testing.T) {
	url := "https://www.test.com"
	testAdd(t, "success", url)
}
