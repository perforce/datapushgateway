package functions

import (
	"bufio"
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

var p4ConfigPath string

func LoadConfig() error {
	configFile := "config.yaml"
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	var config map[string]string
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return err
	}

	path, ok := config["P4CONFIG"]
	if !ok || path == "" {
		return fmt.Errorf("P4CONFIG not found in config.yaml")
	}

	p4ConfigPath = path
	// Add a debug log statement to show the loaded .p4config path
	logger.Debugf("Loaded .p4config file: %s", p4ConfigPath)
	return nil
}

func P4Login(logger *logrus.Logger) error {
	os.Setenv("P4CONFIG", p4ConfigPath)

	// Check if already logged in using 'p4 login -s'
	loginStatusCmd := exec.Command("p4", "login", "-s")
	if err := loginStatusCmd.Run(); err == nil {
		logger.Info("Already logged in to Perforce.")
		return nil // Already logged in
	}

	// Handle trust if needed
	if err := handleP4Trust(logger); err != nil {
		return err
	}

	// Prompt for password and login
	fmt.Print("Enter Perforce password: ")
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read password: %v", err)
	}
	password = strings.TrimSpace(password)

	return runP4Login(password, logger)
}

func HasValidTicket(logger *logrus.Logger) bool {
	os.Setenv("P4CONFIG", p4ConfigPath)
	cmd := exec.Command("p4", "tickets")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Debugf("Error checking tickets: %s", output)
		return false
	}
	return strings.Contains(string(output), "ticket expires in")
}

func handleP4Trust(logger *logrus.Logger) error {
	// Check if trust is already established
	checkTrustCmd := exec.Command("p4", "trust", "-l")
	checkOutput, checkErr := checkTrustCmd.CombinedOutput()
	if checkErr == nil && strings.Contains(string(checkOutput), "Trust already established") {
		logger.Info("Perforce trust already established.")
		return nil // Trust is already established, no need to proceed further
	}

	// Establish trust
	cmd := exec.Command("p4", "trust", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf("Error running 'p4 trust': %v", err)
		logger.Errorf("Output: %s", output)
		return err
	}
	logger.Infof("p4 trust output: %s", output)
	return nil
}

func runP4Login(password string, logger *logrus.Logger) error {
	cmd := exec.Command("p4", "login", "-a")
	var stdin bytes.Buffer
	stdin.Write([]byte(password + "\n"))
	cmd.Stdin = &stdin

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Errorf("Error running 'p4 login': %v", err)
		logger.Errorf("Stderr: %s", stderr.String())
		return err
	}

	logger.Infof("Stdout: %s", stdout.String())
	return nil
}

func RunP4CommandWithEnvAndDir(command string, args []string, includeDataDir bool, dataDir string, customer string, logger *logrus.Logger) error {
	os.Setenv("P4CONFIG", p4ConfigPath)
	cmdArgs := make([]string, 0)
	if includeDataDir {
		dataFlag := filepath.Join(dataDir, customer)
		cmdArgs = append(cmdArgs, "-d", dataFlag)
	}
	cmdArgs = append(cmdArgs, args...)

	// Log the full command for debugging
	logger.Debugf("Executing P4 command: %s %v", command, cmdArgs)

	cmd := exec.Command(command, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf("Error executing command '%s %v': %v", command, cmdArgs, err)
		logger.Debugf("Command output: %s", string(output))
		return err
	}

	// Log command output
	logger.Debugf("Command output: %s", string(output))
	return nil
}

func P4SyncIT(p4Command, dataDir, customer, instance string, logger *logrus.Logger) error {
	recArgs := []string{"rec"}
	syncArgs := []string{"sync"}
	resolveArgs := []string{"resolve", "-ay"}
	customerDirPath := filepath.Join(dataDir, customer, "/...")

	// Run 'p4 rec'
	logger.Infof("Running P4 command: %s %s", p4Command, strings.Join(recArgs, " "))
	if err := RunP4CommandWithEnvAndDir(p4Command, recArgs, true, dataDir, customer, logger); err != nil {
		logger.Errorf("Error running 'p4 rec': %v", err)
		return err
	}

	// Run 'p4 sync'
	logger.Infof("Running P4 command: %s %s", p4Command, strings.Join(syncArgs, " "))
	if err := RunP4CommandWithEnvAndDir(p4Command, syncArgs, true, dataDir, customer, logger); err != nil {
		logger.Errorf("Error running 'p4 sync': %v", err)
		return err
	}

	// Run 'p4 resolve -ay'
	logger.Infof("Running P4 command: %s %s", p4Command, strings.Join(resolveArgs, " "))
	if err := RunP4CommandWithEnvAndDir(p4Command, resolveArgs, true, dataDir, customer, logger); err != nil {
		logger.Errorf("Error running 'p4 resolve -ay': %v", err)
		return err
	}

	// Check for changes to submit
	if hasChangesToSubmit(p4Command, customerDirPath, logger) {
		// Construct and execute the 'p4 submit' command
		submitCmdArgs := []string{
			"submit",
			"-d", fmt.Sprintf("Customer: %s, Instance: %s, monitoring submit", customer, instance),
			customerDirPath,
		}
		logger.Infof("Running P4 command: %s submit %s", p4Command, strings.Join(submitCmdArgs, " "))
		if err := RunP4CommandWithEnvAndDir(p4Command, submitCmdArgs, false, "", customer, logger); err != nil {
			logger.Errorf("Error running 'p4 submit': %v", err)
			return err
		}
	} else {
		logger.Info("No changes to submit.")
	}

	logger.Infof("P4 commands executed successfully")
	return nil
}

func hasChangesToSubmit(p4Command, customerDirPath string, logger *logrus.Logger) bool {
	cmdArgs := []string{"opened", customerDirPath}
	cmd := exec.Command(p4Command, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Debugf("Error checking for changes: %v", err)
		return false
	}
	return strings.Contains(string(output), "//")
}
