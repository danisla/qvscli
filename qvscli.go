package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	var qtsURL string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var loginFile string
	var metaDataFile string
	var userDataFile string

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

	getClient := func() *QVSClient {
		return NewQVSClient(qtsURL, loginFile)
	}

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "login to QVS and obtain session cookie stored in ${HOME}/.qvs_login",
			Action: func(c *cli.Context) error {
				client := getClient()
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
						// client := NewQVSClient(qtsURL, loginFile)

						name := c.Args().Get(0)

						// Verify name is valid
						nameRegex := regexp.MustCompile(`^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`)
						if !nameRegex.MatchString(name) {
							return fmt.Errorf("invalid instance name: %s", name)
						}
						isoDestFile := fmt.Sprintf("%s-config.iso", strings.Replace(strings.Replace(name, " ", "-", -1), "_", "-", -1))
						if err := makeConfigISO(isoDestFile, metaDataFile, userDataFile); err != nil {
							return err
						}

						log.Printf("Created config ISO image: %s\n", isoDestFile)

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
