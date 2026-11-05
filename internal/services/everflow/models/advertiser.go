package models

type Advertiser struct {
	ID                             BigInt `json:"network_advertiser_id"`
	NetworkID                      Int    `json:"network_id"`
	Name                           string `json:"name"`
	Status                         string `json:"account_status"`
	EmployeeID                     BigInt `json:"network_employee_id"`
	InternalNotes                  string `json:"internal_notes"`
	AddressID                      BigInt `json:"address_id"`
	IsContactAddressEnabled        bool   `json:"is_contact_address_enabled"`
	SalesManagerID                 BigInt `json:"sales_manager_id"`
	IsExposePublisherReportingData bool   `json:"is_expose_publisher_reporting_data"`
	DefaultCurrencyID              string `json:"default_currency_id"`
	PlatformName                   string `json:"platform_name"`
	PlatformURL                    string `json:"platform_url"`
	PlatformUsername               string `json:"platform_username"`
	ReportingTimezoneID            BigInt `json:"reporting_timezone_id"`
	AccountingContactEmail         string `json:"accounting_contact_email"`
	VerificationToken              string `json:"verification_token"`
	OfferIDMacro                   string `json:"offer_id_macro"`
	AffiliateIDMacro               string `json:"affiliate_id_macro"`
	TimeCreated                    BigInt `json:"time_created"`
	TimeSaved                      BigInt `json:"time_saved"`
	Relationship                   struct {
		Labels         Labels         `json:"labels"`
		AccountManager AccountManager `json:"account_manager"`
		SalesManager   AccountManager `json:"sales_manager"`
	} `json:"relationship"`
	AttributionMethod      string `json:"attribution_method"`
	EmailAttributionMethod string `json:"email_attribution_method"`
	AttributionPriority    string `json:"attribution_priority"`
}
