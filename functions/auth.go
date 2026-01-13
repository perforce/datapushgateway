package functions

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type User struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	URLPrefix string `yaml:"url_prefix"`
}

type AuthFile struct {
	Users []User `yaml:"users"`
}

var usersPasswords = map[string]string{}

func VerifyUserPass(username, password string) bool {
	wantPass, hasUser := usersPasswords[username]
	if !hasUser {
		return false
	}
	return wantPass == password
}

func ReadAuthFile(fname string) error {
	yfile, err := os.ReadFile(fname)
	if err != nil {
		log.Fatal(err)
	}

	authFile := AuthFile{}
	err = yaml.Unmarshal(yfile, &authFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, user := range authFile.Users {
		usersPasswords[user.Username] = user.Password
	}
	return nil
}
func HandleHTTP(w http.ResponseWriter, req *http.Request, logger *logrus.Logger, dataDir string) (string, string, error) {
	// Ensure that the request is a POST request
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return "", "", fmt.Errorf("Method not allowed")
	}

	user, pass, ok := req.BasicAuth()
	if !ok || !VerifyUserPass(user, pass) {
		w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", "", fmt.Errorf("Unauthorized")
	}

	query := req.URL.Query()
	customer := query.Get("customer")
	instance := query.Get("instance")
	if customer == "" || instance == "" {
		http.Error(w, "Please specify customer and instance", http.StatusBadRequest)
		return "", "", fmt.Errorf("Customer or Instance not specified")
	}
	// Whitelist check using regular expression
	var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(customer) || !validName.MatchString(instance) {
		http.Error(w, "Invalid characters in customer or instance name", http.StatusBadRequest)
		return "", "", fmt.Errorf("Invalid characters detected")
	}
	// All checks have passed
	return customer, instance, nil
}
