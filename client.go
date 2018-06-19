package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/howeyc/gopass"
)

type LoginFile struct {
	SessionID string `json:"sessionid"`
}

type QVSClient struct {
	URL        string
	LoginFile  string
	SessionID  string
	CookieJar  *cookiejar.Jar
	LoginPath  string
	GetMACPath string
}

const QVSAuthLogin = "/auth/login"
const QVSGetMAC = "/vms/mac"
const QVSGetVMs = "/vms"

type CreateMACResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

type ListVMsResponse struct {
	Status int          `json:"status"`
	Data   []VMResponse `json:"data"`
}

type VMResponse struct {
	ID         int    `json:"id"`
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Cores      int    `json:"cores"`
	PowerState string `json:"power_state"`
}

func NewQVSClient(qvsURL string, loginFile string) *QVSClient {
	c := &QVSClient{
		URL:       strings.TrimSpace(qvsURL),
		LoginFile: strings.TrimSpace(loginFile),
	}

	err := c.loadCookieFromFile()
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

	data := fmt.Sprintf("{\"username\":\"%s\",\"password\":\"%s\"}", strings.TrimSpace(username), base64.StdEncoding.EncodeToString(password))

	authURL := fmt.Sprintf("%s%s", c.URL, QVSAuthLogin)

	req, _ := http.NewRequest("POST", authURL, bytes.NewBuffer([]byte(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Jar: c.CookieJar,
	}

	_, err := client.Do(req)
	if err != nil {
		return err
	}

	sessionid := c.getSessionIDFromCookie()
	if sessionid == "" {
		return fmt.Errorf("error, login failed")
	}

	f, err := os.OpenFile(c.LoginFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error, failed to open login file '%s' for writting: %v", c.LoginFile, err)
	}

	// Persist sessionid
	var lf LoginFile
	lf.SessionID = sessionid
	lfData, _ := json.Marshal(&lf)
	f.Write(lfData)
	f.Close()

	return nil
}

func (c *QVSClient) getSessionIDFromCookie() string {
	u, _ := url.Parse(c.URL)
	cookies := c.CookieJar.Cookies(u)
	if len(cookies) == 0 {
		return ""
	}

	for _, c := range cookies {
		if c.Name == "sessionid" {
			return c.Value
		}
	}
	return ""
}

func (c *QVSClient) loadCookieFromFile() error {
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

	var cookies []*http.Cookie
	u, _ := url.Parse(c.URL)
	cookie := &http.Cookie{
		Name:  "sessionid",
		Value: lf.SessionID,
	}
	cookies = append(cookies, cookie)
	c.CookieJar.SetCookies(u, cookies)

	return nil
}

func (c *QVSClient) qvsReq(method string, path string, data string) (*http.Response, error) {
	if c.getSessionIDFromCookie() == "" {
		return nil, fmt.Errorf("not logged in, run 'qvscli login'")
	}

	req, _ := http.NewRequest(method, fmt.Sprintf("%s%s", c.URL, path), bytes.NewBuffer([]byte(data)))
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
