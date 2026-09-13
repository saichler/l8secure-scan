package services

import (
	"github.com/saichler/l8secure-scan/go/secscan/customer"
	"github.com/saichler/l8secure-scan/go/secscan/cve"
	"github.com/saichler/l8secure-scan/go/secscan/imagecategory"
	"github.com/saichler/l8secure-scan/go/secscan/imagegroup"
	"github.com/saichler/l8secure-scan/go/secscan/imageref"
	"github.com/saichler/l8secure-scan/go/secscan/imagerefcve"
	"github.com/saichler/l8secure-scan/go/secscan/imgrefadd"
	"github.com/saichler/l8secure-scan/go/secscan/scanjob"
	"github.com/saichler/l8secure-scan/go/secscan/vulnrep"
	"github.com/saichler/l8types/go/ifs"
)

// ActivateSecscanServices activates every secscan-owned service in this
// process (the secscan backend -- SingleOwnerDatabaseTable: this is the
// one and only process where the ORM is activated for these seven Prime
// Objects). ImgRefAdd and VulnRep are stateless action handlers hosted in
// the same process; they have no table/ORM of their own (PRD §9).
func ActivateSecscanServices(creds, dbname string, vnic ifs.IVNic) {
	customer.Activate(creds, dbname, vnic)
	imagecategory.Activate(creds, dbname, vnic)
	imagegroup.Activate(creds, dbname, vnic)
	imageref.Activate(creds, dbname, vnic)
	cve.Activate(creds, dbname, vnic)
	imagerefcve.Activate(creds, dbname, vnic)
	scanjob.Activate(creds, dbname, vnic)

	imgrefadd.Activate(vnic)
	vulnrep.Activate(vnic)
}
