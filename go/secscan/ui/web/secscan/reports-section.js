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
        // Real inline SVG (stroke="currentColor") -- matches the sidebar's
        // Reports icon (app.html), themed instead of a fixed-color emoji.
        icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/><path d="M16 13H8M16 17H8M10 9H8"/></svg>',
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
