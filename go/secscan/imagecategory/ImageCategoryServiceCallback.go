package imagecategory

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: Before() on POST takes customer_id from the request body
// (client-trusted, §4) and auto-generates the primary key.
func newImageCategoryServiceCallback(vnic ifs.IVNic) ifs.IServiceCallback {
	return l8common.NewValidation(&secscan.ImageCategory{}, vnic).
		Require(func(e interface{}) string { return e.(*secscan.ImageCategory).CustomerId }, "CustomerId").
		Require(func(e interface{}) string { return e.(*secscan.ImageCategory).Name }, "Name").
		Build()
}
