# qvscli

A command line tool for interacting with the QNAP Virtualization Station REST API.

## Usage

```
qvscli login

qvscli vm list

qvs cli vm desc 1
```

## Building

```
go get -u github.com/howeyc/gopass
go get -u github.com/urfave/cli
go install
```