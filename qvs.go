package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *QVSClient) qvsReq(method string, path string, data string) (*http.Response, error) {
	req, _ := http.NewRequest(method, fmt.Sprintf("%s%s", c.QtsURL, path), bytes.NewBuffer([]byte(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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

	return resp, nil
}

func (c *QVSClient) MACCreate() (string, error) {
	resp, err := c.qvsReq("GET", QVSGetMAC, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	type CreateMACResponse struct {
		Status int    `json:"status"`
		Data   string `json:"data"`
	}
	var macResp CreateMACResponse
	err = json.NewDecoder(resp.Body).Decode(&macResp)
	if err != nil {
		return "", err
	}

	return macResp.Data, err
}

func (c *QVSClient) VMList() ([]VMResponse, error) {
	resp, err := c.qvsReq("GET", QVSGetVMs, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var vmList ListVMsResponse
	err = json.NewDecoder(resp.Body).Decode(&vmList)
	if err != nil {
		return nil, err
	}

	return vmList.Data, err
}

func (c *QVSClient) VMDescribe(id string) (string, error) {
	path := fmt.Sprintf("%s/%s", QVSGetVMs, id)
	resp, err := c.qvsReq("GET", path, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)

	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)

	status := int(jsonData["status"].(float64))
	if status != 0 {
		return "", fmt.Errorf("error making request, status was: %d", status)
	}

	pretty, _ := json.MarshalIndent(jsonData["data"], "", "  ")
	return string(pretty), err
}

func (c *QVSClient) QVSListNet() ([]QVSNet, error) {
	netMgrNetworks, err := c.NetMgrList()
	if err != nil {
		return nil, err
	}

	var networks []QVSNet

	// Filter networks
	for _, network := range netMgrNetworks {
		if network.DisplayName != "" {
			qvsNet := &QVSNet{
				DisplayName: network.DisplayName,
				Name:        network.VSwitchName,
				IP:          network.VSwitchIP,
				NICs:        []string{network.PhysicalNIC},
			}
			networks = append(networks, *qvsNet)
		}
	}

	return networks, nil
}
