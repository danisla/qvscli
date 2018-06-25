package main

const DefaultUserDataTemplate = `#cloud-config
hostname: {{.Hostname}}

password: "{{.LoginPassword}}"
ssh_pwauth: True
chpasswd: { expire: False }

write_files:
- path: /etc/network/if-up.d/show-ip-address
  permissions: '0755'
  content: |
    #!/bin/sh
    egrep -q -e "eth0=[0-9].*" /etc/issue && exit 0
    sed -i'' 's/\\n \\l.*$/\\n \\l '"eth0=$(hostname -I)"'/g' /etc/issue

{{- if .StartupScript}}
- path: /var/lib/cloud/scripts/per-instance/startup-script.sh
  permissions: '0755'
  content: |
{{.StartupScript | indent 4}}
{{- end}}

ssh_authorized_keys:
- {{.AuthorizedKey}}
power_state:
  mode: reboot
`
