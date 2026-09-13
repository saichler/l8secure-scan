package imagegroup

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: ImageGroup rows only ever emerge from ImageRef ingestion (no
// user-facing "Add Group" form) -- common.GenerateID on POST is all that's
// needed; customerId is copied from the triggering ImageRef by the
// ingestion helper (go/secscan/common), not derived here. Category
// reassignment happens via a plain PUT from the UI (§11.3).
func newImageGroupServiceCallback(vnic ifs.IVNic) ifs.IServiceCallback {
	return l8common.NewValidation(&secscan.ImageGroup{}, vnic).
		Require(func(e interface{}) string { return e.(*secscan.ImageGroup).CustomerId }, "CustomerId").
		Require(func(e interface{}) string { return e.(*secscan.ImageGroup).ImageName }, "ImageName").
		Build()
}
