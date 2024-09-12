#!/bin/bash

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

# Function to back up existing files and confirm overwriting
backup_if_exists() {
    local file=$1
    if [ -f "$file" ]; then
        echo "$file already exists."
        read -p "Do you want to overwrite this file? (y/N): " overwrite
        if [[ "$overwrite" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            local backup_file="${file}.bak.$(date +%Y%m%d%H%M%S)"
            echo "Backing up $file to $backup_file"
            run_command "cp $file $backup_file"
            return 0  # Return 0 to indicate file can be overwritten
        else
            echo "Skipping overwrite of $file."
            return 1  # Return 1 to indicate file will not be overwritten
        fi
    fi
    return 0  # Return 0 if file doesn't exist
}

# Check if script is run with sudo
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root or with sudo."
    exit 1
fi

# Prompt for install directories or use defaults
WORKING_INSTALL_DIR=$(prompt "Enter WorkingDirectory installation directory" "/opt/perforce/datapushgateway")
CONF_INSTALL_DIR=$(prompt "Enter config.yaml installation directory" "/opt/perforce/datapushgateway")
CONF_FILENAME=$(prompt "Enter name for config file" "config.yaml")
AUTH_INSTALL_DIR=$(prompt "Enter auth.yaml installation directory" "/opt/perforce/datapushgateway")
AUTH_FILENAME=$(prompt "Enter name for auth file" "auth.yaml")
P4CONFIG_INSTALL_DIR=$(prompt "Enter .P4CONFIG installation directory" "/opt/perforce/datapushgateway")
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

# Create necessary directories and check if they were successfully created
create_directory() {
    local dir=$1
    if $DRY_RUN; then
        echo "[DRY RUN] mkdir -p $dir"
    else
        mkdir -p "$dir"
        if [ ! -d "$dir" ]; then
            echo "Failed to create $dir"
            exit 1
        fi
    fi
}

create_directory "${CONF_INSTALL_DIR}"
create_directory "${AUTH_INSTALL_DIR}"
create_directory "${P4CONFIG_INSTALL_DIR}"
create_directory "${WORKING_INSTALL_DIR}"
create_directory "${DATA_DIR}"

# Backup files if they already exist and ask for confirmation to overwrite
backup_if_exists "${AUTH_INSTALL_DIR}/auth.yaml"
if [ $? -eq 0 ]; then
    # Copy auth.yaml only if user allowed overwrite
    run_command "install -m 0644 auth.yaml ${AUTH_INSTALL_DIR}/${AUTH_FILENAME}"
    # Secure the auth.yaml file by restricting permissions
    run_command "chmod 600 ${AUTH_INSTALL_DIR}/${AUTH_FILENAME}"
else
    echo "Continuing without overwriting ${AUTH_INSTALL_DIR}/auth.yaml"
fi

backup_if_exists "${CONF_INSTALL_DIR}/config.yaml"
if [ $? -eq 0 ]; then
    # Copy config.yaml only if user allowed overwrite
    run_command "install -m 0644 config.yaml ${CONF_INSTALL_DIR}/${CONF_FILENAME}"
    # Secure the config.yaml file by restricting permissions
    run_command "chmod 600 ${CONF_INSTALL_DIR}/${CONF_FILENAME}"
else
    echo "Continuing without overwriting ${CONF_INSTALL_DIR}/config.yaml"
fi

backup_if_exists "${P4CONFIG_INSTALL_DIR}/.p4config"
if [ $? -eq 0 ]; then
    # Copy .p4config only if user allowed overwrite
    run_command "install -m 0644 .p4config ${P4CONFIG_INSTALL_DIR}/.p4config"
    # Secure the .p4config file by restricting permissions
    run_command "chmod 600 ${P4CONFIG_INSTALL_DIR}/.p4config"
else
    echo "Continuing without overwriting ${P4CONFIG_INSTALL_DIR}/.p4config"
fi

backup_if_exists "${BIN_DIR}/datapushgateway"
if [ $? -eq 0 ]; then
    # Copy binary only if user allowed overwrite
    run_command "install -m 0755 datapushgateway ${BIN_DIR}/datapushgateway"
else
    echo "Continuing without overwriting ${BIN_DIR}/datapushgateway"
fi

# Set ownership for the files and directories to the Service User
# Ensure all files in INSTALL_DIR and DATA_DIR are owned by SERVICE_USER
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${CONF_INSTALL_DIR}"
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${AUTH_INSTALL_DIR}"
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${P4CONFIG_INSTALL_DIR}/.p4config"
run_command "chown -R ${SERVICE_USER}:${SERVICE_USER} ${DATA_DIR}"

# SELinux adjustments
if selinuxenabled; then
    echo "Applying SELinux policies..."
    run_command "semanage fcontext -a -t bin_t \"${BIN_DIR}/datapushgateway\"" || echo "Failed to apply SELinux policy for ${BIN_DIR}/datapushgateway"
    run_command "semanage fcontext -a -t etc_t \"${CONF_INSTALL_DIR}(/.*)?\"" || echo "Failed to apply SELinux policy for ${CONF_INSTALL_DIR}"
    run_command "semanage fcontext -a -t etc_t \"${AUTH_INSTALL_DIR}(/.*)?\"" || echo "Failed to apply SELinux policy for ${AUTH_INSTALL_DIR}"
    run_command "semanage fcontext -a -t etc_t \"${P4CONFIG_INSTALL_DIR}(/.*)?\"" || echo "Failed to apply SELinux policy for ${P4CONFIG_INSTALL_DIR}"
    run_command "semanage fcontext -a -t var_lib_t \"${DATA_DIR}(/.*)?\"" || echo "Failed to apply SELinux policy for ${DATA_DIR}"
    run_command "restorecon -Rv ${BIN_DIR}/datapushgateway ${CONF_INSTALL_DIR} ${AUTH_INSTALL_DIR} ${P4CONFIG_INSTALL_DIR} ${DATA_DIR}" || echo "Failed to restore SELinux context"
fi

# Create systemd service file
if $DRY_RUN; then
    echo "[DRY RUN] Creating service file at ${SERVICE_FILE} with the following content:"
    cat <<EOF
[Unit]
Description=Data Push Gateway Service
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/datapushgateway -c ${CONF_INSTALL_DIR}/${CONF_FILENAME} -a ${AUTH_INSTALL_DIR}/${AUTH_FILENAME} -d ${DATA_DIR} > ${WORKING_INSTALL_DIR}/datapushgateway.log 2>&1
Restart=on-failure
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${WORKING_INSTALL_DIR}
Environment="P4CONFIG=${P4CONFIG_INSTALL_DIR}/.p4config"

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
ExecStart=${BIN_DIR}/datapushgateway -c ${CONF_INSTALL_DIR}/${CONF_FILENAME} -a ${AUTH_INSTALL_DIR}/${AUTH_FILENAME} -d ${DATA_DIR} > ${WORKING_INSTALL_DIR}/datapushgateway.log 2>&1
Restart=on-failure
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${WORKING_INSTALL_DIR}
Environment="P4CONFIG=${P4CONFIG_INSTALL_DIR}/.p4config"

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd to recognize the new service
run_command "systemctl daemon-reload"

# Enable the service but do not start it immediately
# run_command "systemctl enable datapushgateway"

echo "Installation completed."
if $DRY_RUN; then
    echo "Dry run completed. No changes were made."
fi

# If not in dry run, echo the contents of the configuration files
if ! $DRY_RUN; then
    echo "Contents of ${AUTH_INSTALL_DIR}/auth.yaml:"
    cat "${AUTH_INSTALL_DIR}/auth.yaml"
    echo "------------PLEASE EDIT--------------"

    echo "Contents of ${CONF_INSTALL_DIR}/config.yaml:"
    cat "${CONF_INSTALL_DIR}/config.yaml"
    echo "------------PLEASE EDIT--------------"

    echo "Contents of ${P4CONFIG_INSTALL_DIR}/.p4config:"
    cat "${P4CONFIG_INSTALL_DIR}/.p4config"
    echo "------------PLEASE EDIT--------------"
fi
