/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Image Group Detail popup (PRD §11.7, parity with desktop's
// group-detail.js): header with editable Category + Trend panel, an
// embedded ImageRef Layer8MTable (baseWhereClause-scoped, custom checkbox
// multi-select -- Layer8MTable/Layer8MEditTable have no native
// row-selection either, verified), a "Scan Selected" toolbar action, and
// card-tap into Vulnerability Detail. Composes Layer8MPopup.show +
// Layer8MTable with baseWhereClause -- no real ecosystem precedent for this
// exact combination exists on mobile either (same gap already found on
// desktop, Phase 4), built the same well-justified way.
window.SecScanGroupDetail_M = (function() {
    'use strict';

    // ScanStatus enum order matches proto/secscan.proto exactly (same
    // Layer8EnumFactory used by desktop -- only the renderer differs).
    const SCAN_STATUS = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Pending', 'pending', 'mobile-status-pending'],
        ['Scanning', 'scanning', 'mobile-status-active'],
        ['Completed', 'completed', 'mobile-status-active'],
        ['Failed', 'failed', 'mobile-status-terminated']
    ]);
    const renderScanStatus = Layer8MRenderers.createStatusRenderer(SCAN_STATUS.enum, SCAN_STATUS.classes);

    let selectedIds = new Set();
    let refTable = null;

    function open(imageGroupId) {
        selectedIds = new Set();

        fetchGroup(imageGroupId).then(function(group) {
            if (!group) {
                Layer8MUtils.showError('Image group not found');
                return;
            }
            render(group);
        }).catch(function(err) {
            console.error('Group Detail (mobile): failed to load group', err);
            Layer8MUtils.showError('Failed to load image group');
        });
    }

    function fetchGroup(id) {
        const query = "select * from ImageGroup where imageGroupId='" + id + "'";
        return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ImgGroup?body=' + encodeURIComponent(JSON.stringify({ text: query }))))
            .then(function(data) { return (data && data.list && data.list[0]) || null; });
    }

    function render(group) {
        const html = headerHtml(group) +
            '<div class="secscan-m-group-toolbar">' +
            '<button class="mobile-popup-btn mobile-popup-btn-save" id="secscan-m-scan-selected-btn" disabled>Scan Selected</button>' +
            '</div>' +
            // Empty on purpose -- Layer8DProgressBar.attach() populates this
            // container with its own generic markup (plans/scanjob-live-progress.md
            // Phase 5-m).
            '<div id="secscan-m-scan-progress" class="secscan-m-scan-progress" hidden></div>' +
            '<div id="secscan-m-group-refs-container"></div>';

        Layer8MPopup.show({
            title: group.imageName,
            content: html,
            size: 'large',
            showFooter: false,
            onShow: function(popup) {
                attachCategoryPicker(popup.body, group);
                attachScanSelected(popup.body, group);
                renderRefTable(popup.body, group);
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

        return '<div class="secscan-m-group-header">' +
            '<div class="secscan-m-category-row" id="secscan-m-category-row">' +
            '<span class="secscan-m-category-label">Category:</span> ' +
            '<span id="secscan-m-category-value">' + Layer8MUtils.escapeHtml(SecScanVuln.getCategoryName(group.categoryId)) + '</span>' +
            '<span class="secscan-m-category-tap-hint">(tap to change)</span>' +
            '</div>' +
            '<table class="secscan-m-trend-table"><thead><tr><th>Severity</th><th>Newest</th><th>Oldest</th><th>Reduction</th></tr></thead>' +
            '<tbody>' + trendRows + '</tbody></table>' +
            (newest && oldest ? '' : '<p class="secscan-m-trend-empty">Trend data available once at least one scan completes.</p>') +
            '</div>';
    }

    function attachCategoryPicker(body, group) {
        const row = body.querySelector('#secscan-m-category-row');
        if (!row) return;
        row.addEventListener('click', function() {
            Layer8MReferencePicker.show({
                endpoint: Layer8MConfig.resolveEndpoint('/60/ImgCat'),
                modelName: 'ImageCategory',
                idColumn: 'categoryId',
                displayColumn: 'name',
                title: 'Select Category',
                currentValue: group.categoryId || null,
                onChange: function(id) {
                    saveCategory(group, id);
                }
            });
        });
    }

    function saveCategory(group, categoryId) {
        Layer8MAuth.put(Layer8MConfig.resolveEndpoint('/60/ImgGroup'), { imageGroupId: group.imageGroupId, categoryId: categoryId || '' })
            .then(function() {
                group.categoryId = categoryId || '';
                const valueEl = document.getElementById('secscan-m-category-value');
                if (valueEl) valueEl.textContent = SecScanVuln.getCategoryName(categoryId);
                Layer8MUtils.showSuccess('Category updated');
            }).catch(function(err) {
                console.error('Group Detail (mobile): failed to save category', err);
                Layer8MUtils.showError('Failed to update category: ' + err.message);
            });
    }

    function sevCell(counts, sevKey) {
        if (!counts) return '';
        var v = counts[sevKey];
        return (v === undefined || v === null) ? '' : String(v);
    }

    function updateScanButton(body) {
        const btn = body.querySelector('#secscan-m-scan-selected-btn');
        if (btn) btn.disabled = selectedIds.size === 0;
    }

    function renderRefTable(body, group) {
        const columns = [
            Object.assign({}, Layer8ColumnFactory.custom('repoName', 'Repo', function(item) {
                return '<label class="secscan-m-ref-select-label" onclick="event.stopPropagation()">' +
                    '<input type="checkbox" class="secscan-m-ref-select" data-id="' + item.imageRefId + '"> ' +
                    Layer8MUtils.escapeHtml(item.repoName) + ':' + Layer8MUtils.escapeHtml(item.tag || '') +
                    '</label>';
            })[0], { primary: true }),
            Object.assign({}, Layer8ColumnFactory.custom('buildDate', 'Build Date', function(item) {
                if (item.scanError && !item.buildDate) {
                    return 'Failed: ' + Layer8MUtils.escapeHtml(item.scanError);
                }
                return item.buildDate ? Layer8MUtils.formatDate(item.buildDate) : 'Resolving…';
            }, { sortKey: 'buildDate' })[0], { secondary: true }),
            Object.assign({}, Layer8ColumnFactory.status('scanStatus', 'Scan Status', SCAN_STATUS.values, renderScanStatus)[0], { secondary: true }),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total (C/H/M/L)', function(item) {
                return sevCell(item.totalCounts, 'critical') + '/' + sevCell(item.totalCounts, 'high') + '/' +
                    sevCell(item.totalCounts, 'medium') + '/' + sevCell(item.totalCounts, 'low');
            }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct (C/H/M/L)', function(item) {
                return sevCell(item.distinctCounts, 'critical') + '/' + sevCell(item.distinctCounts, 'high') + '/' +
                    sevCell(item.distinctCounts, 'medium') + '/' + sevCell(item.distinctCounts, 'low');
            }, { sortKey: false })
        ];

        refTable = new Layer8MTable('secscan-m-group-refs-container', {
            endpoint: Layer8MConfig.resolveEndpoint('/60/ImageRef'),
            modelName: 'ImageRef',
            columns: columns,
            primaryKey: 'imageRefId',
            baseWhereClause: "imageGroupId='" + group.imageGroupId + "'",
            sortable: true,
            defaultSort: { column: 'buildDate', direction: 'desc' },
            onCardClick: function(item) {
                if (typeof SecScanVulnDetail_M !== 'undefined') {
                    SecScanVulnDetail_M.open(item.imageRefId, item);
                }
            }
        });

        const container = body.querySelector('#secscan-m-group-refs-container');
        if (container) {
            container.addEventListener('click', function(e) {
                if (e.target && e.target.classList.contains('secscan-m-ref-select')) {
                    e.stopPropagation();
                    const id = e.target.getAttribute('data-id');
                    if (e.target.checked) {
                        selectedIds.add(id);
                    } else {
                        selectedIds.delete(id);
                    }
                    updateScanButton(body);
                }
            });
        }
    }

    // JobStatus enum (proto/secscan.proto): 1=QUEUED (never set anymore --
    // no poll/claim step left to queue behind) 2=RUNNING 3=COMPLETED
    // 4=FAILED 5=PARTIAL.
    function scanJobStatusLabel(status) {
        switch (status) {
            case 1: return 'Queued';
            case 2: return 'Scanning';
            case 3: return 'Completed';
            case 4: return 'Failed';
            case 5: return 'Partially completed';
            default: return 'Scanning';
        }
    }

    function scanJobProgress(job) {
        const total = job.totalImages || 1;
        const done = (job.completedImages || 0) + (job.failedImages || 0);
        const pct = Math.min(100, Math.round((done / total) * 100));
        return {
            percent: pct,
            label: scanJobStatusLabel(job.status) + ': ' + done + ' / ' + total + ' image(s)' +
                (job.failedImages ? ' (' + job.failedImages + ' failed)' : ''),
            done: job.status === 3 || job.status === 4 || job.status === 5
        };
    }

    // ScanJobs is the renamed, ORM-backed persistence service -- the
    // stateless ScanJob action service's own Get() is stubbed "not
    // supported", so fetching a job's current status must target
    // /60/ScanJobs, not /60/ScanJob (the POST-only endpoint below still
    // uses). "register" in the query text registers this session's live
    // subscription server-side
    // (l8utils/plans/generic-websocket-change-notifications.md).
    function fetchScanJob(scanJobId) {
        const query = "select * from ScanJob where scanJobId='" + scanJobId + "' register";
        return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ScanJobs?body=' + encodeURIComponent(JSON.stringify({ text: query }))))
            .then(function(data) { return (data && data.list && data.list[0]) || null; });
    }

    function attachScanSelected(body, group) {
        const btn = body.querySelector('#secscan-m-scan-selected-btn');
        if (!btn) return;
        btn.addEventListener('click', function() {
            if (selectedIds.size === 0) return;
            const customerId = SecScan.getCurrentCustomerId();
            if (!customerId) {
                Layer8MUtils.showError('No customer context found for this session');
                return;
            }
            const ids = Array.from(selectedIds);
            Layer8MAuth.post(Layer8MConfig.resolveEndpoint('/60/ScanJob'), { customerId: customerId, imageRefIds: ids })
                .then(function(job) {
                    if (!job || !job.scanJobId) {
                        throw new Error('No scan job returned');
                    }
                    Layer8MUtils.showSuccess('Scanning ' + ids.length + ' image(s)');
                    selectedIds = new Set();
                    updateScanButton(body);
                    const wrap = body.querySelector('#secscan-m-scan-progress');
                    if (wrap && typeof Layer8DProgressBar !== 'undefined') {
                        Layer8DProgressBar.attach(wrap, {
                            modelType: 'ScanJob', // protobuf type name, not ServiceName
                            primaryKey: job.scanJobId,
                            fetchCurrent: function() { return fetchScanJob(job.scanJobId); },
                            getProgress: scanJobProgress,
                            onDone: function() {
                                if (refTable) refTable.refresh();
                                setTimeout(function() { wrap.hidden = true; }, 3000);
                            }
                        });
                    }
                }).catch(function(err) {
                    console.error('Scan Selected (mobile) error:', err);
                    Layer8MUtils.showError('Failed to start scan: ' + err.message);
                });
        });
    }

    return { open: open };
})();
