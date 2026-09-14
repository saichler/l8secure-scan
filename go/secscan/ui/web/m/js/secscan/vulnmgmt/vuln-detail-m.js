/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Vulnerability Detail popup (PRD §11.7, parity with desktop's
// vuln-detail.js): summary strip of ImageRef's total/distinct counts, plus
// a read-only Layer8MTable over ImageRefCve (baseWhereClause-scoped,
// sort-by severity desc). ImageRefCve is never user-created/edited.
window.SecScanVulnDetail_M = (function() {
    'use strict';

    // Severity enum order matches proto/secscan.proto exactly:
    // 0=UNSPECIFIED,1=LOW,2=MEDIUM,3=HIGH,4=CRITICAL.
    const SEVERITY = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Low', 'low', 'mobile-status-inactive'],
        ['Medium', 'medium', 'mobile-status-pending'],
        ['High', 'high', 'mobile-status-warning'],
        ['Critical', 'critical', 'mobile-status-terminated']
    ]);
    const renderSeverity = Layer8MRenderers.createStatusRenderer(SEVERITY.enum, SEVERITY.classes);

    function open(imageRefId, refItem) {
        const html = summaryHtml(refItem) + '<div id="secscan-m-vuln-findings-container"></div>';

        Layer8MPopup.show({
            title: (refItem.repoName || '') + ':' + (refItem.tag || refItem.digest || ''),
            content: html,
            size: 'large',
            showFooter: false,
            onShow: function() {
                renderFindingsTable(imageRefId);
            }
        });
    }

    function summaryHtml(item) {
        function row(label, counts) {
            if (!counts) return '<tr><td>' + label + '</td><td colspan="4">Not yet scanned</td></tr>';
            return '<tr><td>' + label + '</td><td>' + (counts.critical || 0) + '</td><td>' +
                (counts.high || 0) + '</td><td>' + (counts.medium || 0) + '</td><td>' + (counts.low || 0) + '</td></tr>';
        }
        return '<div class="secscan-m-vuln-summary">' +
            '<table class="secscan-m-trend-table">' +
            '<thead><tr><th></th><th>Critical</th><th>High</th><th>Medium</th><th>Low</th></tr></thead>' +
            '<tbody>' + row('Total', item.totalCounts) + row('Distinct', item.distinctCounts) + '</tbody>' +
            '</table></div>';
    }

    function renderFindingsTable(imageRefId) {
        const columns = [
            Object.assign({}, Layer8ColumnFactory.col('cveId', 'CVE ID')[0], { primary: true }),
            Object.assign({}, Layer8ColumnFactory.status('severity', 'Severity', SEVERITY.values, renderSeverity)[0], { secondary: true }),
            ...Layer8ColumnFactory.col('packageName', 'Package'),
            ...Layer8ColumnFactory.col('installedVersion', 'Installed Version'),
            ...Layer8ColumnFactory.col('fixedVersion', 'Fixed Version'),
            ...Layer8ColumnFactory.col('title', 'Title')
        ];

        new Layer8MTable('secscan-m-vuln-findings-container', {
            endpoint: Layer8MConfig.resolveEndpoint('/60/ImgRefCve'),
            modelName: 'ImageRefCve',
            columns: columns,
            primaryKey: 'imageRefCveId',
            baseWhereClause: "imageRefId='" + imageRefId + "'",
            sortable: true,
            defaultSort: { column: 'severity', direction: 'desc' }
        });
    }

    return { open: open };
})();
