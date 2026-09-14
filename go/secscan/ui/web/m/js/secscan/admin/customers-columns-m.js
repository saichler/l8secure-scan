/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Customer columns: decorates the EXISTING desktop
// SecScanAdmin.columns.Customer array.
window.MobileSecScanAdmin = window.MobileSecScanAdmin || {};

(function() {
    'use strict';

    var PRIMARY = { name: true };
    var HIDDEN = { customerId: true };

    MobileSecScanAdmin.columns = MobileSecScanAdmin.columns || {};
    MobileSecScanAdmin.columns.Customer = SecScanAdmin.columns.Customer.map(function(c) {
        var mc = Object.assign({}, c);
        if (PRIMARY[mc.key]) mc.primary = true;
        else if (HIDDEN[mc.key]) mc.hidden = true;
        else mc.secondary = true;
        return mc;
    });

    MobileSecScanAdmin.primaryKeys = MobileSecScanAdmin.primaryKeys || {};
    MobileSecScanAdmin.primaryKeys.Customer = SecScanAdmin.primaryKeys.Customer;
})();
