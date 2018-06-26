package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"

	"github.com/sethvargo/go-password/password"
	"github.com/urfave/cli"
)

func main() {
	var httpDebug bool
	var outputFormat string
	var qtsURL string
	var qvsDisksDir string
	var qvsImagesDir string
	var defaultLoginFile = fmt.Sprintf("%s/.qvs_login", os.Getenv("HOME"))
	var defaultPubKeyFile = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa.pub")
	var loginFile string
	var metaDataFile string
	var userDataFile string
	var vmStartupScript string
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
	var vmNoLocalLogin bool
	var vmAuthorizedKey string
	var vmVNCPassword string
	var vmSnapshotIDOrName string

	getClient := func() *QVSClient {
		client, err := NewQVSClient(qtsURL, loginFile, false, httpDebug)
		if err != nil {
			log.Fatal(err)
		}
		return client
	}

	app := cli.NewApp()
	app.Name = "qvscli"
	app.Usage = "Interact with QNAP Virtualization Station"
	app.Version = "0.0.3-dev"

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
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Enable HTTP response debugging",
			Destination: &httpDebug,
			EnvVar:      "QVSCLI_HTTP_DEBUG",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "login to QVS and obtain session cookie stored in ${HOME}/.qvs_login",
			Action: func(c *cli.Context) error {
				client, err := NewQVSClient(qtsURL, loginFile, true, httpDebug)
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
			Name:    "images",
			Aliases: []string{"image"},
			Usage:   "options for vm disk images",
			Subcommands: []cli.Command{
				{
					Name:      "list",
					Aliases:   []string{"ls"},
					Usage:     "list images found in the qvs-images-dir",
					ArgsUsage: "[path]",
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

						imageFilesPath := c.Args().First()

						listPath := filepath.Join(qvsImagesDir, imageFilesPath)

						imageFiles, err := client.ListDir(listPath)
						if err != nil {
							return err
						}

						if outputFormat == "json" {
							pretty, _ := json.MarshalIndent(imageFiles, "", "  ")
							fmt.Println(string(pretty))
						} else if outputFormat == "text" {
							// Sort by filename
							sort.Slice(imageFiles, func(i, j int) bool {
								return strings.ToLower(imageFiles[i].Filename) < strings.ToLower(imageFiles[j].Filename)
							})

							w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
							fmt.Fprintln(w, "NAME")
							for _, f := range imageFiles {
								// Filter by *.img, folders
								if (strings.Contains(f.Filename, ".img") || f.IsFolder == 1) && !strings.Contains(f.Filename, "@") {
									displayName := filepath.Join(imageFilesPath, f.Filename)
									if f.IsFolder == 1 {
										displayName += "/"
									}
									fmt.Fprintf(w, strings.Join([]string{
										displayName,
									}, "\t")+"\n")
								}
							}
							w.Flush()
						} else {
							return fmt.Errorf("invalid output format: %s", outputFormat)
						}
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
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "list virtual networks",
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
							// Sort by display name
							sort.Slice(networks, func(i, j int) bool {
								return strings.ToLower(networks[i].DisplayName) < strings.ToLower(networks[j].DisplayName)
							})

							w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
							fmt.Fprintln(w, "NAME\tBRIDGE\tIP\tINTERFACES")
							for _, n := range networks {
								fmt.Fprintf(w, strings.Join([]string{
									n.DisplayName,
									n.Name,
									n.IP,
									strings.Join(n.NICs, ","),
								}, "\t")+"\n")
							}
							w.Flush()
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
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "list virtual machines",
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
							// Sort by name
							sort.Slice(vms, func(i, j int) bool {
								return strings.ToLower(vms[i].Name) < strings.ToLower(vms[j].Name)
							})

							w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
							fmt.Fprintln(w, "NAME\tID\tSTATE\tNETWORK\tMAC ADDRESS\tVNC PORT")
							for _, v := range vms {
								vncPort := ""
								if len(v.Graphics) > 0 && v.Graphics[0].Port > 0 {
									vncPort = fmt.Sprintf("%d", v.Graphics[0].Port)
								}
								fmt.Fprintf(w, strings.Join([]string{
									v.Name,
									fmt.Sprintf("%d", v.ID),
									v.PowerState,
									v.Adapters[0].Bridge,
									v.Adapters[0].MAC,
									vncPort,
								}, "\t")+"\n")
							}
							w.Flush()
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
					Name:  "reset",
					Usage: "reset a VM by ID or name",
					Action: func(c *cli.Context) error {
						client := getClient()
						idOrName := c.Args().First()
						id, err := client.VMGetID(idOrName)
						if err != nil {
							return err
						}
						if err := client.VMReset(id); err != nil {
							return err
						} else {
							log.Printf("INFO: reset VM: %s", idOrName)
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
							if vmForceShutdown {
								log.Printf("INFO: VM stopped: %s.", idOrName)
							} else {
								log.Printf("INFO: Sent ACPI shutdown signal to VM: %s.", idOrName)
							}
						}
						return nil
					},
				},
				{
					Name:    "delete",
					Aliases: []string{"del", "rm"},
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
							Name:        "startup-script",
							Value:       "",
							Usage:       "Path to startup script to run as runcmd action in cloud-init",
							Destination: &vmStartupScript,
							EnvVar:      "QVSCLI_STARTUP_SCRIPT",
						},
						cli.StringFlag{
							Name:        "meta-data",
							Value:       "",
							Usage:       "Path to meta-data file override for cloud-init, default is generated automatically",
							Destination: &metaDataFile,
							EnvVar:      "QVSCLI_META_DATA_FILE",
						},
						cli.StringFlag{
							Name:        "user-data",
							Value:       "",
							Usage:       "Path to user-data file for cloud-init",
							Destination: &userDataFile,
							EnvVar:      "QVSCLI_USER_DATA_FILE",
						},
						cli.StringFlag{
							Name:        "authorized-key",
							Value:       defaultPubKeyFile,
							Usage:       "Path to public ssh key file when --user-data is not provided.",
							Destination: &vmAuthorizedKey,
							EnvVar:      "QVSCLI_META_DATA_FILE",
						},
						cli.BoolFlag{
							Name:        "no-local-login",
							Usage:       "Disable local login, a password will not be generated and only SSH can be used to access the VM.",
							Destination: &vmNoLocalLogin,
							EnvVar:      "QVSCLI_NO_LOCAL_LOGIN",
						},
						cli.BoolFlag{
							Name:        "no-cloud-init",
							Usage:       "Disable cloud-init metadata ISO creation",
							Destination: &noCloudInit,
							EnvVar:      "QVSCLI_NO_CLOUD_INIT",
						},
						cli.StringFlag{
							Name:        "image",
							Value:       "ubuntu-cloud/xenial.img",
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
							EnvVar:      "QVSCLI_VM_DESCRIPTION",
						},
						cli.IntFlag{
							Name:        "cores",
							Value:       1,
							Usage:       "Number of cores for VM",
							Destination: &vmCores,
							EnvVar:      "QVSCLI_VM_CORES",
						},
						cli.IntFlag{
							Name:        "memory, mem",
							Value:       2,
							Usage:       "Memory for VM in integer Gigabytes",
							Destination: &vmMemoryGB,
							EnvVar:      "QVSCLI_VM_MEM_GB",
						},
						cli.StringFlag{
							Name:        "vnc-password",
							Value:       "",
							Usage:       "VNC password up to 8 characters long. If not set, one will be automatically generated.",
							Destination: &vmVNCPassword,
							EnvVar:      "QVSCLI_VM_VNC_PASSWORD",
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
							dir, err := ioutil.TempDir("", "ci-metadata-iso")
							defer os.RemoveAll(dir)
							if err != nil {
								return err
							}
							metadataISOFile = filepath.Join(dir, fmt.Sprintf("metadata_%d.iso", ts))

							if metaDataFile == "" {
								metaDataFile = filepath.Join(dir, fmt.Sprintf("meta-data"))
								mf, err := os.OpenFile(metaDataFile, os.O_RDWR|os.O_CREATE, 0644)
								if err != nil {
									return err
								}
								_, err = mf.WriteString(fmt.Sprintf(DefaultMetaData, name, ts, name))
								if err != nil {
									return err
								}
								mf.Close()
							}

							if userDataFile == "" {
								authKeyData, err := ioutil.ReadFile(vmAuthorizedKey)
								if err != nil {
									return fmt.Errorf("could not generate user-data, error reading %s and --authorized-key not provided, %v", vmAuthorizedKey, err)
								}
								userDataFile = filepath.Join(dir, fmt.Sprintf("user-data"))
								uf, err := os.OpenFile(userDataFile, os.O_RDWR|os.O_CREATE, 0644)
								if err != nil {
									return err
								}

								// Generate SSH password
								vmSSHPassword, err := password.Generate(8, 2, 0, false, false)
								log.Printf("Your SSH password is: %s", vmSSHPassword)

								// Generate user-data from template
								t, _ := template.New("user-data").Funcs(sprig.TxtFuncMap()).Parse(DefaultUserDataTemplate)
								type tmplData struct {
									Hostname      string
									LocalLogin    bool
									LoginPassword string
									StartupScript string
									AuthorizedKey string
								}

								// Read startup-script file, if defined.
								var startupScript []byte
								_, err = os.Stat(vmStartupScript)
								if err == nil {
									startupScript, err = ioutil.ReadFile(vmStartupScript)
									if err != nil {
										return err
									}
								}

								data := tmplData{
									Hostname:      name,
									LocalLogin:    !vmNoLocalLogin,
									LoginPassword: vmSSHPassword,
									StartupScript: string(startupScript),
									AuthorizedKey: string(authKeyData),
								}
								if err = t.Execute(uf, data); err != nil {
									return err
								}
								uf.Close()
							}

							if _, err := os.Stat(userDataFile); os.IsNotExist(err) {
								return fmt.Errorf("user-data file does not exist: %s", userDataFile)
							}
							if _, err := os.Stat(metaDataFile); os.IsNotExist(err) {
								return fmt.Errorf("meta-data file does not exist: %s", metaDataFile)
							}
							if err = makeConfigISO(metadataISOFile, metaDataFile, userDataFile); err != nil {
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
							vmDescription = fmt.Sprintf("Created with qvscli at %s", now.Format("20060102150405"))
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

						// Generate VNC Password if not given
						if vmVNCPassword == "" {
							// Generate a password that is 8 characters long with 3 digits, 0 symbols,
							// allowing upper and lower case letters, disallowing repeat characters.
							vmVNCPassword, err = password.Generate(8, 2, 0, false, false)
							log.Printf("Your VNC password is: %s", vmVNCPassword)
						}

						// Create VM
						if err := client.VMCreate(name, vmDescription, "linux", vmCores, vmMemoryGB, vmNetName, vmMACAddress, metadataISODest, vmImagePath, vmVNCPassword); err != nil {
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
								v, err := client.VMGet(name)
								if err != nil {
									return err
								}
								log.Printf("INFO: VM started. VNC port: %d", v.Graphics[0].Port)
							}
						}
						return nil
					},
				},
				{
					Name:    "snapshot",
					Aliases: []string{"snap"},
					Usage:   "options for VM disk snapshots",
					Subcommands: []cli.Command{
						{
							Name:    "list",
							Aliases: []string{"ls"},
							Usage:   "list all VM snapshots",
							Action: func(c *cli.Context) error {
								client := getClient()
								snapDir := filepath.Join(qvsImagesDir, "snapshots")
								snapFiles, err := client.ListDir(snapDir)
								if err != nil {
									return err
								}
								w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
								fmt.Fprintln(w, "NAME\tTIMESTAMP")
								for _, f := range snapFiles {
									// ts := time.Unix(f.EpochMT, 0)
									if f.IsFolder == 0 {
										fmt.Fprintf(w, strings.Join([]string{
											f.Filename,
											f.MT,
										}, "\t")+"\n")
									}
								}
								w.Flush()

								return nil
							},
						},
						{
							Name:  "create",
							Usage: "create a snapshot by VM ID or name",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:        "vm",
									Usage:       "The ID or name of the VM to snapshot.",
									Destination: &vmSnapshotIDOrName,
								},
							},
							Action: func(c *cli.Context) error {
								client := getClient()
								id, err := client.VMGetID(vmSnapshotIDOrName)
								if err != nil {
									return err
								}

								name := c.Args().First()
								snapDir := filepath.Join(qvsImagesDir, "snapshots")
								snap, err := client.VMDiskSnapshotCreate(id, name, snapDir)
								if err != nil {
									return err
								}

								log.Printf("Created disk snapshot %s", snap)

								return nil
							},
						},
						{
							Name:      "delete",
							Aliases:   []string{"del", "rm"},
							Usage:     "delete a snapshot",
							ArgsUsage: "[snapshot file name]",
							Action: func(c *cli.Context) error {
								client := getClient()

								snapFile := c.Args().First()
								snapDir := filepath.Join(qvsImagesDir, "snapshots")
								snapFiles, err := client.ListDir(snapDir)
								if err != nil {
									return nil
								}

								for _, f := range snapFiles {
									if filepath.Base(f.Filename) == snapFile {
										log.Printf("Deleting snapshot file: %s", f.Filename)
										return client.DeleteFile(filepath.Join(snapDir, f.Filename))
									}
								}

								return fmt.Errorf("failed to find snapshot file '%s' in snapshot directory", snapFile)
							},
						},
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
