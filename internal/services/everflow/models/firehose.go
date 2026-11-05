package models

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/nickarnoldi/funnel-service/internal/constants"
	"github.com/nickarnoldi/funnel-service/internal/utils"
)

// Int is a custom type for handling integers, that may or may not be in string form
type Int int32

func (i Int) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(i))
}

func (i *Int) UnmarshalJSON(data []byte) error {
	asString := strings.Trim(string(data), `"`)
	if asString == "" {
		*i = 0
		return nil
	} else {
		value, err := strconv.ParseInt(asString, 10, 64)
		if err != nil {
			log.Printf("Error parsing Int: %v", err)
			return err
		}
		*i = Int(value)
	}

	return nil
}

// BigInt is a custom type for handling big integers, that may or may not be in string form
type BigInt int64

func (i BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}

func (i *BigInt) UnmarshalJSON(data []byte) error {
	asString := strings.Trim(string(data), `"`)
	if asString == "" {
		*i = 0
		return nil
	} else {
		value, err := strconv.ParseInt(asString, 10, 64)
		if err != nil {
			log.Printf("Error parsing BigInt: %v", err)
			return err
		}
		*i = BigInt(value)
	}

	return nil
}

type AccountManager struct {
	ID                         BigInt `json:"network_employee_id,omitempty"`
	FirstName                  string `json:"first_name"`
	LastName                   string `json:"last_name"`
	Email                      string `json:"email"`
	WorkPhone                  string `json:"work_phone"`
	CellPhone                  string `json:"cell_phone"`
	InstantMessagingID         Int    `json:"instant_messaging_id"`
	InstantMessagingIdentifier string `json:"instant_messaging_identifier"`
}

type Address struct {
	ID            BigInt `json:"address_id,omitempty"`
	NetworkID     Int    `json:"network_id"`
	Address1      string `json:"address_1"`
	Address2      string `json:"address_2"`
	City          string `json:"city"`
	RegionCode    string `json:"region_code"`
	CountryID     Int    `json:"country_id"`
	CountryCode   string `json:"country_code"`
	ZipPostalCode string `json:"zip_postal_code"`
}

type Channels struct {
	Total   int       `json:"total"`
	Entries []Channel `json:"entries"`
}

type Channel struct {
	ID          BigInt `json:"network_channel_id,omitempty"`
	NetworkID   Int    `json:"network_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	TimeCreated int64  `json:"time_created"`
	TimeSaved   int64  `json:"time_saved"`
}

type Meta struct {
	OfferID                BigInt `json:"network_offer_id"`
	NetworkID              Int    `json:"network_id"`
	AppID                  string `json:"app_id"`
	MetaPlatform           string `json:"meta_platform"`
	AdvertiserCampaignName string `json:"advertiser_campaign_name"`
	BundleID               string `json:"bundle_id"`
	Category               string `json:"category"`
	StoreURL               string `json:"store_url"`
	ScoreRating            Int    `json:"score_rating"`
	TotalRating            Int    `json:"total_rating"`
	Size                   Int    `json:"size"`
	Developer              string `json:"developer"`
	Version                string `json:"version"`
	IsAppUnavailable       bool   `json:"is_app_unavailable"`
	TimeCreated            int64  `json:"time_created"`
	TimeAppDataSaved       int64  `json:"time_app_data_saved"`
	TimeSaved              int64  `json:"time_saved"`
}

type Category struct {
	ID          int    `json:"network_category_id,omitempty"`
	NetworkID   int    `json:"network_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	TimeCreated int64  `json:"time_created"`
	TimeSaved   int64  `json:"time_saved"`
}

type Labels struct {
	Total   int      `json:"total"`
	Entries []string `json:"entries"`
}

type PayoutRevenue struct {
	Total   int                  `json:"total"`
	Entries []PayoutRevenueEntry `json:"entries"`
}

type PayoutRevenueEntry struct {
	ID                             int     `json:"network_offer_payout_revenue_id"`
	NetworkID                      int     `json:"network_id"`
	OfferID                        int     `json:"network_offer_id"`
	EntryName                      string  `json:"entry_name"`
	PayoutType                     string  `json:"payout_type"`
	PayoutAmount                   float64 `json:"payout_amount"`
	PayoutPercentage               float64 `json:"payout_percentage"`
	RevenueType                    string  `json:"revenue_type"`
	RevenueAmount                  float64 `json:"revenue_amount"`
	RevenuePercentage              float64 `json:"revenue_percentage"`
	IsDefault                      bool    `json:"is_default"`
	IsPrivate                      bool    `json:"is_private"`
	IsPostbackDisabled             bool    `json:"is_postback_disabled"`
	IsEnforceCaps                  bool    `json:"is_enforce_caps"`
	TimeCreated                    int64   `json:"time_created"`
	GlobalAdvertiserEventID        int64   `json:"global_advertiser_event_id"`
	IsMustApproveConversion        bool    `json:"is_must_approve_conversion"`
	IsAllowDuplicateConversion     bool    `json:"is_allow_duplicate_conversion"`
	IsEmailAttributionDefaultEvent bool    `json:"is_email_attribution_default_event"`
	RemoteOfferResource            struct {
		OfferID          int    `json:"network_offer_id"`
		NetworkID        int    `json:"network_id"`
		ResourceType     string `json:"resource_type"`
		RemoteResourceID string `json:"remote_resource_id"`
		ResourceID       int    `json:"resource_id"`
		LastValueMD5     string `json:"last_value_md5"`
		JSONConfig       string `json:"json_config"`
		JSONData         string `json:"json_data"`
		TimeCreated      int64  `json:"time_created"`
		TimeSaved        int64  `json:"time_saved"`
	} `json:"remote_offer_resource"`
}

type ThumbnailAsset struct {
	AssetID               int    `json:"network_asset_id"`
	ContentType           string `json:"content_type"`
	Filename              string `json:"filename"`
	URL                   string `json:"url"`
	FileSize              int    `json:"file_size"`
	ImageWidth            int    `json:"image_width"`
	ImageHeight           int    `json:"image_height"`
	OptimizedThumbnailURL string `json:"optimized_thumbnail_url"`
}

type Geolocation struct {
	CountryCode  string `json:"countryCode"`
	CountryName  string `json:"countryName"`
	RegionCode   string `json:"regionCode"`
	RegionName   string `json:"regionName"`
	CityName     string `json:"cityName"`
	ISPName      string `json:"ispName"`
	DMA          int    `json:"dma"`
	PostalCode   string `json:"postalCode"`
	Timezone     string `json:"timezone"`
	CarrierName  string `json:"carrierName"`
	CarrierCode  int    `json:"carrierCode"`
	Organization string `json:"organization"`
	IsMobile     bool   `json:"isMobile"`
	IsProxy      bool   `json:"isProxy"`
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

type DeviceInfo struct {
	IsMobile           bool   `json:"isMobile"`
	PlatformName       string `json:"platformName"`
	OSVersion          string `json:"osVersion"`
	Brand              string `json:"brand"`
	Model              string `json:"model"`
	IsTablet           bool   `json:"isTablet"`
	BrowserName        string `json:"browserName"`
	BrowserVersion     string `json:"browserVersion"`
	DeviceType         string `json:"deviceType"`
	Language           string `json:"language"`
	HTTPAcceptLanguage string `json:"httpAcceptLanguage"`
	IsRobot            bool   `json:"isRobot"`
	IsFilter           bool   `json:"isFilter"`
}
