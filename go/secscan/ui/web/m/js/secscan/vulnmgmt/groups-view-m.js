/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Image Groups list (PRD §11.7, parity with desktop's
// dashboard-kpis.js + the generic 'groups' table): the 'groups' service in
// the nav-config uses customInit (the real Layer8MNav mechanism for a
// non-generic service view, verified in layer8m-nav.js) instead of the
// default Layer8MNavData pipeline, because that pipeline has no extension
// point for extra chrome (KPI strip + Add Images/Export CSV Report
// toolbar) above the table -- mirrors desktop's own hand-composition in
// dashboard-kpis.js/group-detail.js for the exact same reason.
window.SecScanGroupsView_M = (function() {
    'use strict';

    const CONTAINER_ID = 'secscan-m-groups-view-container';

    function statCard(value, label, valueClass) {
        return '<div class="nav-stat-card"><div class="nav-stat-content">' +
            '<div class="nav-stat-value' + (valueClass ? ' ' + valueClass : '') + '">' + Layer8MUtils.escapeHtml(String(value)) + '</div>' +
            '<div class="nav-stat-label">' + Layer8MUtils.escapeHtml(label) + '</div>' +
            '</div></div>';
    }

    function renderStrip(kpis) {
        const cve = kpis.cveStats || { critical: 0, high: 0, medium: 0, low: 0 };
        // Per-severity breakdown IS the card's value, not a combined sum --
        // same "C:x H:x M:x L:x" format used everywhere else in this app
        // (Images table's Vulnerabilities column, CSV report).
        const cveValue = 'C:' + cve.critical + ' H:' + cve.high + ' M:' + cve.medium + ' L:' + cve.low;
        return '<div class="nav-stats-grid secscan-m-kpi-strip">' +
            statCard(kpis.totalGroups, 'Image Groups') +
            statCard(kpis.pendingScans, 'Pending Scans') +
            statCard(cveValue, 'CVEs (Latest)', 'secscan-m-kpi-cve-value') +
            statCard(kpis.unscannedGroups, 'Not Yet Scanned') +
            '</div>';
    }

    function renderToolbar() {
        return '<div class="secscan-m-groups-toolbar">' +
            '<button class="mobile-popup-btn mobile-popup-btn-save" id="secscan-m-add-images-btn">Add Images</button>' +
            '<button class="mobile-popup-btn" id="secscan-m-export-report-btn">Export CSV Report</button>' +
            '</div>';
    }

    function renderTable() {
        const columns = MobileSecScanVuln.columns.ImageGroup;
        // 'groups' uses customInit (this file), which bypasses the generic
        // layer8m-nav-data.js pipeline entirely -- so it needs its own
        // customer-scoping call, same as categories/scanjobs get via
        // LAYER8M_NAV_CONFIG's baseWhereClause (layer8m-nav-config-secscan.js).
        new Layer8MTable('secscan-m-groups-table', {
            endpoint: Layer8MConfig.resolveEndpoint('/60/ImgGroup'),
            modelName: 'ImageGroup',
            columns: columns,
            primaryKey: MobileSecScanVuln.primaryKeys.ImageGroup,
            baseWhereClause: LAYER8M_NAV_CONFIG.customerScoped(),
            sortable: true,
            onCardClick: function(item) {
                SecScanGroupDetail_M.open(item.imageGroupId);
            }
        });
    }

    function initialize() {
        const container = document.getElementById(CONTAINER_ID);
        if (!container) return;

        container.innerHTML =
            '<div class="secscan-m-kpi-strip secscan-m-kpi-loading">Loading…</div>' +
            renderToolbar() +
            '<div id="secscan-m-groups-table"></div>';

        container.querySelector('#secscan-m-add-images-btn').addEventListener('click', function() {
            SecScanAddImages_M.open();
        });
        container.querySelector('#secscan-m-export-report-btn').addEventListener('click', function() {
            SecScanExportReport_M.run();
        });

        SecScanVuln.loadCategoryCache();
        renderTable();

        const customerId = SecScan.getCurrentCustomerId();
        const stripEl = container.querySelector('.secscan-m-kpi-strip');
        if (!customerId) {
            if (stripEl) stripEl.textContent = 'No customer context found for this session.';
            return;
        }
        SecScan.loadKpis(customerId).then(function(kpis) {
            if (stripEl) stripEl.outerHTML = renderStrip(kpis);
        }).catch(function(err) {
            console.error('Dashboard KPIs (mobile): failed to load', err);
        });
    }

    return { initialize: initialize };
})();
