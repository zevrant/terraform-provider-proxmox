package services

import (
	"fmt"
	"strconv"
	"strings"
)

type ProxmoxUtilService interface {
	MapKeyValuePairsToMap(pairs []string) map[string]string
	MapBoolToProxmoxString(aBooleanValue bool) string
	MapProxmoxStringToBool(proxmoxBoolString string) bool
	ConvertSizeToGibibytes(sizeString string) int64
}

type ProxmoxUtilServiceImpl struct {
}

func NewProxmoxUtilService() ProxmoxUtilService {
	proxmoxUtils := ProxmoxUtilServiceImpl{}
	return &proxmoxUtils
}

func (proxmoxUtils *ProxmoxUtilServiceImpl) MapKeyValuePairsToMap(pairs []string) map[string]string {
	mappedPairs := make(map[string]string)

	for _, pair := range pairs {
		if strings.Contains(pair, "=") {
			splitPair := strings.Split(pair, "=")
			mappedPairs[splitPair[0]] = splitPair[1]
		} else {
			fmt.Println(fmt.Sprintf("Pair didn't contain '=': %s", pair))
		}
	}

	return mappedPairs
}

// MapBoolToProxmoxString
/**
 * @description proxmox uses the string value of 1 for true and 0 for false when dealing with booleans in their api
 * @param aBooleanValue: the value to be converted to strings used by proxmox
 *
 * @return 1 when true  0 when false
 */
func (proxmoxUtils *ProxmoxUtilServiceImpl) MapBoolToProxmoxString(aBooleanValue bool) string {
	if aBooleanValue {
		return "1"
	}
	return "0"
}

// MapProxmoxStringToBool
/**
 * @description proxmox uses the string value of 1 for true and 0 for false when dealing with booleans in their api
 * @param aProxmoxBoolString: the proxmox string value to be converted to a go boolean
 *
 * @return true when 1, false when 0
 */
func (proxmoxUtils *ProxmoxUtilServiceImpl) MapProxmoxStringToBool(proxmoxBoolString string) bool {
	return proxmoxBoolString == "1"
}

func (proxmoxUtils *ProxmoxUtilServiceImpl) ConvertSizeToGibibytes(sizeString string) int64 {
	unitLabel := sizeString[len(sizeString)-1:]
	size, _ := strconv.ParseInt(sizeString[:len(sizeString)-1], 10, 64)
	switch unitLabel {
	case "T":
		return size * 1024
	case "M":
		return size / 1024
	case "P":
		return size * 1024 * 1024
	}

	return size
}
