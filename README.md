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

## Using Ubuntu Cloud images

If image hangs on first boot, to the fowlloing to remove the `console=ttyS0` line from the `grub.cfg` in the disk image before creating the VM:

```
apt-get update && apt-get install -y curl qemu-utils nbd-client
modprobe nbd
curl -LO http://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-disk1.img
qemu-nbd --connect=/dev/nbd0 xenial-server-cloudimg-amd64-disk1.img
mkdir -p /mnt/ubuntu
mount /dev/nbd0p1 /mnt/ubuntu
sed -i 's/console=ttyS0//g' /mnt/ubuntu/boot/grub.cfg
umount /mnt/ubuntu

# SFTP image to NAS
```