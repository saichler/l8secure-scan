package common

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// RegisterSecscanTypes registers every secscan Prime Object with the
// introspector (primary-key decorator) and type registry. Called by both
// the backend main.go (this phase -- the ORM needs it) and, in Phase 4,
// go/secscan/ui/main.go (reuse this function rather than redefining the
// same calls there, per Duplication Prevention).
func RegisterSecscanTypes(resources ifs.IResources) {
	l8common.RegisterType(resources, &secscan.Customer{}, &secscan.CustomerList{}, "CustomerId")
	l8common.RegisterType(resources, &secscan.ImageCategory{}, &secscan.ImageCategoryList{}, "CategoryId")
	l8common.RegisterType(resources, &secscan.ImageGroup{}, &secscan.ImageGroupList{}, "ImageGroupId")
	l8common.RegisterType(resources, &secscan.ImageRef{}, &secscan.ImageRefList{}, "ImageRefId")
	l8common.RegisterType(resources, &secscan.Cve{}, &secscan.CveList{}, "CveId")
	l8common.RegisterType(resources, &secscan.ImageRefCve{}, &secscan.ImageRefCveList{}, "ImageRefCveId")
	l8common.RegisterType(resources, &secscan.ScanJob{}, &secscan.ScanJobList{}, "ScanJobId")

	resources.Registry().Register(&secscan.ImgRefAddRequest{})
	resources.Registry().Register(&secscan.ImgRefAddResponse{})
	resources.Registry().Register(&secscan.VulnRepRequest{})
	// VulnRep's response reuses the generic l8api.L8CsvExportResponse
	// (VulnRep.go) rather than a custom secscan type -- unlike
	// ImgRefAddResponse, nothing else in this project ever registered it,
	// so the web-frontend process failed to deserialize VulnRep's endpoint
	// definition ("Unknown Type: L8CsvExportResponse"), silently leaving
	// its Endpoints map empty ("endpoint not found for action VulnRep
	// area 60 1" on every POST).
	resources.Registry().Register(&l8api.L8CsvExportResponse{})
	// Same class of bug hit again, live: without these two, secscan-web
	// couldn't deserialize ImgRefDel's endpoint definition either
	// ("endpoint not found for action ImgRefDel area 60 1" on every
	// POST) -- every custom action-service request/response type needs
	// registering here too, not just on the backend that actually
	// Activate()s the service.
	resources.Registry().Register(&secscan.ImgRefDeleteRequest{})
	resources.Registry().Register(&secscan.ImgRefDeleteResponse{})
}
