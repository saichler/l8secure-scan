package cve

import (
	"errors"

	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// PRD §9: no common.GenerateID -- cve_id (e.g. "CVE-2023-1234") is a
// natural key supplied by the scanner's find-or-create logic (§13.1), not
// generated. Not customer-scoped (no scope-cve deny rule, §14). Uses the
// raw builder (not NewValidation) specifically to avoid its automatic
// UUID-based setID, which would be wrong for a natural-key entity.
func newCveServiceCallback() ifs.IServiceCallback {
	return l8common.NewServiceCallback(
		"Cve",
		func(v interface{}) bool { _, ok := v.(*secscan.Cve); return ok },
		func(v interface{}) {},
		validateCve,
	)
}

func validateCve(e interface{}, vnic ifs.IVNic) error {
	c, ok := e.(*secscan.Cve)
	if !ok {
		return errors.New("invalid Cve type")
	}
	if c.CveId == "" {
		return errors.New("CveId is required")
	}
	return l8common.ValidateEnum(int32(c.Severity), secscan.Severity_name, "Severity")
}
