/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Image Group Detail popup (PRD §11.7, parity with desktop's
// group-detail.js): header with editable Category, an
// embedded ImageRef Layer8MTable (baseWhereClause-scoped, custom checkbox
// multi-select -- Layer8MTable/Layer8MEditTable have no native
// row-selection either, verified), a "Scan Selected" toolbar action, a
// "Scan all pending images" checkbox (scoped to this group only -- mobile
// has no cross-page Dashboard selection/scan button like desktop's, so
// scanning only ever happens from inside one group's own popup here), and
// card-tap into Vulnerability Detail. Composes Layer8MPopup.show +
// Layer8MTable with baseWhereClause -- no real ecosystem precedent for this
// exact combination exists on mobile either (same gap already found on
// desktop, Phase 4), built the same well-justified way.
window.SecScanGroupDetail_M = (function() {
    'use strict';

    // ScanStatus enum order matches proto/secscan.proto exactly (same
    // Layer8EnumFactory used by desktop -- only the renderer differs).
    // Missing (5) uses the same "warning" style as desktop's -- the image
    // reference itself couldn't be resolved, a different condition than
    // Failed (scan attempted against a real image but errored).
    const SCAN_STATUS = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Pending', 'pending', 'mobile-status-pending'],
        ['Scanning', 'scanning', 'mobile-status-active'],
        ['Completed', 'completed', 'mobile-status-active'],
        ['Failed', 'failed', 'mobile-status-terminated'],
        ['Missing', 'missing', 'mobile-status-warning']
    ]);
    const renderScanStatus = Layer8MRenderers.createStatusRenderer(SCAN_STATUS.enum, SCAN_STATUS.classes);

    let selectedIds = new Set();
    let refTable = null;
    let currentGroupId = null;
    // Mirrors desktop dashboard-page.js's scanAllPendingIds: remembers
    // exactly which ids the "Scan all pending" checkbox itself added, so
    // unchecking it removes only those -- not any refs the user separately
    // tapped by hand in this same table.
    let scanAllPendingIds = null;

    function open(imageGroupId) {
        selectedIds = new Set();
        currentGroupId = imageGroupId;
        scanAllPendingIds = null;

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
            '<label class="secscan-scan-all-pending-label">' +
            '<input type="checkbox" id="secscan-m-scan-all-pending-checkbox"> Scan all pending images' +
            '</label>' +
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
                attachScanAllPending(popup.body, group);
                renderRefTable(popup.body, group);
            }
        });
    }

    function headerHtml(group) {
        return '<div class="secscan-m-group-header">' +
            '<div class="secscan-m-category-row" id="secscan-m-category-row">' +
            '<span class="secscan-m-category-label">Category:</span> ' +
            '<span id="secscan-m-category-value">' + Layer8MUtils.escapeHtml(SecScanVuln.getCategoryName(group.categoryId)) + '</span>' +
            '<span class="secscan-m-category-tap-hint">(tap to change)</span>' +
            '</div></div>';
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

    // Same "T:<total> C:<critical> H:<high> M:<medium> L:<low>" format as
    // desktop's group-detail.js (explicit user request) -- Total and
    // Distinct stay separate columns since they're different metrics (raw
    // vulnerability-instance count vs. unique CVE count), not severities.
    function vulnCell(counts) {
        var c = (counts && counts.critical) || 0;
        var h = (counts && counts.high) || 0;
        var m = (counts && counts.medium) || 0;
        var l = (counts && counts.low) || 0;
        var t = c + h + m + l;
        return 'T:' + t + ' C:' + c + ' H:' + h + ' M:' + m + ' L:' + l;
    }

    function updateScanButton(body) {
        const btn = body.querySelector('#secscan-m-scan-selected-btn');
        if (btn) btn.disabled = selectedIds.size === 0;
    }

    // Scoped to THIS group only (desktop's Dashboard equivalent scans
    // across every group for the customer -- mobile has no cross-page
    // selection/Dashboard scan button at all, scanning only ever happens
    // from inside a specific group's own popup here, so "all pending"
    // means "all pending in this group").
    function fetchPendingImageRefs(groupId) {
        const query = "select * from ImageRef where imageGroupId='" + groupId + "' and scanStatus=1 limit 999 page 0";
        return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ImageRef?body=' + encodeURIComponent(JSON.stringify({ text: query }))))
            .then(function(data) { return (data && data.list) || []; });
    }

    function attachScanAllPending(body, group) {
        const checkbox = body.querySelector('#secscan-m-scan-all-pending-checkbox');
        if (!checkbox) return;
        checkbox.addEventListener('change', function() {
            if (checkbox.checked) {
                checkbox.disabled = true;
                fetchPendingImageRefs(group.imageGroupId).then(function(refs) {
                    scanAllPendingIds = refs.map(function(r) { return r.imageRefId; });
                    refs.forEach(function(r) { selectedIds.add(r.imageRefId); });
                    if (refs.length === 0) {
                        Layer8MUtils.showSuccess('No pending images to scan');
                    }
                    updateScanButton(body);
                    if (refTable) refTable.refresh();
                }).catch(function(err) {
                    console.error('Scan All Pending (mobile): failed to load pending images', err);
                    Layer8MUtils.showError('Failed to load pending images');
                    checkbox.checked = false;
                }).finally(function() {
                    checkbox.disabled = false;
                });
            } else if (scanAllPendingIds) {
                scanAllPendingIds.forEach(function(id) { selectedIds.delete(id); });
                scanAllPendingIds = null;
                updateScanButton(body);
                if (refTable) refTable.refresh();
            }
        });
    }

    function renderRefTable(body, group) {
        const columns = [
            Object.assign({}, Layer8ColumnFactory.custom('repoName', 'Repo', function(item) {
                const checked = selectedIds.has(item.imageRefId) ? ' checked' : '';
                const label = (item.repoName || '') + (item.tag ? ':' + item.tag : '');
                return '<label class="secscan-m-ref-select-label" onclick="event.stopPropagation()">' +
                    '<input type="checkbox" class="secscan-m-ref-select" data-id="' + item.imageRefId + '"' + checked + '> ' +
                    Layer8MUtils.escapeHtml(item.repoName) + ':' + Layer8MUtils.escapeHtml(item.tag || '') +
                    '</label>' +
                    '<button type="button" class="secscan-m-ref-delete-btn" data-action="delete-ref" ' +
                    'data-id="' + item.imageRefId + '" data-label="' + Layer8MUtils.escapeHtml(label) + '" ' +
                    'onclick="event.stopPropagation()">Delete</button>';
            })[0], { primary: true }),
            Object.assign({}, Layer8ColumnFactory.custom('buildDate', 'Build Date', function(item) {
                if (item.scanError && !item.buildDate) {
                    return 'Failed: ' + Layer8MUtils.escapeHtml(item.scanError);
                }
                return item.buildDate ? Layer8MUtils.formatDate(item.buildDate) : 'Resolving…';
            }, { sortKey: 'buildDate' })[0], { secondary: true }),
            Object.assign({}, Layer8ColumnFactory.status('scanStatus', 'Scan Status', SCAN_STATUS.values, renderScanStatus)[0], { secondary: true }),
            ...Layer8ColumnFactory.custom('totalCounts', 'Total', function(item) { return vulnCell(item.totalCounts); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct', function(item) { return vulnCell(item.distinctCounts); }, { sortKey: false })
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
                } else if (e.target && e.target.getAttribute('data-action') === 'delete-ref') {
                    e.stopPropagation();
                    const id = e.target.getAttribute('data-id');
                    const label = e.target.getAttribute('data-label');
                    deleteImageRef(id, label, body);
                }
            });
        }
    }

    // ImgRefDelete (a dedicated action service, not a plain ORM DELETE on
    // ImageRef) also recomputes the parent ImageGroup's rollup cache
    // server side -- ImageRefServiceCallback.After() (RecomputeImageGroupCache)
    // only fires on PUT/PATCH, never DELETE, so a plain DELETE here would
    // leave the group's cached imageRefCount/newestCounts stale (verified
    // against l8common's genericCallback source, same fix as desktop's
    // group-detail.js).
    function deleteImageRef(id, label, body) {
        if (!confirm('Delete "' + label + '"? This cannot be undone.')) return;
        Layer8MAuth.post(Layer8MConfig.resolveEndpoint('/60/ImgRefDelete'), { imageRefId: id })
            .then(function(resp) {
                if (!resp) throw new Error('Delete failed');
                Layer8MUtils.showSuccess('Image reference deleted');
                selectedIds.delete(id);
                updateScanButton(body);
                if (refTable) refTable.refresh();
            }).catch(function(err) {
                console.error('Group Detail (mobile): failed to delete image ref', err);
                Layer8MUtils.showError('Failed to delete image reference: ' + err.message);
            });
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

    // --- Scan Failure Report (shown once a job with failedImages > 0 is done) ----
    // Same approach as desktop's dashboard-page.js: ScanJob only carries an
    // aggregate failedImages count, so the report is built by re-fetching
    // the actual ImageRefs (scanStatus FAILED/MISSING + scanError, already
    // persisted per-image by scanloop.go), scoped to this group and
    // filtered to the ids this specific job scanned. Two plain equality
    // queries (scanStatus=4, scanStatus=5), not one OR/IN query -- this
    // project's L8QL only supports simple AND-chained equality (verified
    // elsewhere: a LIKE query was rejected outright).
    function fetchGroupStatusRefs(groupId, status) {
        const query = "select * from ImageRef where imageGroupId='" + groupId + "' and scanStatus=" + status + " limit 999 page 0";
        return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ImageRef?body=' + encodeURIComponent(JSON.stringify({ text: query }))))
            .then(function(data) { return (data && data.list) || []; });
    }

    function showScanFailureReport(groupId, jobRefIds) {
        Promise.all([fetchGroupStatusRefs(groupId, 4), fetchGroupStatusRefs(groupId, 5)])
            .then(function(results) {
                const SCAN_STATUS_MISSING = 5;
                const refs = results[0].concat(results[1]).filter(function(r) { return jobRefIds.has(r.imageRefId); });
                if (refs.length === 0) return;
                const items = refs.map(function(r) {
                    const label = (r.repoName || '') + (r.tag ? ':' + r.tag : '');
                    const statusText = r.scanStatus === SCAN_STATUS_MISSING ? 'Missing' : 'Failed';
                    return '<div class="secscan-m-scan-failure-item">' +
                        '<div class="secscan-m-scan-failure-title">' + Layer8MUtils.escapeHtml(label) + ' &mdash; ' + statusText + '</div>' +
                        '<div class="secscan-m-scan-failure-reason">' + Layer8MUtils.escapeHtml(r.scanError || '(no reason recorded)') + '</div>' +
                        '</div>';
                }).join('');
                Layer8MPopup.show({
                    title: 'Scan Failures (' + refs.length + ')',
                    content: items,
                    size: 'large',
                    showFooter: false
                });
            }).catch(function(err) {
                console.error('Scan Failure Report (mobile): failed to load details', err);
            });
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
                .then(function(data) {
                    // POST /60/ScanJob's response body is the same generic
                    // {list, metadata} wrapper every query response uses
                    // (verified live, matching desktop's dashboard-page.js
                    // comment on the same endpoint) -- Layer8MAuth.post()
                    // returns response.json() completely unwrapped, so this
                    // was checking a bare-object shape the response never
                    // actually has. Real, pre-existing bug: mobile's Scan
                    // Selected always fell into the catch block below with
                    // "No scan job returned" and never actually started a
                    // scan, caught only by running this against a live
                    // server end to end.
                    const job = data && data.list && data.list[0];
                    if (!job || !job.scanJobId) {
                        throw new Error('No scan job returned');
                    }
                    // .showInfo, not .showSuccess -- this fires the instant
                    // the ScanJob is CREATED, before a single image has
                    // actually been scanned (real user confusion, reported
                    // live, on the desktop equivalent of this same message).
                    Layer8MUtils.showInfo('Started scanning ' + ids.length + ' image(s)');
                    selectedIds = new Set();
                    scanAllPendingIds = null;
                    const scanAllCheckbox = body.querySelector('#secscan-m-scan-all-pending-checkbox');
                    if (scanAllCheckbox) scanAllCheckbox.checked = false;
                    updateScanButton(body);
                    const wrap = body.querySelector('#secscan-m-scan-progress');
                    if (wrap && typeof Layer8DProgressBar !== 'undefined') {
                        Layer8DProgressBar.attach(wrap, {
                            modelType: 'ScanJob', // protobuf type name, not ServiceName
                            primaryKey: job.scanJobId,
                            fetchCurrent: function() { return fetchScanJob(job.scanJobId); },
                            getProgress: scanJobProgress,
                            onDone: function(finishedJob) {
                                if (refTable) refTable.refresh();
                                setTimeout(function() { wrap.hidden = true; }, 3000);
                                if (finishedJob && finishedJob.failedImages > 0) {
                                    showScanFailureReport(group.imageGroupId, new Set(finishedJob.imageRefIds || []));
                                }
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
