package common

// ServiceArea is shared by every secscan-owned service (Maintainability:
// "ServiceArea same for all services in a module").
const ServiceArea = byte(60)

// ServiceName constants (PRD §9). Centralized here, not redeclared per
// package, since the ingestion helpers below need to reference ImageRef
// and ImageGroup by name without importing those packages (would cycle,
// since both import this package).
const (
	CustomerServiceName      = "Customer"
	ImageCategoryServiceName = "ImgCat"
	ImageGroupServiceName    = "ImgGroup"
	ImageRefServiceName      = "ImageRef"
	CveServiceName           = "Cve"
	ImageRefCveServiceName   = "ImgRefCve"
	// ScanJobsServiceName is the ORM/Postgres-backed persistence service --
	// pure CRUD, no ServiceCallback, nothing polls it (plans/scanjob-live-progress.md).
	ScanJobsServiceName = "ScanJobs"
	// ScanJobServiceName is the stateless action service hosted inside
	// secscan-scanner: the Dashboard/mobile "Scan Selected" POST target.
	// It writes to ScanJobsServiceName as scanning progresses
	// (plans/scanjob-live-progress.md).
	ScanJobServiceName   = "ScanJob"
	ImgRefAddServiceName = "ImgRefAdd"
	VulnRepServiceName   = "VulnRep"
	// Must be under 10 characters (SLA service-name limit, a real panic
	// caught only by actually running this: "SLA Service name
	// ImgRefDelete must be less than 10 characters long").
	ImgRefDeleteServiceName = "ImgRefDel"
)

// PREFIX is the project's REST API prefix (LoginJsonAdaptation).
const PREFIX = "/scan/"

// DB_CREDS/DB_NAME identify the postgres credential entry in the security
// config plugin (../l8secure/go/secure/plugin/secscan/secscan.json):
// credentials[DB_CREDS].creds[DB_NAME].
var DB_CREDS = "postgres"
var DB_NAME = "secscan"
