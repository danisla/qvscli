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
	var qtsDisksDir string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var loginFile string
	var metaDataFile string
	var userDataFile string

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
			Destination: &qtsDisksDir,
			EnvVar:      "QVSCLI_QTS_DISKS_DIR",
		},
		cli.StringFlag{
			Name:        "loginfile",
			Value:       defaultLoginFile,
			Usage:       "Override default login file.",
			Destination: &loginFile,
			EnvVar:      "QVSCLI_LOGIN_FILE",
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
					Name:    "create",
					Aliases: []string{"c"},
					Usage:   "create a VM with provided meta-data and user-data",
					Action: func(c *cli.Context) error {
						client := getClient()

						name := c.Args().Get(0)

						// Verify name is valid
						nameRegex := regexp.MustCompile(`^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`)
						if !nameRegex.MatchString(name) {
							return fmt.Errorf("invalid instance name: %s", name)
						}
						ts := time.Now().UTC().Unix()
						isoDestFile := fmt.Sprintf("metadata_%d.iso", ts)
						if err := makeConfigISO(isoDestFile, metaDataFile, userDataFile); err != nil {
							return err
						}

						qtsDest := path.Join(qtsDisksDir, name, path.Base(isoDestFile))

						// Check for existing folder
						files, err := client.ListDir(qtsDisksDir)
						if err != nil {
							return err
						}
						found := false
						for _, qf := range files {
							if qf.Filename == name {
								found = true
								break
							}
						}

						// Create directory for VM disk
						if found == false {
							log.Printf("INFO: Creating directory on NAS for VM: %s", path.Dir(qtsDest))
							if err := client.CreateDir(path.Dir(qtsDest)); err != nil {
								return err
							}
						}

						f, err := os.Open(isoDestFile)
						if err != nil {
							return err
						}

						if err := client.UploadFile(f, qtsDest); err != nil {
							return err
						}

						log.Printf("Uploaded metadata ISO image to NAS: %s\n", qtsDest)

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

func makeConfigISO(isoDestFile, metaDataFile, userDataFile string) error {
	cmd := "genisoimage"
	args := []string{"-output", isoDestFile, "-volid", "cidata", "-joliet", "-rock", userDataFile, metaDataFile}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		return fmt.Errorf("%v, %s", os.Stderr, err)
	}
	return nil
}
