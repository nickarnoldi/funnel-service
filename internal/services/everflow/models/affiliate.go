package models

type Affiliate struct {
	ID                           int    `json:"network_affiliate_id"`
	NetworkID                    int    `json:"network_id"`
	Name                         string `json:"name"`
	Status                       string `json:"account_status"`
	EmployeeID                   int    `json:"network_employee_id"`
	InternalNotes                string `json:"internal_notes"`
	HasNotifications             bool   `json:"has_notifications"`
	TrafficSourceID              int    `json:"network_traffic_source_id"`
	AccountExecutiveID           int    `json:"account_executive_id"`
	AddressID                    int    `json:"adress_id"` /* typo in original; address_id */
	DefaultCurrencyID            string `json:"default_currency_id"`
	IsContactAddressEnabled      bool   `json:"is_contact_address_enabled"`
	EnableMediaCostTrackingLinks bool   `json:"enable_media_cost_tracking_links"`
	ReferrerID                   int    `json:"referrer_id"`
	TimeCreated                  int64  `json:"time_created"`
	TimeSaved                    int64  `json:"time_saved"`
	Relationship                 struct {
		Labels         Labels         `json:"labels"`
		AccountManager AccountManager `json:"account_manager"`
		ContactAddress Address        `json:"contact_address"`
		EncodedValue   string         `json:"encoded_value"`
	} `json:"relationship"`
}
