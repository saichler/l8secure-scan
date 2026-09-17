package vulnrep

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Consolidated: one column per severity-count group ("T:<total> C:<critical>
// H:<high> M:<medium> L:<low>", matching the Images table's own
// Vulnerabilities column format exactly) instead of one column per
// severity -- explicit request, applied uniformly to every such group in
// this report, not just Newest.
var headers = []string{
	"Name", "Category", "Newest", "Oldest", "Reduction %", "Image Refs",
}

// Post generates the consolidated 6-column cross-group CSV report (PRD §10), one row
// per ImageGroup matching the request's customerId -- see the package doc
// in VulnRep.go for why customerId must be explicit rather than relying on
// automatic row-scoping.
func (this *VulnRep) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*secscan.VulnRepRequest)
	if !ok {
		return object.NewError("invalid VulnRep request type")
	}
	if req.CustomerId == "" {
		return object.NewError("customerId is required")
	}
	if err := scommon.ValidateNoQuote(req.CustomerId, "customerId"); err != nil {
		return object.NewError(err.Error())
	}

	query := fmt.Sprintf("select * from ImageGroup where customerId='%s'", req.CustomerId)
	groups, err := l8common.GetEntitiesByQuery(scommon.ImageGroupServiceName, scommon.ServiceArea, query, vnic)
	if err != nil {
		return object.NewError("fetch error: " + err.Error())
	}

	rows := make([][]string, 0, len(groups))
	for _, e := range groups {
		g, ok := e.(*secscan.ImageGroup)
		if !ok || g == nil {
			continue
		}
		row, err := buildRow(g, vnic)
		if err != nil {
			return object.NewError("row error for group " + g.ImageGroupId + ": " + err.Error())
		}
		rows = append(rows, row)
	}

	csvData := buildCSV(headers, rows)
	filename := fmt.Sprintf("VulnRep_%s.csv", time.Now().Format("2006-01-02"))

	resp := &l8api.L8CsvExportResponse{
		CsvData:  csvData,
		Filename: filename,
		RowCount: int32(len(rows)),
	}
	return object.New(nil, resp)
}

func buildRow(g *secscan.ImageGroup, vnic ifs.IVNic) ([]string, error) {
	category, err := categoryName(g.CategoryId, vnic)
	if err != nil {
		return nil, err
	}
	refsCell, err := imageRefsCell(g.ImageGroupId, vnic)
	if err != nil {
		return nil, err
	}

	return []string{
		g.ImageName,
		category,
		vulnCountCell(g.NewestCounts),
		vulnCountCell(g.OldestCounts),
		reductionPctCell(g.NewestCounts, g.OldestCounts, g.ScannedRefCount),
		refsCell,
	}, nil
}

func categoryName(categoryId string, vnic ifs.IVNic) (string, error) {
	if categoryId == "" {
		return "Uncategorized", nil
	}
	result, err := l8common.GetEntity(scommon.ImageCategoryServiceName, scommon.ServiceArea, &secscan.ImageCategory{CategoryId: categoryId}, vnic)
	if err != nil {
		return "", err
	}
	cat, ok := result.(*secscan.ImageCategory)
	if !ok || cat == nil {
		return "Uncategorized", nil
	}
	return cat.Name, nil
}

func imageRefsCell(groupId string, vnic ifs.IVNic) (string, error) {
	query := fmt.Sprintf("select * from ImageRef where imageGroupId='%s'", groupId)
	all, err := l8common.GetEntitiesByQuery(scommon.ImageRefServiceName, scommon.ServiceArea, query, vnic)
	if err != nil {
		return "", err
	}
	refs := make([]*secscan.ImageRef, 0, len(all))
	for _, e := range all {
		if r, ok := e.(*secscan.ImageRef); ok && r != nil {
			refs = append(refs, r)
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].BuildDate > refs[j].BuildDate })

	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		dateStr := "Resolving"
		if r.BuildDate > 0 {
			dateStr = time.Unix(r.BuildDate, 0).UTC().Format(time.RFC3339)
		}
		parts = append(parts, fmt.Sprintf("%s:%s (%s)", r.RepoName, r.Tag, dateStr))
	}
	return strings.Join(parts, "; "), nil
}

func critical(c *secscan.VulnerabilityCounts) int32 { return c.Critical }
func high(c *secscan.VulnerabilityCounts) int32     { return c.High }
func medium(c *secscan.VulnerabilityCounts) int32   { return c.Medium }
func low(c *secscan.VulnerabilityCounts) int32      { return c.Low }
func total(c *secscan.VulnerabilityCounts) int32 {
	if c == nil {
		return 0
	}
	return c.Critical + c.High + c.Medium + c.Low
}

// vulnCountCell matches the Images table's own Vulnerabilities column
// format exactly ("T:<total> C:<critical> H:<high> M:<medium> L:<low>").
func vulnCountCell(c *secscan.VulnerabilityCounts) string {
	if c == nil {
		c = &secscan.VulnerabilityCounts{}
	}
	return fmt.Sprintf("T:%d C:%d H:%d M:%d L:%d", total(c), c.Critical, c.High, c.Medium, c.Low)
}

// reductionPctCell mirrors vulnCountCell's T/C/H/M/L shape, but every
// value is a reductionPct() result (a percentage string, or "N/A") --
// `total` satisfies the same sev func(*VulnerabilityCounts) int32 shape
// reductionPct already takes, so the T value is just the same math applied
// to each side's combined severity count instead of one severity's.
func reductionPctCell(newest, oldest *secscan.VulnerabilityCounts, scannedRefCount int32) string {
	return fmt.Sprintf("T:%s C:%s H:%s M:%s L:%s",
		reductionPct(newest, oldest, scannedRefCount, total),
		reductionPct(newest, oldest, scannedRefCount, critical),
		reductionPct(newest, oldest, scannedRefCount, high),
		reductionPct(newest, oldest, scannedRefCount, medium),
		reductionPct(newest, oldest, scannedRefCount, low))
}

// reductionPct applies the canonical N/A rule set (PRD §10): N/A when the
// group has fewer than 2 scanned image refs, when the newest and oldest
// resolve to the same image ref (single data point -- by construction in
// scommon.RecomputeImageGroupCache, this always coincides with
// scannedRefCount<2, so no separate check is needed), or when
// oldest.sev=0 (division by zero).
func reductionPct(newest, oldest *secscan.VulnerabilityCounts, scannedRefCount int32, sev func(*secscan.VulnerabilityCounts) int32) string {
	if scannedRefCount < 2 || newest == nil || oldest == nil {
		return "N/A"
	}
	o := sev(oldest)
	n := sev(newest)
	if o == 0 {
		return "N/A"
	}
	pct := float64(o-n) / float64(o) * 100
	return strconv.FormatFloat(pct, 'f', 1, 64)
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func buildCSV(headers []string, rows [][]string) string {
	var sb strings.Builder
	for i, h := range headers {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(escapeCSV(h))
	}
	sb.WriteByte('\n')

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(escapeCSV(cell))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
