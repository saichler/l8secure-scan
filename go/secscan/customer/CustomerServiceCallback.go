package customer

import (
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// Customer rows are created only by opsadmin (not customer-scoped, PRD §9).
func newCustomerServiceCallback(vnic ifs.IVNic) ifs.IServiceCallback {
	return l8common.NewValidation(&secscan.Customer{}, vnic).
		Require(func(e interface{}) string { return e.(*secscan.Customer).Name }, "Name").
		Build()
}
