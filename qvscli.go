package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"time"

	"github.com/urfave/cli"
)

func main() {
	var qtsURL string
	var qvsDisksDir string
	var qvsImagesDir string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var loginFile string
	var metaDataFile string
	var userDataFile string
	var vmImage string
	var macAddress string

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
							Destination: &macAddress,
							EnvVar:      "QVSCLI_VM_MAC",
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
						if macAddress == "" {
							macAddress, err := client.MACCreate()
							if err != nil {
								return err
							}
							log.Printf("INFO: Generated new MAC address for instance: %s", macAddress)
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

						metadataISOFile := fmt.Sprintf("metadata_%d.iso", ts)
						if err := makeConfigISO(metadataISOFile, metaDataFile, userDataFile); err != nil {
							return err
						}

						metadataISODest := path.Join(qvsDisksDir, name, path.Base(metadataISOFile))

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

						f, err := os.Open(metadataISOFile)
						if err != nil {
							return err
						}

						log.Printf("INFO: Uploading metadata ISO image to NAS: %s\n", metadataISODest)
						if err := client.UploadFile(f, metadataISODest); err != nil {
							return err
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

func makeConfigISO(metadataISOFile, metaDataFile, userDataFile string) error {
	cmd := "genisoimage"
	args := []string{"-output", metadataISOFile, "-volid", "cidata", "-joliet", "-rock", userDataFile, metaDataFile}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		return fmt.Errorf("%v, %s", os.Stderr, err)
	}
	return nil
}
