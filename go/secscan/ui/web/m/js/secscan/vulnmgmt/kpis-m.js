/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Shared mobile KPI count-query helper (PRD §11.1/§11.7), used by both the
// home dashboard's #nav-stats strip (sections/dashboard.html) and the
// Image Groups list's own KPI strip (groups-view-m.js) -- extracted here on
// second use per Duplication Prevention. Same metadata.keyCount.counts.Total
// technique as desktop's dashboard-kpis.js.
window.SecScan = window.SecScan || {};

SecScan.countOf = function(endpoint, query) {
    const q = encodeURIComponent(JSON.stringify({ text: query }));
    return Layer8MAuth.get(Layer8MConfig.resolveEndpoint(endpoint + '?body=' + q))
        .then(function(data) {
            return (data && data.metadata && data.metadata.keyCount && data.metadata.keyCount.counts && data.metadata.keyCount.counts.Total) || 0;
        })
        .catch(function() { return 0; });
};

// ImageRefCve rows exist for EVERY scanned ImageRef ever, including ones a
// group has since moved past (a newer, patched build added later) --
// counting them directly meant a remediated image's old findings never
// stopped being counted. Each ImageGroup's own newestCounts is already
// exactly "the severity counts of this group's most-recently-built
// scanned ref" (RecomputeImageGroupCache, server side), so summing that
// across groups gives the latest-image-only, remediation-aware total this
// card needs -- the same source desktop's dashboard-page.js and the
// Images table's own Vulnerabilities column already use.
SecScan.fetchCveStats = function(customerId) {
    const q = encodeURIComponent(JSON.stringify({
        text: "select * from ImageGroup where customerId='" + customerId + "' limit 999 page 0"
    }));
    return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ImgGroup?body=' + q))
        .then(function(data) {
            const list = (data && data.list) || [];
            const stats = { critical: 0, high: 0, medium: 0, low: 0 };
            list.forEach(function(g) {
                const c = g.newestCounts;
                if (!c) return;
                stats.critical += c.critical || 0;
                stats.high += c.high || 0;
                stats.medium += c.medium || 0;
                stats.low += c.low || 0;
            });
            return stats;
        })
        .catch(function() { return { critical: 0, high: 0, medium: 0, low: 0 }; });
};

SecScan.loadKpis = function(customerId) {
    return Promise.all([
        SecScan.countOf('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' limit 1 page 0"),
        // ScanStatus_PENDING = 1 (bare integer, never a quoted name)
        SecScan.countOf('/60/ImageRef', "select * from ImageRef where customerId='" + customerId + "' and scanStatus=1 limit 1 page 0"),
        SecScan.fetchCveStats(customerId),
        SecScan.countOf('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' and scannedRefCount=0 limit 1 page 0")
    ]).then(function(r) {
        return { totalGroups: r[0], pendingScans: r[1], cveStats: r[2], unscannedGroups: r[3] };
    });
};
