/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Image Group Detail popup (PRD §11.3): header with editable Category +
// Trend panel, an embedded ImageRef table (baseWhereClause-scoped, custom
// checkbox multi-select -- Layer8DTable has no built-in row-selection,
// verified), and row click into Vulnerability Detail. Composes two
// independently-real, documented APIs (Layer8DPopup.show + Layer8DTable
// with baseWhereClause) that have no prior combined precedent in the
// ecosystem (verified) via Layer8DPopup's own documented onShow extension
// point.
//
// Checkbox selection here writes into the shared, cross-popup/cross-page
// SecScanImageSelection store (image-selection.js) rather than local
// module state -- the actual "scan" trigger lives on the Dashboard's Scan
// Images button now, not here, so a selection made while browsing one
// group must survive closing this popup and opening another (or none at
// all) before the user goes to Dashboard and presses Scan Images.
window.SecScanGroupDetail = (function() {
    'use strict';

    // ScanStatus enum order matches proto/secscan.proto exactly.
    const SCAN_STATUS = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Pending', 'pending', 'layer8d-status-pending'],
        ['Scanning', 'scanning', 'layer8d-status-active'],
        ['Completed', 'completed', 'layer8d-status-active'],
        ['Failed', 'failed', 'layer8d-status-terminated']
    ]);
    const renderScanStatus = Layer8DRenderers.createStatusRenderer(SCAN_STATUS.enum, SCAN_STATUS.classes);

    let currentGroupId = null;
    let refTable = null;

    function open(imageGroupId) {
        currentGroupId = imageGroupId;

        fetchGroup(imageGroupId).then(function(group) {
            if (!group) {
                Layer8DNotification.error('Image group not found');
                return;
            }
            render(group);
        }).catch(function(err) {
            console.error('Group Detail: failed to load group', err);
            Layer8DNotification.error('Failed to load image group');
        });
    }

    function fetchGroup(id) {
        const query = encodeURIComponent(JSON.stringify({ text: "select * from ImageGroup where imageGroupId='" + id + "'" }));
        return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgGroup?body=' + query))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) { return (data && data.list && data.list[0]) || null; });
    }

    function render(group) {
        const html = headerHtml(group) +
            '<div class="secscan-group-detail-toolbar">' +
            '<span id="secscan-selection-hint" class="secscan-selection-hint"></span>' +
            '</div>' +
            '<div id="secscan-group-refs-table-container"></div>';

        Layer8DPopup.show({
            title: group.imageName,
            content: html,
            size: 'xlarge',
            showFooter: false,
            onShow: function(body) {
                attachCategoryPicker(body, group);
                updateSelectionHint(body);
                renderRefTable(body, group);
            }
        });
    }

    function headerHtml(group) {
        const newest = group.newestCounts || null;
        const oldest = group.oldestCounts || null;
        const trendRows = SecScanVuln.SEVERITIES.map(function(sev) {
            const pct = SecScanVuln.reductionPct(newest, oldest, group.scannedRefCount, sev);
            const label = sev.charAt(0).toUpperCase() + sev.slice(1);
            const nv = newest ? (newest[sev] || 0) : '';
            const ov = oldest ? (oldest[sev] || 0) : '';
            return '<tr><td>' + label + '</td><td>' + nv + '</td><td>' + ov + '</td><td>' +
                SecScanVuln.formatReductionPct(pct) + '</td></tr>';
        }).join('');

        return '<div class="secscan-group-detail-header">' +
            '<div class="secscan-group-detail-category">' +
            '<label>Category:</label> <span id="secscan-category-picker-wrap"></span>' +
            '</div>' +
            '<div class="secscan-group-detail-trend">' +
            '<table class="layer8d-table-simple"><thead><tr><th>Severity</th><th>Newest</th><th>Oldest</th><th>Reduction</th></tr></thead>' +
            '<tbody>' + trendRows + '</tbody></table>' +
            (newest && oldest ? '' : '<p class="secscan-trend-empty">Trend data available once at least one scan completes.</p>') +
            '</div></div>';
    }

    function attachCategoryPicker(body, group) {
        const wrap = body.querySelector('#secscan-category-picker-wrap');
        if (!wrap) return;
        const input = document.createElement('input');
        input.type = 'text';
        input.id = 'secscan-category-picker-input';
        wrap.appendChild(input);

        Layer8DReferencePicker.attach(input, {
            endpoint: Layer8DConfig.resolveEndpoint('/60/ImgCat'),
            modelName: 'ImageCategory',
            idColumn: 'categoryId',
            displayColumn: 'name',
            onChange: function(id) {
                saveCategory(group.imageGroupId, id);
            }
        });

        const existing = SecScanVuln.getCategory(group.categoryId);
        if (existing) {
            Layer8DReferencePicker.setValue(input, group.categoryId, existing.name, existing);
        }
    }

    function saveCategory(imageGroupId, categoryId) {
        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgGroup'), {
            method: 'PUT',
            body: JSON.stringify({ imageGroupId: imageGroupId, categoryId: categoryId || '' })
        }).then(function(resp) {
            if (!resp || !resp.ok) throw new Error('Save failed');
            Layer8DNotification.success('Category updated');
            if (typeof SecScan !== 'undefined' && SecScan.refreshCurrentTable) {
                SecScan.refreshCurrentTable();
            }
        }).catch(function(err) {
            console.error('Group Detail: failed to save category', err);
            Layer8DNotification.error('Failed to update category');
        });
    }

    function renderRefTable(body, group) {
        const columns = [
            ...Layer8ColumnFactory.custom('_select', '', function(item) {
                const label = (item.repoName || '') + (item.tag ? ':' + item.tag : '');
                const checked = SecScanImageSelection.has(item.imageRefId) ? ' checked' : '';
                return '<input type="checkbox" class="secscan-ref-select" data-id="' + item.imageRefId +
                    '" data-label="' + Layer8DUtils.escapeHtml(label) + '"' + checked + '>';
            }, { sortKey: false }),
            ...Layer8ColumnFactory.col('repoName', 'Repo'),
            ...Layer8ColumnFactory.col('tag', 'Tag'),
            ...Layer8ColumnFactory.custom('buildDate', 'Build Date', function(item) {
                if (item.scanError && !item.buildDate) {
                    return '<span class="layer8d-status-terminated">Failed: ' + Layer8DUtils.escapeHtml(item.scanError) + '</span>';
                }
                return item.buildDate ? Layer8DUtils.formatDate(item.buildDate) : 'Resolving…';
            }, { sortKey: 'buildDate' }),
            ...Layer8ColumnFactory.status('scanStatus', 'Scan Status', SCAN_STATUS.values, renderScanStatus),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total C', function(item) { return sevCell(item.totalCounts, 'critical'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total H', function(item) { return sevCell(item.totalCounts, 'high'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total M', function(item) { return sevCell(item.totalCounts, 'medium'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total L', function(item) { return sevCell(item.totalCounts, 'low'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct C', function(item) { return sevCell(item.distinctCounts, 'critical'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct H', function(item) { return sevCell(item.distinctCounts, 'high'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct M', function(item) { return sevCell(item.distinctCounts, 'medium'); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct L', function(item) { return sevCell(item.distinctCounts, 'low'); }, { sortKey: false })
        ];

        refTable = new Layer8DTable({
            containerId: 'secscan-group-refs-table-container',
            endpoint: Layer8DConfig.resolveEndpoint('/60/ImageRef'),
            modelName: 'ImageRef',
            columns: columns,
            primaryKey: 'imageRefId',
            baseWhereClause: "imageGroupId='" + group.imageGroupId + "'",
            serverSide: true,
            sortable: true,
            defaultSort: { column: 'buildDate', direction: 'desc' },
            showActions: false,
            onRowClick: function(item, id) {
                if (typeof SecScanVulnDetail !== 'undefined') {
                    SecScanVulnDetail.open(id, item);
                }
            }
        });
        refTable.init();

        const container = body.querySelector('#secscan-group-refs-table-container');
        if (container) {
            // Capturing-phase, not bubbling: the row's own click listener
            // (Layer8DTable's onRowClick, opening Vulnerability Detail) is
            // attached directly on the <tr>, a descendant of this
            // container -- during the bubble phase that listener always
            // fires BEFORE an event reaches a bubble-phase listener up
            // here, so stopPropagation() there is already too late (this
            // was a real, confirmed bug: checking the box also opened
            // Vulnerability Detail on top of this popup). Capturing at the
            // container runs before the event ever reaches the row, so
            // stopping it here prevents onRowClick from firing at all; the
            // selection toggle itself has to happen in this same listener
            // (a separate bubble-phase listener on this element would never
            // be reached once propagation is stopped during capture).
            container.addEventListener('click', function(e) {
                if (e.target && e.target.classList.contains('secscan-ref-select')) {
                    e.stopPropagation();
                    const id = e.target.getAttribute('data-id');
                    const label = e.target.getAttribute('data-label');
                    SecScanImageSelection.toggle(id, label);
                    updateSelectionHint(body);
                }
            }, true);
        }
    }

    function sevCell(counts, sevKey) {
        if (!counts) return '';
        var v = counts[sevKey];
        return (v === undefined || v === null) ? '' : String(v);
    }

    // Shows the running cross-popup/cross-page selection count -- the
    // actual scan trigger is the Dashboard's Scan Images button, not
    // anything in this popup, so this is feedback only.
    function updateSelectionHint(body) {
        const el = body.querySelector('#secscan-selection-hint');
        if (!el) return;
        const n = SecScanImageSelection.count();
        el.textContent = n === 0
            ? 'Select images below, then go to Dashboard to scan them.'
            : n + ' image' + (n === 1 ? '' : 's') + ' selected for scanning (Dashboard → Scan Images).';
    }

    return { open: open };
})();
