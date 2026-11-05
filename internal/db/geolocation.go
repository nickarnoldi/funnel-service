package db

import (
	"strings"

	"github.com/nickarnoldi/funnel-service/internal/constants"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
	"github.com/nickarnoldi/funnel-service/internal/utils"
)

// Native Geolocation Object
// Everflow version is camelCase, but we use snake_case in our database
type Geolocation struct {
	CountryCode  string `json:"country_code"`
	CountryName  string `json:"country_name"`
	RegionCode   string `json:"region_code"`
	RegionName   string `json:"region_name"`
	CityName     string `json:"city_name"`
	ISPName      string `json:"isp_name"`
	DMA          int    `json:"dma"`
	PostalCode   string `json:"postal_code"`
	Timezone     string `json:"timezone"`
	CarrierName  string `json:"carrier_name"`
	CarrierCode  int    `json:"carrier_code"`
	Organization string `json:"organization"`
	IsMobile     bool   `json:"is_mobile"`
	IsProxy      bool   `json:"is_proxy"`
}

func (geo *Geolocation) Normalize() {
	geo.CountryCode = strings.TrimSpace(geo.CountryCode)
	geo.RegionCode = strings.TrimSpace(geo.RegionCode)
	geo.CityName = strings.TrimSpace(geo.CityName)
	geo.PostalCode = strings.ToUpper(utils.RemoveAllWhitespace(geo.PostalCode))

	// Apparent bug, RegionCode is not being set correctly on some requests
	if geo.RegionCode == "" && geo.RegionName != "" {
		geo.RegionCode = geo.GetRegionCode()
	}
}

func (geo Geolocation) GetRegionCode() string {
	if geo.RegionCode != "" {
		return geo.RegionCode
	} else if geo.RegionName != "" {
		if region, ok := constants.CountryRegionCodes[geo.CountryCode]; ok {
			for code, name := range region.Regions {
				if name == geo.RegionName {
					return code
				}
			}
		}
	}

	return ""
}

func (geo *Geolocation) FromFirehose(geoData models.Geolocation) {
	geo.CountryCode = geoData.CountryCode
	geo.CountryName = geoData.CountryName
	geo.RegionCode = geoData.RegionCode
	geo.RegionName = geoData.RegionName
	geo.CityName = geoData.CityName
	geo.ISPName = geoData.ISPName
	geo.DMA = geoData.DMA
	geo.PostalCode = geoData.PostalCode
	geo.Timezone = geoData.Timezone
	geo.CarrierName = geoData.CarrierName
	geo.CarrierCode = geoData.CarrierCode
	geo.Organization = geoData.Organization
	geo.IsMobile = geoData.IsMobile
	geo.IsProxy = geoData.IsProxy
}
