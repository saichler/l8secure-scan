/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Vulnerability Detail popup (PRD §11.4): summary strip of ImageRef's
// total/distinct counts, plus a read-only Layer8DTable over ImageRefCve
// (baseWhereClause-scoped, sort-by severity desc). ImageRefCve is never
// user-created/edited (§9, §20) -- showActions:false, no forms.
window.SecScanVulnDetail = (function() {
    'use strict';

    // Severity enum order matches proto/secscan.proto exactly:
    // 0=UNSPECIFIED,1=LOW,2=MEDIUM,3=HIGH,4=CRITICAL. Numeric-desc sort on
    // this field therefore produces Critical->Low for free (PRD §11.4).
    const SEVERITY = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Low', 'low', 'layer8d-status-inactive'],
        ['Medium', 'medium', 'layer8d-status-pending'],
        ['High', 'high', 'layer8d-status-warning'],
        ['Critical', 'critical', 'layer8d-status-terminated']
    ]);
    const renderSeverity = Layer8DRenderers.createStatusRenderer(SEVERITY.enum, SEVERITY.classes);

    function open(imageRefId, refItem) {
        const html = summaryHtml(refItem) + '<div id="secscan-vuln-findings-table-container"></div>';

        Layer8DPopup.show({
            title: (refItem.repoName || '') + ':' + (refItem.tag || refItem.digest || ''),
            content: html,
            size: 'xlarge',
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
        return '<div class="secscan-vuln-summary">' +
            '<table class="layer8d-table-simple">' +
            '<thead><tr><th></th><th>Critical</th><th>High</th><th>Medium</th><th>Low</th></tr></thead>' +
            '<tbody>' + row('Total', item.totalCounts) + row('Distinct', item.distinctCounts) + '</tbody>' +
            '</table></div>';
    }

    function renderFindingsTable(imageRefId) {
        const columns = [
            // Layer8ColumnFactory.link's onClick is never wired up anywhere
            // in Layer8DTable's event handling (verified -- no
            // data-action="click" handling exists in layer8d-table-events.js),
            // so a real <a href> anchor is used directly instead; no
            // onRowClick is set on this table, so there's no row-click
            // handler to conflict with.
            ...Layer8ColumnFactory.custom('cveId', 'CVE ID', function(item) {
                var id = item.cveId || '';
                if (!id) return '';
                return '<a href="https://nvd.nist.gov/vuln/detail/' + encodeURIComponent(id) +
                    '" target="_blank" rel="noopener noreferrer">' + Layer8DUtils.escapeHtml(id) + '</a>';
            }, { sortKey: 'cveId' }),
            ...Layer8ColumnFactory.col('packageName', 'Package'),
            ...Layer8ColumnFactory.col('installedVersion', 'Installed Version'),
            ...Layer8ColumnFactory.col('fixedVersion', 'Fixed Version'),
            ...Layer8ColumnFactory.status('severity', 'Severity', SEVERITY.values, renderSeverity),
            ...Layer8ColumnFactory.col('title', 'Title')
        ];

        const table = new Layer8DTable({
            containerId: 'secscan-vuln-findings-table-container',
            endpoint: Layer8DConfig.resolveEndpoint('/60/ImgRefCve'),
            modelName: 'ImageRefCve',
            columns: columns,
            primaryKey: 'imageRefCveId',
            baseWhereClause: "imageRefId='" + imageRefId + "'",
            serverSide: true,
            sortable: true,
            defaultSort: { column: 'severity', direction: 'desc' },
            showActions: false
        });
        table.init();
    }

    return { open: open };
})();
