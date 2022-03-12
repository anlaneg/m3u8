package parse

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/anlaneg/m3u8/tool"
)

type Result struct {
	URL  *url.URL
	M3u8 *M3u8
	Keys map[int]string
}

/*解析url*/
func FromURL(link string) (*Result, error) {
	u, err := url.Parse(link)
	if err != nil {
	    /*uri有误*/
		return nil, err
	}
	link = u.String()
	body, err := tool.Get(link)
	if err != nil {
		return nil, fmt.Errorf("request m3u8 URL failed: %s", err.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer body.Close()

	/*执行m3u8内容解析，产生m3u8对象*/
	m3u8, err := parse(body)
	if err != nil {
		return nil, err
	}

    /*playlist不为空，取首个playlist,递归处理*/
	if len(m3u8.MasterPlaylist) != 0 {
		sf := m3u8.MasterPlaylist[0]
		return FromURL(tool.ResolveURL(u, sf.URI))
	}

	/*seg为空，报错*/
	if len(m3u8.Segments) == 0 {
		return nil, errors.New("can not found any TS file description")
	}

	result := &Result{
		URL:  u,/*uri*/
		M3u8: m3u8,/*m3u8对象*/
		Keys: make(map[int]string),/*对应的所有key*/
	}

    /*遍历收集的所有key*/
	for idx, key := range m3u8.Keys {
		switch {
		case key.Method == "" || key.Method == CryptMethodNONE:
		    /*不加密，跳过key获取*/
			continue
		case key.Method == CryptMethodAES:
			// Request URL to extract decryption key
			keyURL := key.URI
			keyURL = tool.ResolveURL(u, keyURL)
			resp, err := tool.Get(keyURL)
			if err != nil {
				return nil, fmt.Errorf("extract key failed: %s", err.Error())
			}
			keyByte, err := ioutil.ReadAll(resp)
			_ = resp.Close()
			if err != nil {
				return nil, err
			}
			/*记录当前对应的key*/
			fmt.Println("decryption key: ", string(keyByte))
			result.Keys[idx] = string(keyByte)
		default:
			return nil, fmt.Errorf("unknown or unsupported cryption method: %s", key.Method)
		}
	}
	return result, nil
}
