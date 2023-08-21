package functions

import (
	"strings"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	logger.Level = logrus.InfoLevel
}

func SetDebugMode(debug bool) {
	if debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}
func maskSensitiveData(args []string) []string {
	maskedArgs := make([]string, len(args))
	copy(maskedArgs, args)
	for i, arg := range maskedArgs {
		if strings.Contains(arg, "P4PORT") ||
			strings.Contains(arg, "P4USER") ||
			strings.Contains(arg, "P4CLIENT") ||
			strings.Contains(arg, "P4TICKETS") ||
			strings.Contains(arg, "P4TRUST") ||
			strings.Contains(arg, "P4PASSWD") {
			maskedArgs[i] = strings.Split(arg, "=")[0] + "=******"
		}
	}
	return maskedArgs
}
