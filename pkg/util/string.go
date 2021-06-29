package util

import "strings"

// GenerUUID 生成uuid
func GenerUUID() string {

	return strings.Replace(NewV4().String(), "-", "", -1)
}
