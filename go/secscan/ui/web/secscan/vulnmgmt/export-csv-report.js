/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// "Export CSV Report" (PRD §10/§11.1): POSTs to the bespoke VulnRep action
// service (consolidated 6-column cross-group report -- Newest/Oldest/
// Reduction % each a single "T:x C:x H:x M:x L:x" cell, matching the
// Images table's own Vulnerabilities column format), distinct from the generic
// per-row Layer8CsvExport button that auto-attaches to the Image Groups
// table's own pagination bar. Blob-download mechanics mirror
// l8ui/shared/layer8-csv-export.js's real _download implementation (the
// only real precedent for turning a csvData response field into a
// downloaded file in this framework).
window.SecScanExportReport = (function() {
    'use strict';

    function run() {
        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8DNotification.error('No customer context found for this session');
            return;
        }

        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/VulnRep'), {
            method: 'POST',
            body: JSON.stringify({ customerId: customerId })
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Export failed')).then(function(t) {
                    throw new Error(t || 'Export failed');
                });
            }
            return resp.json();
        }).then(function(data) {
            if (!data || !data.csvData) {
                throw new Error('No CSV data returned');
            }
            download(data.csvData, data.filename || 'VulnRep.csv');
            Layer8DNotification.success('Report exported (' + (data.rowCount || 0) + ' rows)');
        }).catch(function(err) {
            console.error('Export CSV Report error:', err);
            Layer8DNotification.error('Failed to export report: ' + err.message);
        });
    }

    function download(csvData, filename) {
        const blob = new Blob([csvData], { type: 'text/csv;charset=utf-8;' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    }

    return { run: run };
})();
