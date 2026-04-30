package repo

import "strings"

func isSQLiteFKViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func isSQLiteUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
