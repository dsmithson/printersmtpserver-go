package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	FilePath string
	SmtpPort int
}

func LoadSettings(args []string) (Config, error) {
	return Config{
		FilePath: getSetting(args, "outputpath", "/attachments/"),
		SmtpPort: getSettingInt(args, "port", 25),
	}, nil
}

func getSettingInt(args []string, settingName string, defaultValue int) int {
	val := getSetting(args, settingName, "")
	if val != "" {
		// parse the value as an integer
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// getSetting retrieves a setting from command line arguments
func getSetting(args []string, settingName, defaultValue string) string {
	// Look for a user-supplied argument
	if len(args) > 0 {
		argName := "--" + settingName
		for i := 0; i < len(args)-1; i++ {
			if args[i] == argName {
				return args[i+1]
			}
		}
	}

	// If not found in command-line args, look in environment variables
	envName := "SMTP_" + strings.ToUpper(settingName)
	if value, ok := os.LookupEnv(envName); ok {
		return value
	}

	return defaultValue
}
