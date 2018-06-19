package main

import "net/http/cookiejar"

const QVSAuthLogin = "/auth/login"
const QVSGetMAC = "/vms/mac"
const QVSGetVMs = "/vms"

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
