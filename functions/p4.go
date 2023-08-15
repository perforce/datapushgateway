package functions

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

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
