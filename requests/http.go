package requests

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// 基础方法，这里多用于访问webapi，配合上json转换。此方法可以运行但是不算完善。
func httpDo(method string, url string, msg string, headers map[string]string) ([]byte, error) {
	// fmt.Println("----", url, "----")
	client := &http.Client{}
	body := bytes.NewBuffer([]byte(msg))
	req, err := http.NewRequest(method,
		url,
		body)
	if err != nil {
		// handle error
		return nil, err
	}

	for key, value := range headers {
		// req.Header.Set("Content-Type", "application/json;charset=utf-8")
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	resultBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return resultBody, nil
}

// post方式
func HttpDoPost(url string, msg string, headers map[string]string) ([]byte, error) {
	return httpDo("POST", url, msg, headers)
}

// get方式
func HttpDoGet(url string, msg string, headers map[string]string) ([]byte, error) {
	return httpDo("GET", url, msg, headers)
}
