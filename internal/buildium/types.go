package buildium

// Types are intentionally minimal.
// Buildium may return many more fields; Go will ignore unknown JSON fields.

// Lease is a minimal shape of Buildium's lease item returned from /leases?expand=tenants,unit
type Lease struct {
	ID          int    `json:"Id"`
	LeaseToDate string `json:"LeaseToDate"` //  Python treats it as "YYYY-MM-DD" :contentReference
	UnitNumber  string `json:"UnitNumber"`

	Tenants []LeaseTenant `json:"Tenants"`
	Unit    *Unit         `json:"Unit"`
}

// LeaseTenant is the compact tenant reference inside an expanded Lease object.
type LeaseTenant struct {
	ID     int    `json:"Id"`
	Status string `json:"Status"`
}

// Unit is expanded inside the lease when using expand=unit.
type Unit struct {
	Address *Address `json:"Address"`
}

// Address is used in both tenant details and unit details.
type Address struct {
	AddressLine1 string `json:"AddressLine1"`
}

// OutstandingBalance is returned by /leases/outstandingbalances
type OutstandingBalance struct {
	LeaseID      int     `json:"LeaseId"`
	TotalBalance float64 `json:"TotalBalance"`
}

// TenantDetails comes from /leases/tenants/{tenantId}
type TenantDetails struct {
	ID           int           `json:"Id"`
	FirstName    string        `json:"FirstName"`
	LastName     string        `json:"LastName"`
	Email        string        `json:"Email"`
	Address      *Address      `json:"Address"`
	PhoneNumbers []PhoneNumber `json:"PhoneNumbers"`
}

type PhoneNumber struct {
	Number string `json:"Number"`
}
