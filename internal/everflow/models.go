package everflow

// Response types keep the JSON names Everflow sends and the fields the
// importers read. Fields this service does not store are omitted;
// encoding/json ignores the extra keys.

type PagingInfo struct {
	TotalCount int `json:"total_count"`
}

type CategoryData struct {
	ID        int    `json:"network_category_id"`
	NetworkID int    `json:"network_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type CategoriesResponseBody struct {
	Categories []CategoryData `json:"categories"`
}

type AdvertisersRequestParams struct {
	PageSize int
}

type AdvertiserData struct {
	ID                     int    `json:"network_advertiser_id"`
	NetworkID              int    `json:"network_id"`
	Name                   string `json:"name"`
	Status                 string `json:"account_status"`
	DefaultCurrencyID      string `json:"default_currency_id"`
	AttributionMethod      string `json:"attribution_method"`
	EmailAttributionMethod string `json:"email_attribution_method"`
	AttributionPriority    string `json:"attribution_priority"`
}

type AdvertisersResponseBody struct {
	Advertisers []AdvertiserData `json:"advertisers"`
	Paging      PagingInfo       `json:"paging"`
}

type GetAffiliatesParams struct {
	PageSize int
}

type AffiliateData struct {
	ID            int    `json:"network_affiliate_id"`
	NetworkID     int    `json:"network_id"`
	Name          string `json:"name"`
	AccountStatus string `json:"account_status"`
}

type AffiliatesResponseBody struct {
	Affiliates []AffiliateData `json:"affiliates"`
	Paging     PagingInfo      `json:"paging"`
}

type OffersRequestParams struct {
	SearchTerms map[int]string
	Filters     map[string]string
	PageSize    int
}

type OffersRequestBody struct {
	SearchTerms map[int]string    `json:"search_terms"`
	Filters     map[string]string `json:"filters"`
}

type OfferData struct {
	ID                  int    `json:"network_offer_id"`
	NetworkID           int    `json:"network_id"`
	Name                string `json:"name"`
	Status              string `json:"offer_status"`
	Visibility          string `json:"visibility"`
	NetworkAdvertiserID int    `json:"network_advertiser_id"`
	Category            string `json:"category"`
	DestinationURL      string `json:"destination_url"`
	ThumbnailURL        string `json:"thumbnail_url"`
}

type OffersResponseBody struct {
	Offers []OfferData `json:"offers"`
	Paging PagingInfo  `json:"paging"`
}

type OfferURLsRequestParams struct {
	PageSize int
}

type OfferURLData struct {
	ID             int    `json:"network_offer_url_id"`
	OfferID        int    `json:"network_offer_id"`
	NetworkID      int    `json:"network_id"`
	Name           string `json:"name"`
	DestinationURL string `json:"destination_url"`
	PreviewURL     string `json:"preview_url"`
	Status         string `json:"url_status"`
}

type OfferURLsResponseBody struct {
	URLs   []OfferURLData `json:"urls"`
	Paging PagingInfo     `json:"paging"`
}

type GetConversionsByDateTimeParams struct {
	FromDate        string
	ToDate          string
	TimezoneID      int
	CurrencyID      string
	ShowConversions *bool
	ShowEvents      *bool
	PageSize        int
}

type ConversionRequestBody struct {
	From            string `json:"from"`
	To              string `json:"to"`
	TimezoneID      int    `json:"timezone_id"`
	CurrencyID      string `json:"currency_id"`
	ShowConversions bool   `json:"show_conversions"`
	ShowEvents      bool   `json:"show_events"`
}

type ConversionData struct {
	ConversionID            string  `json:"conversion_id"`
	ConversionUnixTimestamp int64   `json:"conversion_unix_timestamp"`
	Status                  string  `json:"status"`
	PayoutType              string  `json:"payout_type"`
	RevenueType             string  `json:"revenue_type"`
	Payout                  float64 `json:"payout"`
	Revenue                 float64 `json:"revenue"`
	ConversionUserIP        string  `json:"conversion_user_ip"`
	HTTPUserAgent           string  `json:"http_user_agent"`
	TransactionID           string  `json:"transaction_id"`
	URL                     string  `json:"url"`
	Relationship            struct {
		Offer struct {
			NetworkOfferID int    `json:"network_offer_id"`
			NetworkID      int    `json:"network_id"`
			Name           string `json:"name"`
		} `json:"offer"`
	} `json:"relationship"`
}

type ConversionResponseBody struct {
	Conversions []ConversionData `json:"conversions"`
	Paging      PagingInfo       `json:"paging"`
}

type GetClicksByDateTimeParams struct {
	FromDateTime string
	ToDateTime   string
	TimezoneID   int
}

type ClickRequestBody struct {
	From       string `json:"from"`
	To         string `json:"to"`
	TimezoneID int    `json:"timezone_id"`
}

type ClickData struct {
	TransactionID  string  `json:"transaction_id"`
	IsUnique       int     `json:"is_unique"` // Everflow sends 0 or 1. Importers treat any non-zero value as unique.
	UnixTimestamp  int64   `json:"unix_timestamp"`
	TrackingURL    string  `json:"tracking_url"`
	SourceID       string  `json:"source_id"`
	Sub1           string  `json:"sub1"`
	Sub2           string  `json:"sub2"`
	Sub3           string  `json:"sub3"`
	Sub4           string  `json:"sub4"`
	Sub5           string  `json:"sub5"`
	Payout         float64 `json:"payout"`
	Revenue        float64 `json:"revenue"`
	Referer        string  `json:"referer"`
	ErrorCode      int     `json:"error_code"`
	UserIP         string  `json:"user_ip"`
	URL            string  `json:"url"`
	IsViewThrough  bool    `json:"is_view_through"`
	IsAsync        bool    `json:"is_async"`
	ServerSideURL  string  `json:"server_side_url"`
	IsTestMode     bool    `json:"is_test_mode"`
	Idfa           string  `json:"idfa"`
	GoogleAdID     string  `json:"google_ad_id"`
	ErrorFilterID  string  `json:"error_filter_id"`
	CreativeID     int     `json:"creative_id"`
	CouponCode     string  `json:"coupon_code"`
	RedirectMethod string  `json:"redirect_method"`
	IsSdkClick     bool    `json:"is_sdk_click"`
	CategoryID     int     `json:"category_id"`
	Relationship   struct {
		Offer struct {
			NetworkID      int    `json:"network_id"`
			NetworkOfferID int    `json:"network_offer_id"`
			Name           string `json:"name"`
		} `json:"offer"`
		Advertiser struct {
			NetworkAdvertiserID int    `json:"network_advertiser_id"`
			Name                string `json:"name"`
		} `json:"advertiser"`
		Affiliate struct {
			NetworkAffiliateID int    `json:"network_affiliate_id"`
			Name               string `json:"name"`
		} `json:"affiliate"`
		Geolocation struct {
			CountryCode string `json:"country_code"`
			CountryName string `json:"country_name"`
			RegionCode  string `json:"region_code"`
			RegionName  string `json:"region_name"`
			CityName    string `json:"city_name"`
			Dma         int    `json:"dma"`
			PostalCode  string `json:"postal_code"`
			Timezone    string `json:"timezone"`
			CarrierName string `json:"carrier_name"`
			CarrierCode int    `json:"carrier_code"`
			IspName     string `json:"isp_name"`
			IsMobile    bool   `json:"is_mobile"`
			IsProxy     bool   `json:"is_proxy"`
		} `json:"geolocation"`
		// DeviceInformation is marshaled into the click row, so its fields and tags are the stored document.
		DeviceInformation struct {
			IsMobile           bool   `json:"is_mobile"`
			IsTablet           bool   `json:"is_tablet"`
			PlatformName       string `json:"platform_name"`
			OsVersion          string `json:"os_version"`
			Brand              string `json:"brand"`
			Model              string `json:"model"`
			BrowserName        string `json:"browser_name"`
			BrowserVersion     string `json:"browser_version"`
			DeviceType         string `json:"device_type"`
			Language           string `json:"language"`
			HTTPAcceptLanguage string `json:"http_accept_language"`
			IsRobot            bool   `json:"is_robot"`
			IsFilter           bool   `json:"is_filter"`
		} `json:"device_information"`
		HTTPUserAgent string `json:"http_user_agent"`
	} `json:"relationship"`
}

type ClickResponseBody struct {
	Clicks []ClickData `json:"clicks"`
}
