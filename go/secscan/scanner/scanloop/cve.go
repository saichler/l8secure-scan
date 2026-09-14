package scanloop

import (
	"fmt"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// findOrCreateCve is the Cve-catalog equivalent of the ImageGroup
// find-or-create in scommon.PrepareImageRef (PRD §13.1 step 2) -- same
// "look up by natural key, create if absent" shape, a separate concrete
// function rather than a generic one (NoGoGenerics).
func findOrCreateCve(cveId string, severity secscan.Severity, title string, vnic ifs.IVNic) error {
	query := fmt.Sprintf("select * from Cve where cveId='%s'", cveId)
	existing, err := l8common.GetEntitiesByQuery(scommon.CveServiceName, scommon.ServiceArea, query, vnic)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	cve := &secscan.Cve{CveId: cveId, Severity: severity, Title: title}
	_, err = l8common.PostEntity(scommon.CveServiceName, scommon.ServiceArea, cve, vnic)
	return err
}

// replaceFindings deletes every existing ImageRefCve row for this
// imageRefId (PRD §13.1 step 2 -- re-scan cleanup, see plans/PROGRESS.md
// for why: without it, re-scanning an ImageRef would accumulate stale
// findings from every previous scan forever) and writes the new scan's
// findings. Only called after Trivy has already succeeded and its output
// parsed, so a failed scan attempt never wipes a previous successful
// scan's findings.
func replaceFindings(ref *secscan.ImageRef, report *TrivyReport, vnic ifs.IVNic) error {
	deleteQuery := fmt.Sprintf("select * from ImageRefCve where imageRefId='%s'", ref.ImageRefId)
	if err := scommon.DeleteEntitiesByQuery(scommon.ImageRefCveServiceName, scommon.ServiceArea, deleteQuery, vnic); err != nil {
		return err
	}

	for _, res := range report.Results {
		for _, v := range res.Vulnerabilities {
			sev := mapSeverity(v.Severity)
			if sev == secscan.Severity_SEVERITY_UNSPECIFIED {
				continue
			}
			if err := findOrCreateCve(v.VulnerabilityID, sev, v.Title, vnic); err != nil {
				return err
			}

			finding := &secscan.ImageRefCve{
				CustomerId:       ref.CustomerId,
				ImageRefId:       ref.ImageRefId,
				CveId:            v.VulnerabilityID,
				Severity:         sev,
				PackageName:      v.PkgName,
				InstalledVersion: v.InstalledVersion,
				FixedVersion:     v.FixedVersion,
				Title:            v.Title,
			}
			if _, err := l8common.PostEntity(scommon.ImageRefCveServiceName, scommon.ServiceArea, finding, vnic); err != nil {
				return err
			}
		}
	}
	return nil
}
