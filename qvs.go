package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

func (c *QVSClient) qvsReq(method string, path string, data string) (*http.Response, error) {
	req, _ := http.NewRequest(method, fmt.Sprintf("%s%s", c.QtsURL, path), bytes.NewBuffer([]byte(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", c.QtsURL)
	req.Header.Set("X-CSRFToken", c.QVSCSRFToken)
	c.reqDebug(req)

	client := &http.Client{
		Jar: c.CookieJar,
	}

	resp, err := client.Do(req)
	c.respDebug(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error making request, HTTP status code: %d", resp.StatusCode)
	}

	var d map[string]interface{}
	body, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(body, &d); err != nil {
		return nil, err
	}
	qvsStatus := d["status"].(float64)
	if qvsStatus != QVSStatusOK && qvsStatus != QVSStatusDeferred {
		return nil, fmt.Errorf("error making request, response status was %d: %v", int(qvsStatus), d["detail"])
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

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
		if fmt.Sprintf("%d", v.ID) == idOrName {
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

func (c *QVSClient) VMCreate(name string, description string, osType string, cores int, memGB int, network string, mac string, bootISOPath string, diskImagePath string, vncPassword string) error {
	var vm QVSCreateRequest
	vm.Name = name
	vm.Description = description

	if osType == "linux" {
		vm.OSType = "ubuntuzesty"
	} else {
		vm.OSType = "win100"
	}

	vm.IsAgentEnabled = true

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
	if vncPassword != "" {
		passwordBase64 := base64.StdEncoding.EncodeToString([]byte(vncPassword))

		vm.Graphics = []QVSCreateGraphicsRequest{
			QVSCreateGraphicsRequest{
				Type:           "vnc",
				EnablePassword: true,
				Password:       passwordBase64,
			},
		}
	}

	jsonData, _ := json.Marshal(&vm)
	_, err := c.qvsReq("POST", QVSVMs, string(jsonData))
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) VMStart(id string) error {
	_, err := c.qvsReq("POST", fmt.Sprintf(QVSVMStart, id), "{}")
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) VMReset(id string) error {
	_, err := c.qvsReq("POST", fmt.Sprintf(QVSVMReset, id), "{}")
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) VMShutdown(id string, force bool) error {
	pathTpl := QVSVMShutdown
	if force {
		pathTpl = QVSVMForceShutdown
	}
	_, err := c.qvsReq("POST", fmt.Sprintf(pathTpl, id), "{}")
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) VMDelete(id string) error {
	_, err := c.qvsReq("DELETE", fmt.Sprintf("%s/%s", QVSVMs, id), "{}")
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) VMDiskSnapshotCreate(vmID, name, snapDir string) (string, error) {
	vm, err := c.VMGet(vmID)
	if err != nil {
		return "", err
	}
	if vm.PowerState != "stop" {
		return "", fmt.Errorf("error, VM must be stopped before creating disk snapshot")
	}
	srcPath := vm.Disks[0].RootPath
	srcBase := filepath.Base(srcPath)
	destDir := filepath.Dir(snapDir)
	destPath := filepath.Join(snapDir, fmt.Sprintf("qvs-snap-%s.img", name))

	qfiles, err := c.ListDir(destDir)
	if err != nil {
		return "", err
	}
	snapBase := filepath.Base(snapDir)
	found := false
	for _, f := range qfiles {
		if f.Filename == snapBase && f.IsFolder == 1 {
			found = true
			break
		}
	}
	if !found {
		if err := c.CreateDir(snapDir); err != nil {
			return "", err
		}
	}
	if err := c.CreateDir(snapDir); err != nil {
		log.Printf("Creating snapshot dir: %s", snapDir)
		return "", err
	}
	tmpDestPath := filepath.Join(snapDir, srcBase)
	if err := c.CopyFile(srcPath, tmpDestPath); err != nil {
		return "", err
	}
	// Rename
	if err := c.RenameFile(snapDir, srcBase, filepath.Base(destPath)); err != nil {
		return "", err
	}

	return destPath, nil
}
