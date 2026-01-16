package userdata

import (
	"bytes"
	"fmt"
	"text/template"
)

// MPIConfig contains configuration for MPI user-data generation
type MPIConfig struct {
	Region              string
	JobArrayID          string
	JobArrayIndex       int
	JobArraySize        int
	MPIProcessesPerNode int
	MPICommand          string
}

// GenerateMPIUserData generates the MPI setup script for inclusion in user-data
func GenerateMPIUserData(config MPIConfig) (string, error) {
	tmpl, err := template.New("mpi").Parse(mpiUserDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse MPI template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute MPI template: %w", err)
	}

	return buf.String(), nil
}

const mpiUserDataTemplate = `
# ============================================================================
# MPI (Message Passing Interface) Setup
# ============================================================================

echo "🚀 MPI Setup - Node {{.JobArrayIndex}}/{{.JobArraySize}}"

# 1. Install OpenMPI
echo "📦 Installing OpenMPI..."
yum install -y openmpi openmpi-devel
export PATH=/usr/lib64/openmpi/bin:$PATH
export LD_LIBRARY_PATH=/usr/lib64/openmpi/lib:$LD_LIBRARY_PATH

# 2. Configure MPI environment for all users
cat >> /etc/profile.d/mpi.sh <<'EOFMPI'
export PATH=/usr/lib64/openmpi/bin:$PATH
export LD_LIBRARY_PATH=/usr/lib64/openmpi/lib:$LD_LIBRARY_PATH
export OMPI_MCA_plm_rsh_agent=ssh
export OMPI_MCA_btl_tcp_if_include=eth0
EOFMPI

chmod 644 /etc/profile.d/mpi.sh
source /etc/profile.d/mpi.sh

# 3. SSH Key Distribution for Passwordless SSH
if [ "{{.JobArrayIndex}}" -eq 0 ]; then
    # LEADER: Generate SSH key and upload to S3
    echo "🔑 Leader: Generating SSH key..."
    mkdir -p /root/.ssh
    ssh-keygen -t rsa -N "" -f /root/.ssh/id_rsa -q

    # Upload public key to S3
    aws s3 cp /root/.ssh/id_rsa.pub s3://spawn-binaries-{{.Region}}/mpi-keys/{{.JobArrayID}}/id_rsa.pub

    # Add own key to authorized_keys
    cat /root/.ssh/id_rsa.pub >> /root/.ssh/authorized_keys

    echo "✓ Leader: SSH key uploaded to S3"
else
    # WORKER: Wait for leader's public key, then download
    echo "🔑 Worker: Waiting for leader's SSH key..."
    for i in {1..60}; do
        if aws s3 cp s3://spawn-binaries-{{.Region}}/mpi-keys/{{.JobArrayID}}/id_rsa.pub /tmp/leader_key.pub 2>/dev/null; then
            echo "✓ Leader's key downloaded"
            break
        fi
        sleep 2
    done

    # Add leader's key to authorized_keys
    mkdir -p /root/.ssh
    cat /tmp/leader_key.pub >> /root/.ssh/authorized_keys
    echo "✓ Worker: Leader's SSH key added"
fi

# Set SSH permissions
chmod 700 /root/.ssh
chmod 600 /root/.ssh/authorized_keys
chmod 600 /root/.ssh/id_rsa 2>/dev/null || true

# 4. Configure SSH client (disable host key checking for MPI)
cat >> /root/.ssh/config <<'EOFSSH'
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile=/dev/null
EOFSSH
chmod 600 /root/.ssh/config

# 5. Wait for peer discovery file (barrier)
echo "⏳ Waiting for peer discovery..."
while [ ! -f /etc/spawn/job-array-peers.json ]; do
    sleep 2
done
echo "✓ Peer discovery complete"

# 6. Generate MPI hostfile
echo "📝 Generating MPI hostfile..."
{{if .MPIProcessesPerNode}}
SLOTS={{.MPIProcessesPerNode}}
{{else}}
SLOTS=$(nproc)
{{end}}
jq -r ".[] | \"\(.dns) slots=$SLOTS\"" /etc/spawn/job-array-peers.json > /tmp/mpi-hostfile

echo "Hostfile contents:"
cat /tmp/mpi-hostfile

# 7. Leader runs mpirun
if [ "{{.JobArrayIndex}}" -eq 0 ]; then
    echo "🎯 Leader: Running MPI command..."

    # Calculate total processes
    TOTAL_PROCESSES=$(({{.JobArraySize}} * SLOTS))

    # Wait a bit for all workers to be ready
    sleep 10

    {{if .MPICommand}}
    # Run user-specified command via mpirun
    echo "Command: mpirun -np $TOTAL_PROCESSES -hostfile /tmp/mpi-hostfile {{.MPICommand}}"
    mpirun -np $TOTAL_PROCESSES -hostfile /tmp/mpi-hostfile {{.MPICommand}}
    {{else}}
    # No command specified - cluster ready for interactive use
    echo "✓ MPI cluster ready! No command specified."
    echo "Connect via: spawn connect \${JOB_ARRAY_NAME}-0"
    echo "Then run: mpirun -np $TOTAL_PROCESSES -hostfile /tmp/mpi-hostfile <your-program>"
    {{end}}
else
    echo "✓ Worker {{.JobArrayIndex}} ready, waiting for MPI commands from leader"
fi

echo "✅ MPI setup complete"
`
