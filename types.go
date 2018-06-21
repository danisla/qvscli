package main

import "net/http/cookiejar"

const QTSAuthLogin = "/cgi-bin/authLogin.cgi"
const QTSFileStation = "/cgi-bin/filemanager/utilRequest.cgi"
const QTSNetManager = "/netmgr/api.fcgi/api/net"

const QVSRoot = "/qvs"
const QVSGetMAC = "/qvs/vms/mac"
const QVSVMs = "/qvs/vms"
const QVSVMStart = "/qvs/vms/%s/start"
const QVSVMForceShutdown = "/qvs/vms/%s/forceshutdown"
const QVSVMShutdown = "/qvs/vms/%s/shutdown"
const QVSVNCTpl = "/qvs/#/console/vms/%s"

type LoginFile struct {
	QtsURL       string `json:"qts_url"`
	Username     string `json:"username"`
	QTSSessionID string `json:"qts_sessionid"`
	QVSCSRFToken string `json:"qvs_csrftoken"`
	QVSSessionID string `json:"qvs_sessionid"`
}

type QVSClient struct {
	QtsURL       string
	LoginFile    string
	SessionID    string
	CookieJar    *cookiejar.Jar
	LoginPath    string
	GetMACPath   string
	QVSCSRFToken string
	QVSSessionID string
}

type QTSLoginResponse struct {
	Need2SV    int    `xml:"need_2sv"`
	PWStatus   int    `xml:"pw_status"`
	AuthPassed int    `xml:"authPassed"`
	AuthSID    string `xml:"authSid"`
	Username   string `xml:"username"`
}

type ListFile struct {
	Filename string `json:"filename"`
	IsFolder int    `json:"isfolder"`
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
	ID         int               `json:"id"`
	UUID       string            `json:"uuid"`
	Name       string            `json:"name"`
	Cores      int               `json:"cores"`
	PowerState string            `json:"power_state"`
	Disks      []VMDisksResponse `json:"disks"`
}

type VMDisksResponse struct {
	ID         int    `json:"id"`
	VMID       int    `json:"vm_id"`
	Path       string `json:"path"`
	RootPath   string `json:"root_path"`
	PathExist  bool   `json:"path_exist"`
	Size       int    `json:"size"`
	ActualSize int    `json:"actual_size"`
	Format     string `json:"format"`
	Bus        string `json:"bus"`
	Cache      string `json:"cache"`
	Dev        string `json:"dev"`
	BootOrder  int    `json:"boot_order"`
	Index      int    `json:"index"`
	IsDOM      bool   `json:"is_dom"`
	VolumeName string `json:"volume_name"`
}

type NetMgrNet struct {
	DisplayName string `json:"display_name"`
	PhysicalNIC string `json:"physical_nic"`
	Type        string `json:"type"`
	VSwitchName string `json:"vswitch_name"`
	VSwitchIP   string `json:"vswitch_ip"`
}

type QVSNet struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	IP          string   `json:"ip"`
	NICs        []string `json:"nics"`
}

type QVSCreateRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	OSType         string              `json:"os_type"`
	Cores          int                 `json:"cores"`
	Memory         int                 `json:"memory"`
	Adapters       []map[string]string `json:"adapters"`
	QVM            bool                `json:"qvm"`
	IsAgentEnabled bool                `json:"is_agent_enabled"`
	CDROMs         []map[string]string `json:"cdroms"`
	Disks          []map[string]string `json:"disks"`
}

const DefaultMetaData = `instance-id: qvs-%s-%d
local-hostname: %s`

const DefaultUserData = `#cloud-config
hostname: %s
ssh_authorized_keys:
  - %s
power_state:
  mode: reboot
`
