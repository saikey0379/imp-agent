package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const REQ_TIMEOUT = 10

// CallRestAPI 调用restful api
func CallRestAPI(url string, jsonReq interface{}) ([]byte, error) {
	var req *http.Request
	var resp *http.Response
	var err error
	var reqBody []byte

	if reqBody, err = json.Marshal(jsonReq); err != nil {
		return nil, err
	}

	if req, err = http.NewRequest("POST", url, bytes.NewBuffer(reqBody)); err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if resp, err = http.DefaultClient.Do(req); err != nil {
		return nil, err
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("call url: %s failed %s", url, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	return body, err
}

func PostRestApi(url string, reqBody []byte) ([]byte, error) {
	client := &http.Client{Timeout: time.Second * REQ_TIMEOUT}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Add("X-JMS-ORG", "00000000-0000-0000-0000-000000000002")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("call url: %s failed %s", url, err.Error())
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	return body, err
}
