/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanAdmin = window.SecScanAdmin || {};

(function() {
    'use strict';
    const col = Layer8ColumnFactory;

    SecScanAdmin.columns = SecScanAdmin.columns || {};
    SecScanAdmin.columns.Customer = [
        ...col.col('name', 'Name'),
        ...col.boolean('isActive', 'Active')
    ];

    SecScanAdmin.primaryKeys = SecScanAdmin.primaryKeys || {};
    SecScanAdmin.primaryKeys.Customer = 'customerId';
})();
