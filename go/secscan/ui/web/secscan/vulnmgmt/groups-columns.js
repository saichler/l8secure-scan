/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const col = Layer8ColumnFactory;

    function sevCell(counts, sevKey) {
        if (!counts) return '';
        var v = counts[sevKey];
        return (v === undefined || v === null) ? '' : String(v);
    }

    SecScanVuln.columns = SecScanVuln.columns || {};
    SecScanVuln.columns.ImageGroup = [
        ...col.id('imageGroupId'),
        ...col.col('imageName', 'Name'),
        ...col.custom('categoryId', 'Category', function(item) {
            var cat = SecScanVuln.getCategory(item.categoryId);
            if (!cat) {
                return '<span class="layer8d-tag">Uncategorized</span>';
            }
            return '<span class="layer8d-tag" style="background:' + Layer8DUtils.escapeHtml(cat.colorCode || '#888') +
                '22;border:1px solid ' + Layer8DUtils.escapeHtml(cat.colorCode || '#888') + ';">' +
                Layer8DUtils.escapeHtml(cat.name) + '</span>';
        }, { sortKey: false }),
        ...col.number('imageRefCount', 'Image Refs'),
        ...col.custom('latestBuildDate', 'Newest Build Date', function(item) {
            return item.latestBuildDate ? Layer8DUtils.formatDate(item.latestBuildDate) : 'Resolving…';
        }, { sortKey: 'latestBuildDate' }),
        ...col.custom('newestCounts', 'Newest Critical', function(item) { return sevCell(item.newestCounts, 'critical'); }, { sortKey: false }),
        ...col.custom('newestCounts', 'Newest High', function(item) { return sevCell(item.newestCounts, 'high'); }, { sortKey: false }),
        ...col.custom('newestCounts', 'Newest Medium', function(item) { return sevCell(item.newestCounts, 'medium'); }, { sortKey: false }),
        ...col.custom('newestCounts', 'Newest Low', function(item) { return sevCell(item.newestCounts, 'low'); }, { sortKey: false }),
        ...col.custom('trend', 'Trend', function(item) {
            var parts = SecScanVuln.SEVERITIES.map(function(sev) {
                var pct = SecScanVuln.reductionPct(item.newestCounts, item.oldestCounts, item.scannedRefCount, sev);
                return sev.charAt(0).toUpperCase() + ': ' + SecScanVuln.formatReductionPct(pct);
            });
            var overall = SecScanVuln.reductionPct(item.newestCounts, item.oldestCounts, item.scannedRefCount, 'critical');
            return '<span title="' + Layer8DUtils.escapeHtml(parts.join(', ')) + '">' +
                SecScanVuln.formatReductionPct(overall) + '</span>';
        }, { sortKey: false })
    ];

    SecScanVuln.primaryKeys = SecScanVuln.primaryKeys || {};
    SecScanVuln.primaryKeys.ImageGroup = 'imageGroupId';
})();
