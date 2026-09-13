/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const col = Layer8ColumnFactory;

    SecScanVuln.columns = SecScanVuln.columns || {};
    SecScanVuln.columns.ImageCategory = [
        ...col.id('categoryId'),
        ...col.col('name', 'Name'),
        ...col.custom('colorCode', 'Color', function(item) {
            var color = item.colorCode || '#cccccc';
            return '<span style="display:inline-block;width:14px;height:14px;border-radius:3px;' +
                'background:' + Layer8DUtils.escapeHtml(color) + ';border:1px solid var(--layer8d-border);' +
                'vertical-align:middle;margin-right:6px;"></span>' + Layer8DUtils.escapeHtml(color);
        })
    ];

    SecScanVuln.primaryKeys = SecScanVuln.primaryKeys || {};
    SecScanVuln.primaryKeys.ImageCategory = 'categoryId';
})();
