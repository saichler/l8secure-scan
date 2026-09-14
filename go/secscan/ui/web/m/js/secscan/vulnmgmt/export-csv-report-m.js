/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile "Export CSV Report" (PRD §10/§11.7): POSTs to the bespoke VulnRep
// action service. Blob-download mechanics are the EXACT SAME desktop-style
// Blob+<a download> trick, verbatim -- confirmed no mobile-specific
// download handling exists anywhere in the ecosystem (no Web Share API
// precedent either).
window.SecScanExportReport_M = (function() {
    'use strict';

    function run() {
        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8MUtils.showError('No customer context found for this session');
            return;
        }

        Layer8MAuth.post(Layer8MConfig.resolveEndpoint('/60/VulnRep'), { customerId: customerId })
            .then(function(data) {
                if (!data || !data.csvData) {
                    throw new Error('No CSV data returned');
                }
                download(data.csvData, data.filename || 'VulnRep.csv');
                Layer8MUtils.showSuccess('Report exported (' + (data.rowCount || 0) + ' rows)');
            }).catch(function(err) {
                console.error('Export CSV Report (mobile) error:', err);
                Layer8MUtils.showError('Failed to export report: ' + err.message);
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
