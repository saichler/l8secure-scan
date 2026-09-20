package imageref

import (
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

func Activate(creds, dbname string, vnic ifs.IVNic) {
	sla := l8common.NewOrmSLA(scommon.ImageRefServiceName, scommon.ServiceArea, "ImageRefId", newImageRefServiceCallback(),
		&secscan.ImageRef{}, &secscan.ImageRefList{})
	l8common.ActivateService(sla, creds, dbname, vnic)
}
