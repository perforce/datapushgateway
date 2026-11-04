package functions

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSystemUpdateTagRouting(t *testing.T) {
	items := []struct {
		tag         string
		description string
		output      string
		fileName    string
	}{
		{"kernel release", "Running kernel release", "6.8.0-88-generic", "kernel-info.md"},
		{"kernel packages", "Installed kernel packages", "linux-image-generic 6.8.0-88.89", "kernel-info.md"},
		{"reboot required", "Reboot required status", "No reboot required", "kernel-info.md"},
		{"security updates", "Available security updates", "No security updates available", "package-updates.md"},
	}
	for _, configFile := range []string{"../config.yaml", "../setup/datapushgateway/config.yaml"} {
		t.Run(configFile, func(t *testing.T) {
			dataDir := t.TempDir()
			customer, instance := "eve.games", "eve.games-helixcore-useast1"
			dataMap := make(map[string]string)
			for _, item := range items {
				encoded, err := json.Marshal(map[string]string{
					"monitor_tag": item.tag,
					"description": item.description,
					"output":      base64.StdEncoding.EncodeToString([]byte(item.output)),
				})
				if err != nil {
					t.Fatal(err)
				}
				dataMap[item.tag] = string(encoded)
			}
			ProcessDataMap(dataMap, configFile, dataDir, logrus.New(), customer, instance)

			reportPath := filepath.Join(dataDir, customer, "servers", "HRA-"+instance+".md")
			report, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			previousIndex := -1
			for _, item := range items {
				section := "# " + item.description + "\n```\n" + item.output + "\n```\n"
				index := strings.Index(string(report), section)
				if index < 0 || index <= previousIndex {
					t.Errorf("HRA report missing or misordered section for %q", item.tag)
				}
				previousIndex = index
				content, err := os.ReadFile(filepath.Join(dataDir, customer, "servers", instance, "info", item.fileName))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(content), section) {
					t.Errorf("%s missing decoded output for %q", item.fileName, item.tag)
				}
			}
			unmatchedPath := filepath.Join(dataDir, customer, "servers", instance, "info", "unmatched.md")
			if _, err := os.Stat(unmatchedPath); !os.IsNotExist(err) {
				t.Errorf("unexpected unmatched report: %v", err)
			}
		})
	}
}
