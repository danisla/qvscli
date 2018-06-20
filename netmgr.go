package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *QVSClient) NetMgrList() ([]NetMgrNet, error) {
	reqURL := fmt.Sprintf("%s%s/list?sid=%s", c.QtsURL, QTSNetManager, c.SessionID)

	req, _ := http.NewRequest("GET", reqURL, nil)

	client := &http.Client{
		Jar: c.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error making request, HTTP status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	var networks []NetMgrNet
	json.Unmarshal(data, &networks)

	return networks, nil
}
