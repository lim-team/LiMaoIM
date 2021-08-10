package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntranetIP(t *testing.T) {
	ip, err := GetIntranetIP()
	assert.NoError(t, err)

	fmt.Println("ip--->", ip)
}

func TestGetExternalIP(t *testing.T) {
	ip, err := GetExternalIP()
	assert.NoError(t, err)

	fmt.Println("ip--->", ip)
}
