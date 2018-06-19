package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/urfave/cli"
)

func main() {
	var qvsURL string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var loginFile string

	app := cli.NewApp()
	app.Name = "qvscli"
	app.Usage = "Interact with QNAP Virtualization Station"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "url, u",
			Value:       "http://localhost:8088",
			Usage:       "URL of QVS, typically the IP of your QNAP NAS on port 8088",
			Destination: &qvsURL,
			FilePath:    fmt.Sprintf("%s/.qvscli_url", os.Getenv("HOME")),
			EnvVar:      "QVSCLI_URL",
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
				client := NewQVSClient(qvsURL, loginFile)
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
						client := NewQVSClient(qvsURL, loginFile)
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
						client := NewQVSClient(qvsURL, loginFile)
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
						client := NewQVSClient(qvsURL, loginFile)
						id := c.Args().First()
						vms, err := client.VMDescribe(id)
						if err != nil {
							return err
						}
						fmt.Println(vms)
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
