![Support](https://img.shields.io/badge/Support-Community-yellow.svg)


# DataPushGateway


DataPushGateway is an advanced integration tool designed for managing and organizing supplementary data within Perforce server environments. It primarily functions by sorting and structuring data provided by external tools such as command-runner or report_instance_data.sh, which are integral to the p4prometheus suite. This tool is key in organizing and presenting data in a coherent format, especially Markdown (MD) files, and stores them within a Perforce server, thereby aiding in tracking changes within the Helix Core server ecosystem.


The tool streamlines the process of consolidating server activities and configurations, making it a vital component for efficient data management in Perforce server ecosystems. Its integration with Helix Core versioning control software significantly enhances its capability to handle version-controlled data.


DataPushGateway is especially valuable for organizations that prioritize organized, well-documented server data, transforming raw supplementary data into structured documentation for easy access and comprehension by server administrators and auditors.


- [DataPushGateway](#datapushgateway)
- [Support Status](#support-status)
- [Overview](#overview)
- [Technical Overview](#technical-overview)
- [Key Features and Functions](#key-features-and-functions)
- [Detailed Installation Instructions](#detailed-installation-instructions)
- [API Documentation](#api-documentation)
- [Log Generation](#log-generation)


## Support Status


This is currently a Community Supported Perforce tool.


## Overview


DataPushGateway forms part of a comprehensive solution including the following components:
* [p4prometheus](https://www.github.com/perforce/p4prometheus/) - p4prometheus
* [CommandRunner](https://www.github.com/perforce/Command-Runner) - Command Runner
* [monitor_metrics.sh](demo/monitor_metrics.sh) - an [SDP](https://swarm.workshop.perforce.com/projects/perforce-software-sdp) compatible bash script to generate simple supplementary metrics


## Technical Overview


DataPushGateway serves as a companion to Prometheus Pushgateway, focusing on the management and organization of arbitrary data related to customer and instance names. It's primarily designed to be wrapped by a script that periodically checks in the result, with `report_instance_data.sh` or `command-runner` being the primary clients pushing data to this tool.


### Key Features and Functions:


- **Configurable Authentication and Port Settings**: Allows configuration through command-line flags for authentication files and port settings.
- **HTTP Server Setup**: Sets up an HTTP server to handle incoming requests, with middleware for logging connection details.
- **Data Processing and Endpoints**: Features endpoints for operational status confirmation and handling JSON and general data with basic authentication.
- **Data Validation and Storage**: Validates customer and instance names and saves received data to the filesystem.
- **Perforce Integration**: Synchronizes the saved data with Perforce Helix Core Server


## Detailed Installation Instructions


#### Create a DataPushGateway Bot User:


1. Create a new user (e.g., `bot_HRA_instance_monitor`) on your Perforce server for DataPushGateway to use. This user will be responsible for submitting changes.
    This user should be part of a Service user group without a Timeout set to Unlimited
2. Ensure this user/group has the necessary permissions to create and submit changes to the depot.


#### Create a Perforce Client:


1. Set up a Perforce client for the bot user. Here’s an example client specification:


    ```
    Client: bot_HRA_instance_monitor_ws
    Owner: bot_HRA_instance_monitor
    Root: /home/datapushgateway/data-dir
    Options: allwrite noclobber nocompress unlocked nomodtime rmdir noaltsync
    SubmitOptions: leaveunchanged
    LineEnd: local
    View:
        //datapushgateway/... //bot_HRA_instance_monitor_ws/...
    ```


2. Adjust the `Root` and `View` as per your server setup. The `Root` should match the directory where DataPushGateway will store its data.


Before starting the DataPushGateway, you need to set up the `config.yaml` file with the appropriate settings:


3. Set the `P4CONFIG` path in the `config.yaml` file. This path points to your `.p4config` file which contains necessary Perforce settings. Add the following line to `config.yaml`:


```
        P4CONFIG: /home/datapushgateway/.p4config
```


4. Ensure that other settings in `config.yaml` are correctly configured according to your environment and requirements.


The `auth.yaml` file needs to be configured with user credentials encrypted using bcrypt. This file is used for basic authentication when accessing DataPushGateway. Follow these steps to set up the `auth.yaml` file:


5. Generate a Bcrypt Encrypted Password:


   Use the `mkpasswd` binary located in the `tools` directory to create a bcrypt encrypted password. Run the following command in the terminal:


Follow the prompts to enter and confirm your password. The command will output a bcrypt encrypted password.


6. Configure `auth.yaml`:


Open the `auth.yaml` file and add your username and the bcrypt encrypted password in the following format:


```yaml
basic_auth_users:
  your_username: [bcrypt encrypted password]
```
7. **Start DataPushGateway with Authentication and Data Directory:**


   To start the DataPushGateway, use the following command, specifying the `auth.yaml` file and the data directory path:

```bash
./datapushgateway --auth.file=auth.yaml --data=/home/datapushgateway/data-dir
```


The `--debug` flag is optional and enables detailed logging.


   When you run DataPushGateway for the first time, the tool automatically manages the login process for the bot user (`bot_HRA_instance_monitor`). This includes handling passwords, tickets, and trusts as needed to ensure the user has valid credentials and permissions to submit changes to the depot.


   If the bot user's password is required during this process, DataPushGateway will prompt for it. This step is crucial for verifying that the bot user can access the depot and has the necessary permissions to submit changes.


   After this initial setup, DataPushGateway should operate autonomously, handling subsequent logins and permissions without manual intervention.



## Log Generation


```bash
./datapushgateway --auth.file=auth.yaml --data=/home/datapushgateway/data-dir > datapushgateway.log 2>&1 &
```

# API Documentation


DataPushGateway offers a set of HTTP endpoints designed for managing and organizing supplementary data in Perforce server environments. These endpoints facilitate the reception, processing, and synchronization of data with Perforce.


## Endpoints


### 1. Home Endpoint


- **URL**: `/`
- **Method**: `GET`
- **Description**: Provides operational status confirmation, returning a simple message indicating DataPushGateway's active status.
- **Response**:
  - `200 OK` - Returns "Data PushGateway\n" upon successful connection.


### 2. JSON Data Handling Endpoint


- **URL**: `/json/`
- **Method**: `POST`
- **Description**: Processes JSON formatted data related to customer and instance names.
- **Authentication**: Requires basic HTTP authentication.
- **Request Parameters**: None.
- **Request Body**: JSON formatted data.
- **Response**:
  - `200 OK` - Data processed successfully.
  - Error messages and status codes for various failures.


### 3. Data Submission and Synchronization Endpoint


- **URL**: `/data/`
- **Method**: `POST`
- **Description**: Submits data for saving and synchronization with Perforce. Validates customer and instance names.
- **Authentication**: Requires basic HTTP authentication.
- **Query Parameters**:
  - `customer` - Specifies the customer name.
  - `instance` - Specifies the instance name.
- **Request Body**: Arbitrary data.
- **Response**:
  - `200 OK` - Data saved and synced successfully with confirmation message.
  - `400 Bad Request` - Invalid or missing customer/instance names.
  - `401 Unauthorized` - Authentication failure.
  - `500 Internal Server Error` - Failures in saving or syncing data.


## Authentication


- Both the `/json/` and `/data/` endpoints require basic HTTP authentication.
- Users must provide a valid username and password as configured in the `auth.yaml` file.


### Development Notes:


- **Logging and Debugging**: Utilizes `logrus` for logging with a focus on enhancing logging functionality. Debugging mode can be enabled through a flag.
- **Error Handling**: Comprehensive error handling around data reading, saving, and syncing with Perforce.
- **Command-Line Interface**: Uses `kingpin.v2` for CLI handling, with various configuration flags.

### TODO
- Better User management
 - submit on directory attached to user
- Bug with directory structure and file names seems to run over each other

