package userdata

import (
	"bytes"
	"fmt"
	"text/template"
)

// StorageConfig contains configuration for storage mounting
type StorageConfig struct {
	FSxLustreEnabled bool
	FSxFilesystemDNS string
	FSxMountName     string
	FSxMountPoint    string

	EFSEnabled       bool
	EFSFilesystemDNS string
	EFSMountPoint    string
}

// GenerateStorageUserData generates storage mounting script
func GenerateStorageUserData(config StorageConfig) (string, error) {
	tmpl, err := template.New("storage").Parse(storageUserDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse storage template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute storage template: %w", err)
	}

	return buf.String(), nil
}

const storageUserDataTemplate = `
{{if .FSxLustreEnabled}}
# FSx Lustre mounting
amazon-linux-extras install -y lustre2.12
mkdir -p {{.FSxMountPoint}}
mount -t lustre {{.FSxFilesystemDNS}}@tcp:/{{.FSxMountName}} {{.FSxMountPoint}}
echo "{{.FSxFilesystemDNS}}@tcp:/{{.FSxMountName}} {{.FSxMountPoint}} lustre defaults,noatime,flock,_netdev 0 0" >> /etc/fstab
echo "export FSX_MOUNT={{.FSxMountPoint}}" >> /etc/profile.d/fsx.sh
{{end}}

{{if .EFSEnabled}}
# EFS mounting
yum install -y nfs-utils
mkdir -p {{.EFSMountPoint}}
mount -t nfs4 -o nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2 {{.EFSFilesystemDNS}}:/ {{.EFSMountPoint}}
echo "{{.EFSFilesystemDNS}}:/ {{.EFSMountPoint}} nfs4 nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2,_netdev 0 0" >> /etc/fstab
echo "export EFS_MOUNT={{.EFSMountPoint}}" >> /etc/profile.d/efs.sh
{{end}}
`
