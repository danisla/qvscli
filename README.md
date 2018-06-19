# qvscli

A command line tool for interacting with the QNAP Virtualization Station REST API.

## Usage

```
$ qvscli help
NAME:
   qvscli - Interact with QNAP Virtualization Station

USAGE:
   qvscli [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
     login    login to QVS and obtain session cookie stored in ${HOME}/.qvs_login
     mac      options for mac addresses
     vm       options for virtual machines
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loginfile value      Override default login file. (default: "/Users/disla/.qvs_login") [$QVSCLI_LOGIN_FILE]
   --url value, -u value  URL of QVS, typically the IP of your QNAP NAS on port 8088 (default: "http://localhost:8088") [$QVSCLI_URL] [/Users/disla/.qvscli_url]
   --help, -h             show help
   --version, -v          print the version
```