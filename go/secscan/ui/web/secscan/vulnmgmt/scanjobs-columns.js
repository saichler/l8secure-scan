/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const col = Layer8ColumnFactory;
    const render = SecScanVuln.render;

    SecScanVuln.columns = SecScanVuln.columns || {};
    SecScanVuln.columns.ScanJob = [
        ...col.id('scanJobId'),
        ...col.status('status', 'Status', SecScanVuln.enums.JOB_STATUS_VALUES, render.jobStatus),
        ...col.number('totalImages', 'Total'),
        ...col.number('completedImages', 'Completed'),
        ...col.number('failedImages', 'Failed'),
        ...col.datetime('requestedAt', 'Requested At'),
        ...col.datetime('completedAt', 'Completed At'),
        ...col.col('requestedBy', 'Requested By')
    ];

    SecScanVuln.primaryKeys = SecScanVuln.primaryKeys || {};
    SecScanVuln.primaryKeys.ScanJob = 'scanJobId';
})();
