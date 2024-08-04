package utils

import (
	"path/filepath"
	"strings"
)

// cleanEmailString removes unnecessary characters from the email string
func CleanEmailString(emailLine, prefixToRemove string) string {
	return strings.TrimSpace(strings.TrimPrefix(emailLine, prefixToRemove))
}

// convertEmailRecipientToFolderName creates a folder name from the email recipient
func ConvertEmailRecipientToFolderName(email string) string {
	name := strings.Join(strings.Split(email, string(filepath.Separator)), " ")
	if name == "" {
		name = "Unknown"
	}
	return name
}
