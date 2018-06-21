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
	req.Header.Set("Referer", c.QtsURL)
	req.Header.Set("X-CSRFToken", c.QVSCSRFToken)

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
	resp, err := c.qvsReq("GET", QVSVMs, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// bodyDebug, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(bodyDebug))
	var vmList ListVMsResponse
	err = json.NewDecoder(resp.Body).Decode(&vmList)
	if err != nil {
		return nil, err
	}

	return vmList.Data, err
}

func (c *QVSClient) VMGet(idOrName string) (VMResponse, error) {
	// Lookup ID from name
	vms, err := c.VMList()
	if err != nil {
		return VMResponse{}, err
	}
	for _, v := range vms {
		if v.Name == idOrName {
			return v, nil
		}
		if string(v.ID) == idOrName {
			return v, nil
		}
	}
	return VMResponse{}, fmt.Errorf("VM with id or name '%s' not found", idOrName)
}

func (c *QVSClient) VMGetID(idOrName string) (string, error) {
	vm, err := c.VMGet(idOrName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", vm.ID), nil
}

func (c *QVSClient) VMDescribe(id string) (interface{}, error) {
	path := fmt.Sprintf("%s/%s", QVSVMs, id)
	fmt.Println(path)
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
	return jsonData["data"], err
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

func (c *QVSClient) VMCreate(name string, description string, osType string, cores int, memGB int, network string, mac string, bootISOPath string, diskImagePath string) error {
	var vm QVSCreateRequest
	vm.Name = name
	vm.Description = description

	if osType == "linux" {
		vm.OSType = "ubuntuzesty"
	} else {
		vm.OSType = "win100"
	}

	vm.Cores = cores
	vm.Memory = memGB * 1024 * 1024 * 1024
	vm.Adapters = []map[string]string{
		{
			"mac":    mac,
			"bridge": network,
			"model":  "virtio",
		},
	}
	vm.CDROMs = []map[string]string{
		{
			"path": bootISOPath,
		},
	}
	vm.Disks = []map[string]string{
		{
			"creating_image": "false",
			"path":           diskImagePath,
		},
	}

	jsonData, _ := json.Marshal(&vm)
	resp, err := c.qvsReq("POST", QVSVMs, string(jsonData))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("failed to create VM, status code was %d, %s", resp.StatusCode, data)
	}

	return nil
}

func (c *QVSClient) VMStart(id string) error {
	resp, err := c.qvsReq("POST", fmt.Sprintf(QVSVMStart, id), "{}")
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create VM, status code was %d, %s", resp.StatusCode, data)
	}

	return nil
}

func (c *QVSClient) VMShutdown(id string, force bool) error {
	pathTpl := QVSVMShutdown
	if force {
		pathTpl = QVSVMForceShutdown
	}
	resp, err := c.qvsReq("POST", fmt.Sprintf(pathTpl, id), "{}")
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to force shutdown VM, status code was %d, %s", resp.StatusCode, data)
	}

	return nil
}

func (c *QVSClient) VMDelete(id string) error {
	resp, err := c.qvsReq("DELETE", fmt.Sprintf("%s/%s", QVSVMs, id), "{}")
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete VM, status code was %d, %s", resp.StatusCode, data)
	}

	return nil
}
