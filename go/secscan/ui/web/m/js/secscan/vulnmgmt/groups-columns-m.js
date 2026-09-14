/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile ImageGroup columns (PRD §11.7): decorates the EXISTING desktop
// SecScanVuln.columns.ImageGroup array (../../../secscan/vulnmgmt/groups-columns.js,
// loaded earlier on this page -- same shared Layer8ColumnFactory output, no
// separate mobile column factory exists) with primary/secondary card flags,
// per AddingModule's Mobile<Namespace> convention. Duplication Prevention:
// no column logic is re-authored here.
window.MobileSecScanVuln = window.MobileSecScanVuln || {};

(function() {
    'use strict';

    var PRIMARY = { imageName: true };
    var SECONDARY = { categoryId: true };
    var HIDDEN = { imageGroupId: true };

    MobileSecScanVuln.columns = MobileSecScanVuln.columns || {};
    MobileSecScanVuln.columns.ImageGroup = SecScanVuln.columns.ImageGroup.map(function(c) {
        var mc = Object.assign({}, c);
        if (PRIMARY[mc.key]) mc.primary = true;
        else if (SECONDARY[mc.key]) mc.secondary = true;
        else if (HIDDEN[mc.key]) mc.hidden = true;
        return mc;
    });

    MobileSecScanVuln.primaryKeys = MobileSecScanVuln.primaryKeys || {};
    MobileSecScanVuln.primaryKeys.ImageGroup = SecScanVuln.primaryKeys.ImageGroup;
})();
