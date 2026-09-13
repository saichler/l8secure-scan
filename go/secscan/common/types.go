package common

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
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
}
