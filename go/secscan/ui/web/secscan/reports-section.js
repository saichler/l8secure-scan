/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Reports: its own top-level nav section rather than a button buried in
// the Images table toolbar. The one report this app produces is the
// cross-group CSV export (PRD §10/§11.1, SecScanExportReport) -- it is a
// vulnerabilities report, not a generic "CSV export", so it's named and
// framed as that here. No module/service CRUD table needed, so this
// registers a plain customContent page (Layer8SectionGenerator supports
// that directly) instead of going through Layer8DModuleFactory.
(function() {
    'use strict';

    Layer8SectionConfigs.register('reports', {
        title: 'Reports',
        subtitle: 'Vulnerabilities Report',
        icon: '📊',
        modules: [],
        customContent:
            '<div class="secscan-reports-page">' +
            '<p>Download a CSV report of every image group\'s vulnerability counts and ' +
            'reduction trend for the current customer.</p>' +
            '<button class="layer8d-btn layer8d-btn-primary" id="secscan-vuln-report-btn">' +
            'Download Vulnerabilities Report</button>' +
            '</div>'
    });

    window.initializeSecScanReports = function() {
        const btn = document.getElementById('secscan-vuln-report-btn');
        if (btn && !btn.dataset.secscanReportsAttached) {
            btn.dataset.secscanReportsAttached = '1';
            btn.addEventListener('click', function() {
                if (typeof SecScanExportReport !== 'undefined') SecScanExportReport.run();
            });
        }
    };
})();
