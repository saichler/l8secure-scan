package imagerefcve

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: Before() on POST -- common.GenerateID; customerId is copied
// from the ImageRef being processed by the caller (secscan-scanner sets
// it directly when constructing the row, §13.1) -- this row is written
// only by the scanner's own "scanner" system identity (§14), never by an
// end-user session, so it is not re-derived here.
func newImageRefCveServiceCallback(vnic ifs.IVNic) ifs.IServiceCallback {
	return l8common.NewValidation(&secscan.ImageRefCve{}, vnic).
		Require(func(e interface{}) string { return e.(*secscan.ImageRefCve).CustomerId }, "CustomerId").
		Require(func(e interface{}) string { return e.(*secscan.ImageRefCve).ImageRefId }, "ImageRefId").
		Require(func(e interface{}) string { return e.(*secscan.ImageRefCve).CveId }, "CveId").
		Enum(func(e interface{}) int32 { return int32(e.(*secscan.ImageRefCve).Severity) }, secscan.Severity_name, "Severity").
		Build()
}
