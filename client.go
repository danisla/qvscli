package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/howeyc/gopass"
)

func NewQVSClient(qtsURL string, loginFile string) *QVSClient {
	c := &QVSClient{
		QtsURL:    strings.TrimSpace(qtsURL),
		LoginFile: strings.TrimSpace(loginFile),
	}

	err := c.loadQTSCookieFromFile()
	if err != nil {
		c.CookieJar, _ = cookiejar.New(nil)
	}

	return c
}

func (c *QVSClient) Login() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Printf("Enter Password: ")
	password, _ := gopass.GetPasswd()

	username = strings.TrimSpace(username)

	if username == "" || string(password) == "" {
		return fmt.Errorf("no username and/or password provided.")
	}

	passwordBase64 := base64.StdEncoding.EncodeToString(password)

	if err := c.QTSLogin(username, passwordBase64, ""); err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) QTSLogin(username string, password string, securityCode string) error {
	params := fmt.Sprintf("user=%s&pwd=%s&serviceKey=1&security_code=%s", username, password, securityCode)

	authURL := fmt.Sprintf("%s%s", c.QtsURL, QTSAuthLogin)

	req, _ := http.NewRequest("POST", authURL, bytes.NewBuffer([]byte(params)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Jar: c.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dataBytes, _ := ioutil.ReadAll(resp.Body)

	var login QTSLoginResponse
	if err := xml.Unmarshal(dataBytes, &login); err != nil {
		return err
	}

	if login.AuthPassed == 0 {
		if login.Need2SV == 1 {
			// Get security code
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Enter Security Code: ")
			securityCode, _ := reader.ReadString('\n')
			securityCode = strings.TrimSpace(securityCode)

			// Retry request
			return c.QTSLogin(username, password, securityCode)

		} else {
			return fmt.Errorf("invalid credentials")
		}
	}

	f, err := os.OpenFile(c.LoginFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error, failed to open login file '%s' for writting: %v", c.LoginFile, err)
	}

	// Persist user and session id
	var lf LoginFile
	lf.Username = login.Username
	lf.QTSSessionID = login.AuthSID
	lfData, _ := json.MarshalIndent(&lf, "", "  ")
	f.Write(lfData)
	f.Close()

	return nil
}

func (c *QVSClient) getQTSSessionIDFromCookie() string {
	u, _ := url.Parse(c.QtsURL)
	cookies := c.CookieJar.Cookies(u)
	if len(cookies) == 0 {
		return ""
	}

	for _, c := range cookies {
		if c.Name == "NAS_SID" {
			return c.Value
		}
	}
	return ""
}

func (c *QVSClient) loadQTSCookieFromFile() error {
	raw, err := ioutil.ReadFile(c.LoginFile)
	if err != nil {
		return err
	}

	var lf LoginFile
	err = json.Unmarshal(raw, &lf)
	if err != nil {
		return err
	}

	cookieJar, _ := cookiejar.New(nil)
	c.CookieJar = cookieJar
	u, _ := url.Parse(c.QtsURL)
	c.CookieJar.SetCookies(u, []*http.Cookie{
		&http.Cookie{Name: "NAS_USER", Value: lf.Username},
		&http.Cookie{Name: "NAS_SID", Value: lf.QTSSessionID},
	})

	return nil
}

func (c *QVSClient) qvsReq(method string, path string, data string) (*http.Response, error) {
	if c.getQTSSessionIDFromCookie() == "" {
		return nil, fmt.Errorf("not logged in, run 'qvscli login'")
	}

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
