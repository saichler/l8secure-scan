/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Image Group Detail popup (PRD §11.3): header with editable Category,
// an embedded ImageRef table (baseWhereClause-scoped, custom
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

    // ScanStatus enum order matches proto/secscan.proto exactly. Missing
    // (5) uses the same "warning" style as ScanJob's Partial -- the image
    // reference itself couldn't be resolved (bad tag/digest, deleted from
    // the registry), a different condition than Failed (scan attempted
    // against a real image but errored).
    const SCAN_STATUS = Layer8EnumFactory.create([
        ['Unspecified', null, ''],
        ['Pending', 'pending', 'layer8d-status-pending'],
        ['Scanning', 'scanning', 'layer8d-status-active'],
        ['Completed', 'completed', 'layer8d-status-active'],
        ['Failed', 'failed', 'layer8d-status-terminated'],
        ['Missing', 'missing', 'layer8d-status-warning']
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
        return '<div class="secscan-group-detail-header">' +
            '<div class="secscan-group-detail-category">' +
            '<label>Category:</label> <span id="secscan-category-picker-wrap"></span>' +
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
            ...Layer8ColumnFactory.custom('totalCounts', 'Total', function(item) { return vulnCell(item.totalCounts); }, { sortKey: false }),
            ...Layer8ColumnFactory.custom('distinctCounts', 'Distinct', function(item) { return vulnCell(item.distinctCounts); }, { sortKey: false })
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
            // showActions:true + a truthy onDelete is what makes
            // Layer8DTable render the per-row Delete button at all
            // (layer8d-table-render.js only emits it when onDelete is
            // set) -- the callback here is otherwise never reached
            // (Layer8DTable wires it as a plain bubble-phase listener on
            // the button itself, but the capturing-phase listener below
            // stops the click before it ever reaches that far; see its
            // own comment). The real delete logic lives entirely in that
            // capturing listener instead, same as the checkbox case.
            showActions: true,
            onDelete: function() {},
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
            // be reached once propagation is stopped during capture). The
            // Delete button (rendered by showActions/onDelete above) needs
            // the exact same treatment -- and since stopPropagation() here
            // during capture ALSO prevents Layer8DTable's own target-phase
            // button listener from ever firing, the actual delete call has
            // to happen directly in this handler too, not in onDelete.
            container.addEventListener('click', function(e) {
                if (e.target && e.target.classList.contains('secscan-ref-select')) {
                    e.stopPropagation();
                    const id = e.target.getAttribute('data-id');
                    const label = e.target.getAttribute('data-label');
                    SecScanImageSelection.toggle(id, label);
                    updateSelectionHint(body);
                } else if (e.target && e.target.getAttribute('data-action') === 'delete') {
                    e.stopPropagation();
                    const id = e.target.getAttribute('data-id');
                    deleteImageRef(id, body);
                }
            }, true);
        }
    }

    // ImgRefDelete (a dedicated action service, not a plain ORM DELETE on
    // ImageRef) also recomputes the parent ImageGroup's rollup cache
    // server side -- ImageRefServiceCallback.After() (RecomputeImageGroupCache)
    // only fires on PUT/PATCH, never DELETE, so a plain DELETE here would
    // leave the group's cached imageRefCount/newestCounts stale (verified
    // against l8common's genericCallback source).
    function deleteImageRef(id, body) {
        if (!confirm('Delete this image reference? This cannot be undone.')) return;
        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgRefDelete'), {
            method: 'POST',
            body: JSON.stringify({ imageRefId: id })
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Delete failed')).then(function(t) {
                    throw new Error(t || 'Delete failed');
                });
            }
            Layer8DNotification.success('Image reference deleted');
            SecScanImageSelection.remove(id);
            updateSelectionHint(body);
            // Layer8DTable has no .refresh() -- fetchData(currentPage,
            // pageSize) is the real reload call (same one
            // layer8d-module-crud.js's own _deleteItem uses).
            if (refTable) refTable.fetchData(refTable.currentPage, refTable.pageSize);
            if (typeof SecScan !== 'undefined' && SecScan.refreshCurrentTable) {
                SecScan.refreshCurrentTable();
            }
        }).catch(function(err) {
            console.error('Group Detail: failed to delete image ref', err);
            Layer8DNotification.error('Failed to delete image reference: ' + err.message);
        });
    }

    function sevCell(counts, sevKey) {
        if (!counts) return '';
        var v = counts[sevKey];
        return (v === undefined || v === null) ? '' : String(v);
    }

    // Consolidated "T:<total> C:<critical> H:<high> M:<medium> L:<low>"
    // format, replacing one column per severity (explicit user request).
    // Total and Distinct stay as two separate columns -- they're two
    // different metrics (raw vulnerability-instance count vs. unique CVE
    // count), not severities, so merging them into a single column would
    // conflate two different things rather than consolidate one.
    function vulnCell(counts) {
        var c = (counts && counts.critical) || 0;
        var h = (counts && counts.high) || 0;
        var m = (counts && counts.medium) || 0;
        var l = (counts && counts.low) || 0;
        var t = c + h + m + l;
        return 'T:' + t + ' C:' + c + ' H:' + h + ' M:' + m + ' L:' + l;
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
