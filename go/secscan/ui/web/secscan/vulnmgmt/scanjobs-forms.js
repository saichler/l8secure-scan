/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const f = Layer8FormFactory;

    function ro(fields) {
        return fields.map(function(field) { field.readOnly = true; return field; });
    }

    // ScanJobs are created only via "Scan Selected" in Group Detail
    // (group-detail.js), never a manual add form -- this is a view-only
    // history record.
    SecScanVuln.forms = SecScanVuln.forms || {};
    SecScanVuln.forms.ScanJob = f.form('Scan Job', [
        f.section('Job Details', [
            ...ro(f.select('status', 'Status', SecScanVuln.enums.JOB_STATUS)),
            ...ro(f.number('totalImages', 'Total Images')),
            ...ro(f.number('completedImages', 'Completed')),
            ...ro(f.number('failedImages', 'Failed')),
            ...ro(f.datetime('requestedAt', 'Requested At')),
            ...ro(f.datetime('completedAt', 'Completed At')),
            ...ro(f.text('requestedBy', 'Requested By'))
        ])
    ]);
})();
