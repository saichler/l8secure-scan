/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile ImageCategory columns: decorates the EXISTING desktop
// SecScanVuln.columns.ImageCategory array.
window.MobileSecScanVuln = window.MobileSecScanVuln || {};

(function() {
    'use strict';

    var PRIMARY = { name: true };
    var HIDDEN = { categoryId: true };

    MobileSecScanVuln.columns = MobileSecScanVuln.columns || {};
    MobileSecScanVuln.columns.ImageCategory = SecScanVuln.columns.ImageCategory.map(function(c) {
        var mc = Object.assign({}, c);
        if (PRIMARY[mc.key]) mc.primary = true;
        else if (HIDDEN[mc.key]) mc.hidden = true;
        else mc.secondary = true;
        return mc;
    });

    MobileSecScanVuln.primaryKeys = MobileSecScanVuln.primaryKeys || {};
    MobileSecScanVuln.primaryKeys.ImageCategory = SecScanVuln.primaryKeys.ImageCategory;
})();
