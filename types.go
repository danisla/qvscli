package main

import "net/http/cookiejar"

const QTSAuthLogin = "/cgi-bin/authLogin.cgi"

const QVSGetMAC = "/qvs/vms/mac"
const QVSGetVMs = "/qvs/vms"

type LoginFile struct {
	Username     string `json:"username"`
	QTSSessionID string `json:"qts_sessionid"`
}

type QVSClient struct {
	QtsURL     string
	LoginFile  string
	SessionID  string
	CookieJar  *cookiejar.Jar
	LoginPath  string
	GetMACPath string
}

type QTSLoginResponse struct {
	Need2SV    int    `xml:"need_2sv"`
	PWStatus   int    `xml:"pw_status"`
	AuthPassed int    `xml:"authPassed"`
	AuthSID    string `xml:"authSid"`
	Username   string `xml:"username"`
}

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
