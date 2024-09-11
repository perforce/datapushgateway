// This is a companion to prometheus pushgateway
// It is aimed to allow the saving of some arbitrary data specifying customer and instance names
// The aim is to be wrapped by a script which checks in the result on a regular basis.
// The client which is pushing data to this tool via curl is report_instance_data.sh
package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"datapushgateway/functions"

	"github.com/perforce/p4prometheus/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

// TODO: Better Logging
// TODO: Syncing in data.go

// We extract the bcrypt passwords from the config file used for prometheus pushgateway
// A very simple yaml structure.

// logger is declared at the package level for the main function.
var logger *logrus.Logger

func main() {
	var (
		authFile = kingpin.Flag(
			"auth.file",
			"Auth config file for pushgateway specifying user_basic_auth and list of user/bcrypt passwords.",
		).Short('a').Default("auth.yaml").String()
		configFile = kingpin.Flag(
			"config",
			"Config file specifying location of p4config and various mapping locations.",
		).Short('c').Default("config.yaml").String()
		port = kingpin.Flag(
			"port",
			"Port to listen on.",
		).Default(":9092").String()
		debug = kingpin.Flag(
			"debug",
			"Enable debugging.",
		).Bool()
		dataDir = kingpin.Flag(
			"data",
			"Directory where to store uploaded data.",
		).Short('d').Default("data").String()
	)

	kingpin.Version(version.Print("datapushgateway"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Create the logger after parsing the debug flag
	logger = logrus.New()
	if *debug {
		logger.Level = logrus.DebugLevel
		logger.Debug("Debugging is enabled")
	} else {
		logger.Level = logrus.InfoLevel
	}
	functions.SetDebugMode(*debug)

	config, err := functions.LoadConfig(*configFile)
	if err != nil {
		logger.Fatalf("Error loading config file %s: %v", *configFile, err)
	}

	err = functions.ReadAuthFile(*authFile)
	if err != nil {
		logger.Fatal(err)
	}
	// Ensure Perforce login
	if !functions.HasValidTicket(logger) {
		if err := functions.P4Login(logger); err != nil {
			logger.Fatalf("Failed to log in to Perforce: %v", err)
		}
	}
	mux := http.NewServeMux()

	// Middleware for logging connection details
	ConnectionLoggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if logger.Level == logrus.DebugLevel {
				logger.Debugf("Connection from %s", req.RemoteAddr)
				logger.Debugf("URL: %s", req.URL)
				logger.Debugf("Method: %s", req.Method)
			}
			next.ServeHTTP(w, req)
		}
	}

	mux.HandleFunc("/", ConnectionLoggingMiddleware(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		w.WriteHeader(200)
		fmt.Fprintf(w, "Data PushGateway\n")
	}))

	mux.HandleFunc("/json/", ConnectionLoggingMiddleware(func(w http.ResponseWriter, req *http.Request) {
		customer, instance, err := functions.HandleHTTP(w, req, logger, *dataDir)
		if err != nil {
			return
		}
		functions.HandleJSONData(w, req, logger, *configFile, *dataDir, customer, instance)
	}))

	mux.HandleFunc("/data/", ConnectionLoggingMiddleware(func(w http.ResponseWriter, req *http.Request) {
		var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

		user, pass, ok := req.BasicAuth()
		if ok && functions.VerifyUserPass(user, pass) {
			logger.Debugf("Basic auth verified for user: %s", user)
			query := req.URL.Query()
			customer := query.Get("customer")
			instance := query.Get("instance")

			// Validate the customer and instance parameters
			if customer == "" || instance == "" || !validName.MatchString(customer) || !validName.MatchString(instance) {
				http.Error(w, "Invalid or missing customer or instance name", http.StatusBadRequest)
				return
			}

			// Read the body of the request
			body, err := io.ReadAll(req.Body)
			if err != nil {
				logger.Errorf("Error reading body: %v", err)
				http.Error(w, "Cannot read body", http.StatusBadRequest)
				return
			}
			logger.Debugf("Request Body: %s", string(body))

			// Save the data received to the filesystem
			logger.Debugf("Saving data to dataDir: %s, customer: %s", *dataDir, customer)
			err = functions.SaveData(*dataDir, customer, instance, string(body), logger)
			if err != nil {
				logger.Errorf("Error saving data: %v", err)
				http.Error(w, "Failed to save data", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("Data saved"))

			// Synchronize the saved data with Perforce
			err = functions.P4SyncIT(config.ApplicationConfig.P4Bin, *dataDir, customer, instance, logger)
			if err != nil {
				logger.Errorf("P4SyncIT error: %v", err)
				http.Error(w, "Error syncing data with Perforce", http.StatusInternalServerError)
				return
			}
			w.Write([]byte("Data synced with Perforce"))
		} else {
			// Prompt for basic auth if verification fails
			w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}))

	srv := &http.Server{
		Addr:    *port,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
		},
	}

	logger.Infof("Starting server on %s", *port)
	err = srv.ListenAndServe()
	if err != nil {
		logger.Fatal(err)
	}
}
