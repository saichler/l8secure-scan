package imagerefcve

import (
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

func Activate(creds, dbname string, vnic ifs.IVNic) {
	l8common.ActivateService(l8common.ServiceConfig{
		ServiceName: scommon.ImageRefCveServiceName, ServiceArea: scommon.ServiceArea,
		PrimaryKey: "ImageRefCveId", Callback: newImageRefCveServiceCallback(vnic),
	}, &secscan.ImageRefCve{}, &secscan.ImageRefCveList{}, creds, dbname, vnic)
}
