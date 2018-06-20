package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"time"

	"github.com/urfave/cli"
)

func main() {
	var outputFormat string
	var qtsURL string
	var qvsDisksDir string
	var qvsImagesDir string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var loginFile string
	var metaDataFile string
	var userDataFile string
	var noCloudInit bool
	var vmImage string
	var vmMACAddress string
	var vmNetName string
	var vmDescription string
	var vmCores int
	var vmMemoryGB int

	getClient := func() *QVSClient {
		client, err := NewQVSClient(qtsURL, loginFile, false)
		if err != nil {
			log.Fatal(err)
		}
		return client
	}

	app := cli.NewApp()
	app.Name = "qvscli"
	app.Usage = "Interact with QNAP Virtualization Station"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "qts-url",
			Value:       "https://qnap.homelab.cloud",
			Usage:       "URL of QTS, typically the https DNS name of your QNAP NAS",
			Destination: &qtsURL,
			EnvVar:      "QVSCLI_QTS_URL",
		},
		cli.StringFlag{
			Name:        "qvs-disks-dir",
			Value:       "/VirtualMachines/disks",
			Usage:       "NAS path to folder where disk images are stored",
			Destination: &qvsDisksDir,
			EnvVar:      "QVSCLI_QVS_DISKS_DIR",
		},
		cli.StringFlag{
			Name:        "qvs-images-dir",
			Value:       "/VirtualMachines/images",
			Usage:       "NAS path to base image directory containing folders or .img files",
			Destination: &qvsImagesDir,
			EnvVar:      "QVSCLI_QVS_IMAGES_DIR",
		},
		cli.StringFlag{
			Name:        "loginfile",
			Value:       defaultLoginFile,
			Usage:       "Override default login file.",
			Destination: &loginFile,
			EnvVar:      "QVSCLI_LOGIN_FILE",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "login to QVS and obtain session cookie stored in ${HOME}/.qvs_login",
			Action: func(c *cli.Context) error {
				client, err := NewQVSClient(qtsURL, loginFile, true)
				if err != nil {
					return err
				}
				return client.Login()
			},
		},
		{
			Name:  "mac",
			Usage: "options for mac addresses",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "generate a new mac address",
					Action: func(c *cli.Context) error {
						client := getClient()
						mac, err := client.MACCreate()
						if err != nil {
							return err
						}
						fmt.Printf("%s\n", mac)
						return nil
					},
				},
			},
		},
		{
			Name:    "networks",
			Aliases: []string{"net"},
			Usage:   "options for virtual networks",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list virtual networks",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "output, o",
							Usage:       "Output format, text or json",
							Value:       "text",
							Destination: &outputFormat,
						},
					},
					Action: func(c *cli.Context) error {
						client := getClient()

						networks, err := client.QVSListNet()
						if err != nil {
							return err
						}

						if outputFormat == "json" {
							pretty, _ := json.MarshalIndent(networks, "", "  ")
							fmt.Println(string(pretty))
						} else {
							for _, network := range networks {
								fmt.Printf("%s\t%s\t%s\t%v\n", network.Name, network.DisplayName, network.IP, network.NICs)
							}
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "vm",
			Usage: "options for virtual machines",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list virtual machines",
					Action: func(c *cli.Context) error {
						client := getClient()
						vms, err := client.VMList()
						if err != nil {
							return err
						}
						for _, v := range vms {
							fmt.Printf("id=%d state=%s name=\"%s\"\n", v.ID, v.PowerState, v.Name)
						}
						return nil
					},
				},
				{
					Name:    "describe",
					Aliases: []string{"desc"},
					Usage:   "describe VM by ID",
					Action: func(c *cli.Context) error {
						client := getClient()
						id := c.Args().First()
						vms, err := client.VMDescribe(id)
						if err != nil {
							return err
						}
						fmt.Println(vms)
						return nil
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"del"},
					Usage:   "delete a VM by ID",
					Action: func(c *cli.Context) error {
						// Fetch VM meta
						// Make sure VM is stopped
						// Delete VM
						// Delete disk dir.
						return fmt.Errorf("NYI")
					},
				},
				{
					Name:    "create",
					Aliases: []string{"c"},
					Usage:   "create a VM with provided meta-data and user-data",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "meta-data",
							Value:       "meta-data.yaml",
							Usage:       "Path to meta-data file for cloud-init",
							Destination: &metaDataFile,
							EnvVar:      "QVSCLI_META_DATA_FILE",
						},
						cli.StringFlag{
							Name:        "user-data",
							Value:       "user-data.yaml",
							Usage:       "Path to user-data file for cloud-init",
							Destination: &userDataFile,
							EnvVar:      "QVSCLI_USER_DATA_FILE",
						},
						cli.BoolFlag{
							Name:        "no-cloud-init",
							Usage:       "Disable cloud-init metadata ISO creation",
							Destination: &noCloudInit,
							EnvVar:      "QVSCLI_NO_CLOUD_INIT",
						},
						cli.StringFlag{
							Name:        "image",
							Value:       "debian-cloud/debian-9.img",
							Usage:       "Path to VM base image relative to qvs-images-dir",
							Destination: &vmImage,
							EnvVar:      "QVSCLI_VM_IMAGE",
						},
						cli.StringFlag{
							Name:        "mac",
							Value:       "",
							Usage:       "Set mac address of network interface, if not set, one will be created",
							Destination: &vmMACAddress,
							EnvVar:      "QVSCLI_VM_MAC",
						},
						cli.StringFlag{
							Name:        "network, net",
							Value:       "br0",
							Usage:       "Network interface to attach, get names from 'qvscli net list'",
							Destination: &vmNetName,
							EnvVar:      "QVSCLI_VM_NET",
						},
						cli.StringFlag{
							Name:        "description, desc",
							Value:       "",
							Usage:       "VM description. Default is auto-generated based on the creation time",
							Destination: &vmDescription,
							EnvVar:      "QVS_CLI_VM_DESCRIPTION",
						},
						cli.IntFlag{
							Name:        "cores",
							Value:       1,
							Usage:       "Number of cores for VM",
							Destination: &vmCores,
							EnvVar:      "QVS_CLI_VM_CORES",
						},
						cli.IntFlag{
							Name:        "memory, mem",
							Value:       2,
							Usage:       "Memory for VM in integer Gigabytes",
							Destination: &vmMemoryGB,
							EnvVar:      "QVS_CLI_VM_MEM_GB",
						},
					},
					Action: func(c *cli.Context) error {
						client := getClient()

						name := c.Args().Get(0)

						// Verify name is valid
						nameRegex := regexp.MustCompile(`^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`)
						if !nameRegex.MatchString(name) {
							return fmt.Errorf("invalid instance name: %s", name)
						}

						// Generate MAC address
						if vmMACAddress == "" {
							vmMACAddress, err := client.MACCreate()
							if err != nil {
								return err
							}
							log.Printf("INFO: Generated new MAC address for instance: %s", vmMACAddress)
						}

						// Verify image exists
						vmImageSrc := path.Join(qvsImagesDir, vmImage)
						imageFiles, err := client.ListDir(path.Dir(vmImageSrc))
						if err != nil {
							return err
						}
						found := false
						for _, imageFile := range imageFiles {
							if imageFile.Filename == path.Base(vmImage) {
								found = true
								break
							}
						}
						if found == false {
							return fmt.Errorf("VM image file not found: %s", vmImage)
						}

						// Timestamp for generated artifacts
						ts := time.Now().UTC().Unix()

						// Userdata and metadata handling
						metadataISOFile := ""
						metadataISODest := ""
						if noCloudInit {
							log.Printf("WARN: cloud-init disabled, skipping metadata ISO creation. You may not be able log into the VM after booting.")
						} else {
							metadataISOFile = fmt.Sprintf("metadata_%d.iso", ts)
							if _, err := os.Stat(userDataFile); os.IsNotExist(err) {
								return fmt.Errorf("user-data file does not exist: %s", userDataFile)
							}
							if _, err := os.Stat(metaDataFile); os.IsNotExist(err) {
								return fmt.Errorf("meta-data file does not exist: %s", metaDataFile)
							}
							if err := makeConfigISO(metadataISOFile, metaDataFile, userDataFile); err != nil {
								return err
							}
							metadataISODest = path.Join(qvsDisksDir, name, path.Base(metadataISOFile))
						}

						// Check for existing folder
						files, err := client.ListDir(qvsDisksDir)
						if err != nil {
							return err
						}
						found = false
						for _, qf := range files {
							if qf.Filename == name {
								found = true
								break
							}
						}

						// Create directory for VM disk
						if found == false {
							log.Printf("INFO: Creating directory on NAS for VM: %s", path.Dir(metadataISODest))
							if err := client.CreateDir(path.Dir(metadataISODest)); err != nil {
								return err
							}
						}

						if metadataISODest != "" {
							f, err := os.Open(metadataISOFile)
							if err != nil {
								return err
							}

							log.Printf("INFO: Uploading metadata ISO image to NAS: %s\n", metadataISODest)
							if err := client.UploadFile(f, metadataISODest); err != nil {
								return err
							}
						}

						// Remote copy image to VM disk directory
						vmImageDest := path.Join(qvsDisksDir, name, path.Base(vmImage))
						vmBootDiskFile := fmt.Sprintf("boot_disk_%d.img", ts)

						log.Printf("INFO: Remote copy VM image %s -> %s", vmImageSrc, path.Join(path.Dir(vmImageDest), vmBootDiskFile))
						if err := client.CopyFile(vmImageSrc, vmImageDest); err != nil {
							return err
						}
						if err := client.RenameFile(path.Dir(vmImageDest), path.Base(vmImageDest), vmBootDiskFile); err != nil {
							return err
						}

						return nil
					},
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
