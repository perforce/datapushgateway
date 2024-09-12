#!/bin/bash
set -e

# Function to prompt for input with a default value
prompt() {
    local prompt_message=$1
    local default_value=$2
    local user_input
    read -p "${prompt_message} [${default_value}]: " user_input
    echo "${user_input:-$default_value}"
}

# Dry run mode check
DRY_RUN=true
if [[ "$1" == "-y" ]]; then
    DRY_RUN=false
fi

# Function to print or execute a command based on dry run mode
run_command() {
    local command=$1
    if $DRY_RUN; then
        echo "[DRY RUN] $command"
    else
        eval $command
    fi
}

# Function to back up existing files
backup_if_exists() {
    local file=$1
    if [ -f "$file" ]; then
        local backup_file="${file}.bak.$(date +%Y%m%d%H%M%S)"
        echo "Backing up $file to $backup_file"
        run_command "cp $file $backup_file"
    fi
}

# Check if script is run with sudo
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root or with sudo."
    exit 1
fi

# Prompt for install directories or use defaults
INSTALL_DIR=$(prompt "Enter installation directory" "/opt/perforce/datapushgateway")
DATA_DIR=$(prompt "Enter data directory" "/opt/perforce/datapushgateway/data")
BIN_DIR=$(prompt "Enter binary directory" "/usr/local/bin")
SERVICE_USER=$(prompt "Service User" "perforce")
SERVICE_FILE=$(prompt "Enter service file location" "/etc/systemd/system/datapushgateway.service")

# Dry run info
if $DRY_RUN; then
    echo "Running in dry run mode. No changes will be made."
fi

# Create a system user and group if not exists (assuming perforce user exists)
if ! id -u ${SERVICE_USER} >/dev/null 2>&1; then
    run_command "useradd -r -s /bin/bash ${SERVICE_USER}"
fi

# Create necessary directories
run_command "mkdir -p ${INSTALL_DIR}"
run_command "mkdir -p ${DATA_DIR}"

# Backup files if they already exist
backup_if_exists "${INSTALL_DIR}/auth.yaml"
backup_if_exists "${INSTALL_DIR}/config.yaml"
backup_if_exists "${INSTALL_DIR}/.p4config"
backup_if_exists "${BIN_DIR}/datapushgateway"

# Copy files to appropriate locations
run_command "install -m 0755 datapushgateway ${BIN_DIR}/datapushgateway"
run_command "install -m 0644 auth.yaml ${INSTALL_DIR}/auth.yaml"
run_command "install -m 0644 config.yaml ${INSTALL_DIR}/config.yaml"
run_command "install -m 0644 .p4config ${INSTALL_DIR}/.p4config"

# Set ownership for the files and directories to the Service User
# Ensure all files in INSTALL_DIR and DATA_DIR are owned by SERVICE_USER
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${INSTALL_DIR}"
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${DATA_DIR}"

if $DRY_RUN; then
    echo "[DRY RUN] Creating service file at ${SERVICE_FILE} with the following content:"
    cat <<EOF
[Unit]
Description=Data Push Gateway Service
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/datapushgateway -c ${INSTALL_DIR}/config.yaml -a ${INSTALL_DIR}/auth.yaml -d ${DATA_DIR}
Restart=on-failure
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
Environment="P4CONFIG=${INSTALL_DIR}/.p4config"

[Install]
WantedBy=multi-user.target
EOF
else
    cat <<EOF > ${SERVICE_FILE}
[Unit]
Description=Data Push Gateway Service
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/datapushgateway -c ${INSTALL_DIR}/config.yaml -a ${INSTALL_DIR}/auth.yaml -d ${DATA_DIR}
Restart=on-failure
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
Environment="P4CONFIG=${INSTALL_DIR}/.p4config"

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd to recognize the new service
run_command "systemctl daemon-reload"

# Enable the service but do not start it immediately
# run_command "systemctl enable datapushgateway"

# Apply SELinux policies (if applicable)
if selinuxenabled; then
    run_command "semanage fcontext -a -t bin_t \"${BIN_DIR}/datapushgateway\""
    run_command "semanage fcontext -a -t etc_t \"${INSTALL_DIR}(/.*)?\""
    run_command "semanage fcontext -a -t var_lib_t \"${DATA_DIR}(/.*)?\""
    run_command "restorecon -Rv ${BIN_DIR}/datapushgateway ${INSTALL_DIR}"
fi

echo "Installation completed."
if $DRY_RUN; then
    echo "Dry run completed. No changes were made."
fi

# If not in dry run, echo the contents of the configuration files
if ! $DRY_RUN; then
    echo "Contents of ${INSTALL_DIR}/auth.yaml:"
    cat "${INSTALL_DIR}/auth.yaml"
    echo "------------PLEASE EDIT--------------"

    echo "Contents of ${INSTALL_DIR}/config.yaml:"
    cat "${INSTALL_DIR}/config.yaml"
    echo "------------PLEASE EDIT--------------"

    echo "Contents of ${INSTALL_DIR}/.p4config:"
    cat "${INSTALL_DIR}/.p4config"
    echo "------------PLEASE EDIT--------------"
fi
