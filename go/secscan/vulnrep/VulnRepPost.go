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
		reductionPctCell(g.NewestCounts, g.OldestCounts),
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

	// The reference and nothing else, one per line. Still sorted newest
	// buildDate first, but the date itself is no longer printed -- the
	// cell is a list of image refs, not a changelog. escapeCSV quotes any
	// cell containing a newline, so the line breaks render inside the one
	// cell rather than breaking the row apart.
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		parts = append(parts, fmt.Sprintf("%s:%s", r.RepoName, r.Tag))
	}
	return strings.Join(parts, "\n"), nil
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

// cellWidth is the column width every count value is padded to, so the
// T/C/H/M/L fields line up down the file. Values wider than this are not
// truncated -- they just push their column out, which is preferable to
// losing a digit.
const cellWidth = 4

// reductionCellWidth is the same idea for the Reduction % column, which
// needs more room: its values carry a decimal, a "%", and sometimes a
// minus sign, so they run 4 to 7 characters ("0.0%" up to "-100.0%").
// Padding those to cellWidth would never actually pad anything and the
// column would not line up at all.
const reductionCellWidth = 7

// vulnCountCell matches the Images table's own Vulnerabilities column
// format ("T:<total> C:<critical> H:<high> M:<medium> L:<low>"), with each
// value left-aligned in cellWidth characters.
func vulnCountCell(c *secscan.VulnerabilityCounts) string {
	if c == nil {
		c = &secscan.VulnerabilityCounts{}
	}
	return fmt.Sprintf("T:%-*d C:%-*d H:%-*d M:%-*d L:%-*d",
		cellWidth, total(c), cellWidth, c.Critical, cellWidth, c.High,
		cellWidth, c.Medium, cellWidth, c.Low)
}

// reductionPctCell mirrors vulnCountCell's T/C/H/M/L shape and padding,
// but every value is a reductionPct() result -- `total` satisfies the same
// sev func(*VulnerabilityCounts) int32 shape reductionPct takes, so the T
// value is the same math applied to each side's combined severity count
// instead of one severity's.
//
// Padded to reductionCellWidth rather than cellWidth, since these values
// are wider than the counts. The "%" is appended BEFORE padding, not
// after, so the sign stays welded to its number ("0.0%   ", never
// "0.0   %"). Padding between them would read as two separate values, and
// the cell is whitespace-delimited.
func reductionPctCell(newest, oldest *secscan.VulnerabilityCounts) string {
	// Normalized once here so the per-severity getters below never see a
	// nil -- only total() guards against it on its own.
	if newest == nil {
		newest = &secscan.VulnerabilityCounts{}
	}
	if oldest == nil {
		oldest = &secscan.VulnerabilityCounts{}
	}
	pct := func(sev func(*secscan.VulnerabilityCounts) int32) string {
		return reductionPct(newest, oldest, sev) + "%"
	}
	return fmt.Sprintf("T:%-*s C:%-*s H:%-*s M:%-*s L:%-*s",
		reductionCellWidth, pct(total), reductionCellWidth, pct(critical),
		reductionCellWidth, pct(high), reductionCellWidth, pct(medium),
		reductionCellWidth, pct(low))
}

// reductionPct is (oldest-newest)/oldest as a percentage, and always
// returns a number -- never "N/A", which told the reader nothing and made
// the column impossible to sort or chart.
//
// The old scannedRefCount<2 rule is gone rather than replaced: a group with
// a single scanned ref has newest == oldest by construction
// (scommon.RecomputeImageGroupCache), so the formula already yields 0.0 --
// which is the honest answer. Nothing has changed yet, because there is
// only one measurement.
//
// oldest.sev == 0 is the one case the formula genuinely cannot express,
// the denominator being zero. Split by what it means instead of refusing:
// 0 -> 0 is no change at all, and 0 -> something is a pure regression with
// no baseline to measure it against, reported at the -100.0 floor. Both
// are negative-or-zero, which is what a reduction column should say when
// things got worse.
func reductionPct(newest, oldest *secscan.VulnerabilityCounts, sev func(*secscan.VulnerabilityCounts) int32) string {
	o := sev(oldest)
	n := sev(newest)
	if o == 0 {
		if n == 0 {
			return "0.0"
		}
		return "-100.0"
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
