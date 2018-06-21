package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	var vmForceShutdown bool
	var vmNoStart bool
	var vmNoDiskDel bool
	var vmNoDelInput bool

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
						} else if outputFormat == "text" {
							for _, network := range networks {
								fmt.Printf("%s\t%s\t%s\t%v\n", network.Name, network.DisplayName, network.IP, network.NICs)
							}
						} else {
							return fmt.Errorf("invalid output format: %s", outputFormat)
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
						vms, err := client.VMList()
						if err != nil {
							return err
						}
						if outputFormat == "json" {
							pretty, _ := json.MarshalIndent(vms, "", "  ")
							fmt.Println(string(pretty))
						} else if outputFormat == "text" {
							for _, v := range vms {
								fmt.Printf("id=%d state=%s name=\"%s\"\n", v.ID, v.PowerState, v.Name)
							}
						} else {
							return fmt.Errorf("invalid output format %s", outputFormat)
						}
						return nil
					},
				},
				{
					Name:    "describe",
					Aliases: []string{"desc"},
					Usage:   "describe VM by ID or name",
					Action: func(c *cli.Context) error {
						client := getClient()
						idOrName := c.Args().First()
						id, err := client.VMGetID(idOrName)
						if err != nil {
							return err
						}
						fmt.Printf("id: %s", id)
						vms, err := client.VMDescribe(id)
						if err != nil {
							return err
						}
						pretty, _ := json.MarshalIndent(vms, "", "  ")
						fmt.Println(string(pretty))
						return nil
					},
				},
				{
					Name:  "start",
					Usage: "start a stopped VM by ID or name",
					Action: func(c *cli.Context) error {
						client := getClient()
						idOrName := c.Args().First()
						id, err := client.VMGetID(idOrName)
						if err != nil {
							return err
						}
						if err := client.VMStart(id); err != nil {
							return err
						} else {
							log.Printf("INFO: started VM: %s", idOrName)
						}
						return nil
					},
				},
				{
					Name:    "stop",
					Aliases: []string{"shutdown"},
					Usage:   "stop a VM by ID or name",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:        "force",
							Usage:       "force shutdown the VM",
							Destination: &vmForceShutdown,
						},
					},
					Action: func(c *cli.Context) error {
						client := getClient()
						idOrName := c.Args().First()
						id, err := client.VMGetID(idOrName)
						if err != nil {
							return err
						}
						if err := client.VMShutdown(id, vmForceShutdown); err != nil {
							return err
						} else {
							log.Printf("INFO: VM stopped: %s.", idOrName)
						}
						return nil
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"del"},
					Usage:   "delete a VM by ID or name",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:        "no-input",
							Usage:       "Do not prompt to delete, dangerous!",
							Destination: &vmNoDelInput,
							EnvVar:      "QVSCLI_VM_NO_DEL_INPUT",
						},
						cli.BoolFlag{
							Name:        "no-disk-del",
							Usage:       "Do not delete disks after deleting VM",
							Destination: &vmNoDiskDel,
							EnvVar:      "QVSCLI_VM_NO_DISK_DEL",
						},
					},
					Action: func(c *cli.Context) error {
						client := getClient()
						idOrName := c.Args().First()
						vm, err := client.VMGet(idOrName)
						if err != nil {
							return err
						}
						id := fmt.Sprintf("%d", vm.ID)

						// Confirm deletion
						if !vmNoDelInput {
							reader := bufio.NewReader(os.Stdin)
							fmt.Printf("Delete VM '%s'? (yes/no): ", vm.Name)
							delConfirm, _ := reader.ReadString('\n')
							if strings.ToLower(strings.TrimSpace(delConfirm)) != "yes" {
								return fmt.Errorf("did not answer 'yes' to deleting '%s', aborting.", vm.Name)
							}
						}

						// Make sure VM is stopped
						if vm.PowerState != "stop" {
							log.Printf("WARN: forcing shutdown of running vm: %s", idOrName)
							if err := client.VMShutdown(id, true); err != nil {
								return err
							}
						}

						// Delete VM
						if err := client.VMDelete(id); err != nil {
							return err
						}
						log.Printf("INFO: Deleted VM: %s", idOrName)

						// Delete disk dir.
						if vmNoDiskDel {
							vmDiskPath := filepath.Join(qvsDisksDir, vm.Name)
							return fmt.Errorf("WARN: skipping disk deletion, disk data remains on NAS: %s", vmDiskPath)
						} else if len(vm.Disks) > 0 {
							vmDiskFolder := filepath.Dir(vm.Disks[0].Path)
							if err := client.DeleteFile(vmDiskFolder); err != nil {
								return err
							}
							log.Printf("INFO: Deleted VM disk folder: %s", vmDiskFolder)
						}
						return nil
					},
				},
				{
					Name:    "create",
					Aliases: []string{"c"},
					Usage:   "create a VM with provided meta-data and user-data",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:        "no-start",
							Usage:       "Do not auto-start VM after creation",
							Destination: &vmNoStart,
							EnvVar:      "QVSCLI_VM_NO_START",
						},
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
						if name == "" {
							return fmt.Errorf("no instance name provided")
						}
						nameRegex := regexp.MustCompile(`^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`)
						if !nameRegex.MatchString(name) {
							return fmt.Errorf("invalid instance name: %s", name)
						}

						// Generate MAC address
						if vmMACAddress == "" {
							var err error
							vmMACAddress, err = client.MACCreate()
							if err != nil {
								return err
							}
							log.Printf("INFO: Generated new MAC address for instance: %s", vmMACAddress)
						}

						// Verify image exists
						vmImageSrc := filepath.Join(qvsImagesDir, vmImage)
						imageFiles, err := client.ListDir(filepath.Dir(vmImageSrc))
						if err != nil {
							return err
						}
						found := false
						for _, imageFile := range imageFiles {
							if imageFile.Filename == filepath.Base(vmImage) {
								found = true
								break
							}
						}
						if found == false {
							return fmt.Errorf("VM image file not found: %s", vmImage)
						}

						// Timestamp for generated artifacts
						now := time.Now().UTC()
						ts := now.Unix()

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
							metadataISODest = filepath.Join(qvsDisksDir, name, filepath.Base(metadataISOFile))
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

						if vmDescription == "" {
							vmDescription = fmt.Sprintf("Created with qvmcli at %s", now.Format("20060102150405"))
						}

						// Create directory for VM disk
						if found == false {
							log.Printf("INFO: Creating directory on NAS for VM: %s", filepath.Dir(metadataISODest))
							if err := client.CreateDir(filepath.Dir(metadataISODest)); err != nil {
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
						vmImageDest := filepath.Join(qvsDisksDir, name, filepath.Base(vmImage))
						vmBootDiskFile := fmt.Sprintf("boot_disk_%d.img", ts)
						vmImagePath := filepath.Join(filepath.Dir(vmImageDest), vmBootDiskFile)

						log.Printf("INFO: Remote copy VM image %s -> %s", vmImageSrc, vmImagePath)
						if err := client.CopyFile(vmImageSrc, vmImageDest); err != nil {
							return err
						}
						if err := client.RenameFile(filepath.Dir(vmImageDest), filepath.Base(vmImageDest), vmBootDiskFile); err != nil {
							return err
						}

						// Create VM
						if err := client.VMCreate(name, vmDescription, "linux", vmCores, vmMemoryGB, vmNetName, vmMACAddress, metadataISODest, vmImagePath); err != nil {
							return err
						}
						log.Printf("INFO: VM Created: %s.", name)

						// Start VM
						if vmNoStart {
							return fmt.Errorf("WARN: not starting newly created vm because --no-start flag was passed. To start VM, run: 'qvscli vm start %s", name)
						} else {
							id, err := client.VMGetID(name)
							if err != nil {
								return err
							}
							if err := client.VMStart(id); err != nil {
								return err
							} else {
								log.Printf("INFO: VM started.")
							}
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
