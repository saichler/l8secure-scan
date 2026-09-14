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

SecScan.loadKpis = function(customerId) {
    return Promise.all([
        SecScan.countOf('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' limit 1 page 1"),
        // ScanStatus_PENDING = 1 (bare integer, never a quoted name)
        SecScan.countOf('/60/ImageRef', "select * from ImageRef where customerId='" + customerId + "' and scanStatus=1 limit 1 page 1"),
        // Severity_CRITICAL = 4
        SecScan.countOf('/60/ImgRefCve', "select * from ImageRefCve where customerId='" + customerId + "' and severity=4 limit 1 page 1"),
        SecScan.countOf('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' and scannedRefCount=0 limit 1 page 1")
    ]).then(function(r) {
        return { totalGroups: r[0], pendingScans: r[1], criticalCves: r[2], unscannedGroups: r[3] };
    });
};
