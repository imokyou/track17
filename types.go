package track17

// Tracking status constants represent the main status of a shipment.
const (
	// StatusNotFound indicates no tracking information is available yet.
	StatusNotFound = 0

	// StatusInTransit indicates the package is in transit.
	StatusInTransit = 10

	// StatusExpired indicates tracking has expired.
	StatusExpired = 20

	// StatusPickedUp indicates the package has been picked up.
	StatusPickedUp = 30

	// StatusUndeliverable indicates the package is undeliverable.
	StatusUndeliverable = 35

	// StatusDelivered indicates the package has been delivered.
	StatusDelivered = 40

	// StatusAlert indicates an alert/exception on the package.
	StatusAlert = 50
)

// Sub-status constants provide more granular tracking status information.
const (
	SubStatusNotFound_Other          = 0
	SubStatusInTransit_Other         = 0
	SubStatusInTransit_PickedUp      = 1
	SubStatusInTransit_DepartOrigin  = 2
	SubStatusInTransit_ArriveOrigin  = 3
	SubStatusInTransit_DepartTransit = 4
	SubStatusInTransit_ArriveTransit = 5
	SubStatusInTransit_DepartDest    = 6
	SubStatusInTransit_ArriveDest    = 7
	SubStatusInTransit_CustomsStart  = 8
	SubStatusInTransit_CustomsEnd    = 9
	SubStatusInTransit_OutDelivery   = 10
	SubStatusDelivered_Other         = 0
	SubStatusDelivered_Signed        = 1
	SubStatusDelivered_DropOff       = 2
	SubStatusDelivered_Locker        = 3
	SubStatusAlert_Other             = 0
	SubStatusAlert_AddressIssue      = 1
	SubStatusAlert_ContactCarrier    = 2
	SubStatusAlert_Delayed           = 3
	SubStatusAlert_Returning         = 4
	SubStatusAlert_Returned          = 5
	SubStatusAlert_CustomsIssue      = 6
	SubStatusAlert_Lost              = 7
	SubStatusAlert_Damaged           = 8
	SubStatusAlert_Held              = 9
)

// TrackInfo represents the complete tracking information for a shipment.
type TrackInfo struct {
	// Number is the tracking number.
	Number string `json:"number"`

	// Carrier is the carrier code.
	Carrier int `json:"carrier"`

	// Param is an optional secondary carrier code.
	Param int `json:"param,omitempty"`

	// Tag is a custom tag/label set by the user.
	Tag string `json:"tag,omitempty"`

	// Track contains the tracking details.
	Track *TrackDetail `json:"track,omitempty"`
}

// TrackDetail contains the detailed tracking information.
type TrackDetail struct {
	// Status indicates whether tracking data is available.
	// "Y" means data is available, "N" means not yet.
	Status string `json:"stat,omitempty"`

	// LatestStatus is the latest main tracking status.
	LatestStatus int `json:"b,omitempty"`

	// LatestSubStatus is the latest sub-status.
	LatestSubStatus int `json:"c,omitempty"`

	// LatestEvent is the latest tracking event description.
	LatestEvent string `json:"z0,omitempty"`

	// IsReturning indicates whether the package is being returned.
	IsReturning bool `json:"is1,omitempty"`

	// Events contains the full list of tracking events.
	Events []TrackEvent `json:"z1,omitempty"`

	// EventsTranslated contains translated tracking events.
	EventsTranslated []TrackEvent `json:"z2,omitempty"`

	// Milestone contains key milestone timestamps.
	Milestone *Milestone `json:"ygt1,omitempty"`

	// TransitDays is the total transit time in days.
	TransitDays int `json:"ygt2,omitempty"`

	// OriginCountry is the origin country code.
	OriginCountry string `json:"ln1,omitempty"`

	// DestCountry is the destination country code.
	DestCountry string `json:"ln2,omitempty"`

	// ServiceType is the carrier service type.
	ServiceType string `json:"ln3,omitempty"`

	// Weight is the package weight information.
	Weight string `json:"ln4,omitempty"`
}

// TrackEvent represents a single tracking event in the shipment history.
type TrackEvent struct {
	// Time is the event time in ISO format.
	Time string `json:"a,omitempty"`

	// TimeUTC is the event time in UTC format.
	TimeUTC string `json:"b,omitempty"`

	// TimeRaw is the raw time string from the carrier.
	TimeRaw string `json:"c,omitempty"`

	// Status is the event status code.
	Status int `json:"d,omitempty"`

	// SubStatus is the event sub-status code.
	SubStatus int `json:"e,omitempty"`

	// Location is the event location.
	Location string `json:"z,omitempty"`

	// Description is the event description.
	Description string `json:"a0,omitempty"`
}

// Milestone contains key milestone timestamps for a shipment.
type Milestone struct {
	// PickedUp is the timestamp when the package was picked up.
	PickedUp string `json:"a,omitempty"`

	// DepartOrigin is the timestamp when the package departed origin.
	DepartOrigin string `json:"b,omitempty"`

	// ArriveDest is the timestamp when the package arrived at destination.
	ArriveDest string `json:"c,omitempty"`

	// CustomsClearance is the timestamp when customs clearance was completed.
	CustomsClearance string `json:"d,omitempty"`

	// OutForDelivery is the timestamp when the package was out for delivery.
	OutForDelivery string `json:"e,omitempty"`

	// Delivered is the timestamp when the package was delivered.
	Delivered string `json:"f,omitempty"`
}

// QuotaInfo represents the account quota information.
type QuotaInfo struct {
	// Total is the total quota allocated.
	Total int `json:"total_count,omitempty"`

	// Used is the number of quota used.
	Used int `json:"used_count,omitempty"`

	// Remaining is the remaining available quota.
	Remaining int `json:"remain_count,omitempty"`

	// TotalRegistered is the total number of registered tracking numbers.
	TotalRegistered int `json:"registered_count,omitempty"`
}

// PushConfig represents push notification configuration.
type PushConfig struct {
	// URL is the webhook callback URL.
	URL string `json:"url,omitempty"`

	// Events is the list of event types to push.
	Events []string `json:"events,omitempty"`

	// Lang is the language for translation.
	Lang string `json:"lang,omitempty"`
}
