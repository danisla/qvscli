# qvscli

A command line tool for interacting with the QNAP Virtualization Station REST API.

## External Dependencies

To build the cloud-init metadata ISO, install `genisoimage`:

```
sudo apt-get install genisoimage
```

## Usage

```
qvscli login

qvscli vm list

qvs cli vm desc 1
```

## Building

```
go install
```