package enum

import "strings"

type DeviceTypeEnum int

func (s DeviceTypeEnum) String() string {
	switch DeviceType[s] {
	case "apple":
		return "Apple"
	case "android":
		return "Android"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return "Other"
	}
}

func (d *DeviceTypeEnum) FromString(s string) {
	s = strings.ToLower(s)
	for index, name := range DeviceType {
		if name == s {
			*d = index
			return
		}
	}
	*d = DeviceTypeOther
}

const (
	DeviceTypeOther DeviceTypeEnum = iota
	DeviceTypeApple
	DeviceTypeAndroid
	DeviceTypeWindows
	DeviceTypeLinux
)

var DeviceType = map[DeviceTypeEnum]string{
	DeviceTypeOther:   "other",
	DeviceTypeApple:   "apple",
	DeviceTypeAndroid: "android",
	DeviceTypeWindows: "windows",
	DeviceTypeLinux:   "linux",
}
