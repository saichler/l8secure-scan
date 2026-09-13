/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const factory = window.Layer8EnumFactory;
    const { createStatusRenderer } = Layer8DRenderers;

    // Order matches proto/secscan.proto's JobStatus exactly (index = wire
    // value): 0=UNSPECIFIED,1=QUEUED,2=RUNNING,3=COMPLETED,4=FAILED,5=PARTIAL.
    const JOB_STATUS = factory.create([
        ['Unspecified', null, ''],
        ['Queued', 'queued', 'layer8d-status-pending'],
        ['Running', 'running', 'layer8d-status-active'],
        ['Completed', 'completed', 'layer8d-status-active'],
        ['Failed', 'failed', 'layer8d-status-terminated'],
        ['Partial', 'partial', 'layer8d-status-warning']
    ]);

    SecScanVuln.enums = SecScanVuln.enums || {};
    SecScanVuln.enums.JOB_STATUS = JOB_STATUS.enum;
    SecScanVuln.enums.JOB_STATUS_VALUES = JOB_STATUS.values;

    SecScanVuln.render = SecScanVuln.render || {};
    SecScanVuln.render.jobStatus = createStatusRenderer(JOB_STATUS.enum, JOB_STATUS.classes);
})();
