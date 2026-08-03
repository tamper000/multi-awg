package utils

import "regexp"

var validName = regexp.MustCompile(`^[a-zA-Zа-яёА-ЯЁ0-9]+$`)

func ValidName(name string) bool {
	return validName.MatchString(name)
}
