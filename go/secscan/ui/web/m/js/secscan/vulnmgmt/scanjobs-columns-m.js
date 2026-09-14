/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile ScanJob columns: decorates the EXISTING desktop
// SecScanVuln.columns.ScanJob array (its status column keeps the
// desktop-authored badge renderer from SecScanVuln.render.jobStatus --
// the layer8d-status-* CSS classes it emits are part of the shared theme,
// loaded on this mobile page too, so the badge renders identically).
window.MobileSecScanVuln = window.MobileSecScanVuln || {};

(function() {
    'use strict';

    var PRIMARY = { requestedAt: true };
    var SECONDARY = { requestedBy: true };
    var HIDDEN = { scanJobId: true };

    MobileSecScanVuln.columns = MobileSecScanVuln.columns || {};
    MobileSecScanVuln.columns.ScanJob = SecScanVuln.columns.ScanJob.map(function(c) {
        var mc = Object.assign({}, c);
        if (PRIMARY[mc.key]) mc.primary = true;
        else if (SECONDARY[mc.key]) mc.secondary = true;
        else if (HIDDEN[mc.key]) mc.hidden = true;
        return mc;
    });

    MobileSecScanVuln.primaryKeys = MobileSecScanVuln.primaryKeys || {};
    MobileSecScanVuln.primaryKeys.ScanJob = SecScanVuln.primaryKeys.ScanJob;
})();
