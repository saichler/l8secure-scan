package common

import "fmt"

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
// MaxAutoScanBytes is the size above which an image is not scanned as part
// of a multi-image job. Scanning is serialized on one Trivy CLI and one
// shared cache (scanloop's trivyMu), so a single multi-gigabyte pull
// stalls every other image queued behind it -- which is exactly what a
// bulk "scan all pending" sweep is. Such an image is still scannable, just
// on its own, where it blocks nothing.
//
// 1 GiB, not 1 GB: sizes here come from the registry as raw byte counts
// and every tool that displays them (docker, crane, GCR's own console)
// divides by 1024.
const MaxAutoScanBytes int64 = 1 << 30

// HumanBytes renders a byte count the way every container tool does --
// binary units, one decimal. Used in operator-facing messages, never for
// arithmetic.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

var DB_CREDS = "postgres"
var DB_NAME = "secscan"
