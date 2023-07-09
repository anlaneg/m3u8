package tool

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

/*请求url*/
func Get(url string) (io.ReadCloser, error) {
	c := http.Client{
		Timeout: time.Duration(60) * time.Second,
	}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		/*对端返回非200，执行报错*/
		return nil, fmt.Errorf("http error: status code %d", resp.StatusCode)
	}
	/*返回响应内容*/
	return resp.Body, nil
}
