package functions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// TODO Lets login and save a tickets in a respective directory using the .p4config file
// P4Login logs into Perforce and generates a session ticket
func P4Login(username, password, p4Port string, logger *logrus.Logger) error {
	// Construct the p4 login command
	cmd := exec.Command("p4", "-p", p4Port, "login", "-a")

	// Create a buffer to write the password
	var stdin bytes.Buffer
	stdin.Write([]byte(password + "\n"))
	cmd.Stdin = &stdin

	// Capture the output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		logger.Errorf("Error running 'p4 login': %v", err)
		logger.Errorf("Stderr: %s", stderr.String())
		return err
	}

	logger.Infof("Stdout: %s", stdout.String())
	return nil
}

// TODO This package needs to be smarter.. Load the yaml to use it later and functions need to be a bit more flexiable
// RunP4CommandWithEnvAndDir runs a p4 command with specified arguments, environment variables,
// and an optional data directory.
func RunP4CommandWithEnvAndDir(command string, args []string, includeDataDir bool, dataDir string, customer string) error {

	// Read the environment variables from the config.yaml file in the root directory
	configFile := "config.yaml"
	configFilePath := filepath.Join(configFile)

	// Load the environment variables from the config.yaml file
	envVars := make(map[string]string)
	configFileData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(configFileData, &envVars)
	if err != nil {
		return err
	}

	// Create the environment variables for the command
	cmdEnv := os.Environ()
	for key, value := range envVars {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Create the command with arguments
	cmdArgs := []string{}
	if includeDataDir {
		dataFlag := filepath.Join(dataDir, customer)
		cmdArgs = append(cmdArgs, "-d", dataFlag)
	}

	// Add the -E flag with the environment variable value
	for key, value := range envVars {
		cmdArgs = append(cmdArgs, "-E", fmt.Sprintf("%s=%s", key, value))
	}

	cmdArgs = append(cmdArgs, args...)

	// Create the command
	cmd := exec.Command(command, cmdArgs...)
	cmd.Env = cmdEnv

	// Run the command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Sync up
// P4SyncIT is the function that runs the sequence of P4 commands for data processing.
func P4SyncIT(p4Command, dataDir, customer, instance string, logger *logrus.Logger) error {
	// Read the environment variables from the config.yaml file in the root directory
	configFile := "config.yaml"
	configFilePath := filepath.Join(configFile)

	// Load the environment variables from the config.yaml file
	envVars := make(map[string]string)
	configFileData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(configFileData, &envVars)
	if err != nil {
		return err
	}

	// Define the P4 arguments for each command
	recArgs := []string{"rec"}
	syncArgs := []string{"sync"}
	resolveArgs := []string{"resolve", "-ay"}
	customerDirPath := filepath.Join(dataDir, customer, "/...")

	//p4 rec
	//	logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(recArgs, " "))
	logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(maskSensitiveData(recArgs), " "))

	err = RunP4CommandWithEnvAndDir(p4Command, recArgs, true, dataDir, customer)
	if err != nil {
		logger.Errorf("Error running 'p4 rec': %v", err)
		return err
	}

	// p4 sync
	//logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(syncArgs, " "))
	logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(maskSensitiveData(syncArgs), " "))
	err = RunP4CommandWithEnvAndDir(p4Command, syncArgs, true, dataDir, customer)
	if err != nil {
		logger.Errorf("Error running 'p4 sync': %v", err)
		return err
	}

	// p4 resolve
	//logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(resolveArgs, " "))
	logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(maskSensitiveData(resolveArgs), " "))
	err = RunP4CommandWithEnvAndDir(p4Command, resolveArgs, true, dataDir, customer)
	if err != nil {
		logger.Errorf("Error running 'p4 resolve -ay': %v", err)
		return err
	}
	////
	//// TODO No files to submit from the default changelist.
	//// TODO Fix Error because there is no reason to submit
	////
	// Construct and execute the submit command
	submitCmdArgs := []string{
		"submit",
		"-d", fmt.Sprintf("Customer: %s, Instance: %s, monitoring submit", customer, instance),
		customerDirPath,
	}

	submitCmdWithEnv := []string{} // Create a new slice to hold the -E flag environment variables
	for key, value := range envVars {
		submitCmdWithEnv = append(submitCmdWithEnv, "-E", fmt.Sprintf("%s=%s", key, value))
	}
	submitCmdWithEnv = append(submitCmdWithEnv, submitCmdArgs...) // Add the rest of the command arguments

	//logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(submitCmdWithEnv, " "))
	logger.Infof("Running P4 command: %s %s\n", p4Command, strings.Join(maskSensitiveData(submitCmdWithEnv), " "))

	submitCmd := exec.Command(p4Command, submitCmdWithEnv...)
	submitCmd.Env = os.Environ() // Use the current environment variables
	submitCmd.Stdout = os.Stdout
	submitCmd.Stderr = os.Stderr
	err = submitCmd.Run()
	if err != nil {
		logger.Errorf("Error running 'p4 submit': %v", err)
		return err
	}

	logger.Infof("P4 commands executed successfully")
	return nil
}
