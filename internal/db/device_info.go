package db

import "github.com/nickarnoldi/funnel-service/internal/services/everflow/models"

// Native Device Information Object
// Everflow version is camelCase, but we use snake_case in our database
type DeviceInfo struct {
	IsMobile           bool   `json:"is_mobile"`
	PlatformName       string `json:"platform_name"`
	OSVersion          string `json:"os_version"`
	Brand              string `json:"brand"`
	Model              string `json:"model"`
	IsTablet           bool   `json:"is_tablet"`
	BrowserName        string `json:"browser_name"`
	BrowserVersion     string `json:"browser_version"`
	DeviceType         string `json:"device_type"`
	Language           string `json:"language"`
	HTTPAcceptLanguage string `json:"http_accept_language"`
	IsRobot            bool   `json:"is_robot"`
	IsFilter           bool   `json:"is_filter"`
}

func (dev *DeviceInfo) FromFirehose(devData models.DeviceInfo) {
	dev.IsMobile = devData.IsMobile
	dev.PlatformName = devData.PlatformName
	dev.OSVersion = devData.OSVersion
	dev.Brand = devData.Brand
	dev.Model = devData.Model
	dev.IsTablet = devData.IsTablet
	dev.BrowserName = devData.BrowserName
	dev.BrowserVersion = devData.BrowserVersion
	dev.DeviceType = devData.DeviceType
	dev.Language = devData.Language
	dev.HTTPAcceptLanguage = devData.HTTPAcceptLanguage
	dev.IsRobot = devData.IsRobot
	dev.IsFilter = devData.IsFilter
}
