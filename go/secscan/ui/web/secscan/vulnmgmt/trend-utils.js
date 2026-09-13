/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Canonical Reduction % "N/A" rule (PRD §10), shared by the dashboard's
// Trend indicator column (groups-columns.js) and the Group Detail Trend
// panel (group-detail.js) -- extracted on second use per Duplication
// Prevention. N/A when scannedRefCount<2 (covers both "fewer than 2
// scanned refs" and "newest/oldest resolve to the same ref" -- by
// construction in the Go backend's cache-maintenance hook those always
// coincide, PROGRESS.md) or when oldest.sev=0 (division by zero).
window.SecScanVuln = window.SecScanVuln || {};

SecScanVuln.SEVERITIES = ['critical', 'high', 'medium', 'low'];

// Returns a number (percentage) or null for N/A.
SecScanVuln.reductionPct = function(newestCounts, oldestCounts, scannedRefCount, sevKey) {
    if (!scannedRefCount || scannedRefCount < 2 || !newestCounts || !oldestCounts) {
        return null;
    }
    var o = oldestCounts[sevKey] || 0;
    var n = newestCounts[sevKey] || 0;
    if (o === 0) {
        return null;
    }
    return ((o - n) / o) * 100;
};

SecScanVuln.formatReductionPct = function(pct) {
    if (pct === null || pct === undefined) {
        return 'N/A';
    }
    var rounded = Math.round(pct * 10) / 10;
    if (rounded > 0) {
        return '▼' + rounded + '%'; // reduction, down arrow
    }
    if (rounded < 0) {
        return '▲' + Math.abs(rounded) + '%'; // increase, up arrow
    }
    return 'flat';
};
