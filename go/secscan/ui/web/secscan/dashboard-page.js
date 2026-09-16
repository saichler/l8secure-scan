/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Vulnerabilities Dashboard: its own top-level nav section (first in the
// sidebar) rather than a KPI strip/toolbar injected above the Images
// table. Holds the KPI cards, the Add Images action, and the Scan Images
// action -- Scan Images fires against whatever image refs are currently
// selected via SecScanImageSelection (checkboxes in Group Detail's ref
// table, image-selection.js), which is why this needs its own page: a
// selection made while browsing Images has to survive navigating here.
window.SecScanDashboardKpis = (function() {
    'use strict';

    function countOfEndpoint(endpoint, query) {
        const q = encodeURIComponent(JSON.stringify({ text: query }));
        return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint(endpoint + '?body=' + q))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) {
                return (data && data.metadata && data.metadata.keyCount && data.metadata.keyCount.counts && data.metadata.keyCount.counts.Total) || 0;
            })
            .catch(function() { return 0; });
    }

    function loadKpis(customerId) {
        return Promise.all([
            countOfEndpoint('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' limit 1 page 0"),
            // ScanStatus_PENDING = 1 (bare integer, never a quoted name -- verified L8Query rule)
            countOfEndpoint('/60/ImageRef', "select * from ImageRef where customerId='" + customerId + "' and scanStatus=1 limit 1 page 0"),
            // Severity_CRITICAL = 4
            countOfEndpoint('/60/ImgRefCve', "select * from ImageRefCve where customerId='" + customerId + "' and severity=4 limit 1 page 0"),
            countOfEndpoint('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' and scannedRefCount=0 limit 1 page 0")
        ]).then(function(results) {
            return {
                totalGroups: results[0],
                pendingScans: results[1],
                criticalCves: results[2],
                unscannedGroups: results[3]
            };
        });
    }

    // Layer8DWidget.render() only ever renders kpi.iconSvg verbatim -- the
    // kpi.icon key ('icon-image' etc, kept for its own CSS class hook) was
    // never looked up anywhere, so these 4 cards rendered with no icon at
    // all (confirmed: Layer8DWidget.renderEnhancedStatsGrid is the only
    // place that resolves an icon key, via a caller-supplied iconMap, and
    // nothing here ever called it). stroke="currentColor" so each icon
    // follows .layer8d-widget-icon's themed `color`.
    const KPI_ICONS = {
        'icon-image': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>',
        'icon-clock': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>',
        'icon-alert': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
        'icon-question': '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>'
    };

    function renderStrip(kpis) {
        const cards = [
            Layer8DWidget.render({ label: 'Images', icon: 'icon-image', iconSvg: KPI_ICONS['icon-image'] }, kpis.totalGroups, {}),
            Layer8DWidget.render({ label: 'Pending Scans', icon: 'icon-clock', iconSvg: KPI_ICONS['icon-clock'] }, kpis.pendingScans, {}),
            Layer8DWidget.render({ label: 'Critical CVEs', icon: 'icon-alert', iconSvg: KPI_ICONS['icon-alert'] }, kpis.criticalCves, {}),
            Layer8DWidget.render({ label: 'Groups Not Yet Scanned', icon: 'icon-question', iconSvg: KPI_ICONS['icon-question'] }, kpis.unscannedGroups, {})
        ];
        return '<div class="secscan-kpi-strip">' + cards.join('') + '</div>';
    }

    function loadStrip() {
        const stripEl = document.getElementById('secscan-dashboard-kpi-strip');
        if (!stripEl) return;
        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            stripEl.textContent = 'No customer context found for this session.';
            return;
        }
        stripEl.innerHTML = '<div class="secscan-kpi-strip secscan-kpi-loading">Loading…</div>';
        loadKpis(customerId).then(function(kpis) {
            stripEl.innerHTML = renderStrip(kpis);
        }).catch(function(err) {
            console.error('Dashboard KPIs: failed to load', err);
        });
        loadTopVulnChart(customerId);
    }

    // --- Top Images with Vulnerabilities chart -----------------------------
    // Not the same chart as Images' own table/chart view-switcher
    // (secscan-config.js, categoryField:'imageName' valueField:'newestCounts.
    // critical') -- that one only ever plots critical counts. This one ranks
    // by TOTAL vulnerability count across all four severities, a derived
    // field Layer8DChart has no direct field path for, so it's computed
    // client-side per group before handing static data to setData()
    // (bypassing the dataSource/pagination machinery entirely, same as the
    // KPI cards above do for their own counts).
    let topVulnChart = null;

    function loadTopVulnChart(customerId) {
        const chartEl = document.getElementById('secscan-top-vuln-chart');
        if (!chartEl) return;
        const q = encodeURIComponent(JSON.stringify({ text: "select * from ImageGroup where customerId='" + customerId + "' limit 100 page 0" }));
        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgGroup?body=' + q))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) {
                const list = (data && data.list) || [];
                const withTotals = list.map(function(g) {
                    const c = g.newestCounts || {};
                    return Object.assign({}, g, {
                        totalVulnCount: (c.critical || 0) + (c.high || 0) + (c.medium || 0) + (c.low || 0)
                    });
                });
                withTotals.sort(function(a, b) { return b.totalVulnCount - a.totalVulnCount; });
                const top = withTotals.slice(0, 10);

                // Re-entering this section (Layer8SectionGenerator) tears
                // down and recreates #secscan-top-vuln-chart's DOM node each
                // time -- a cached topVulnChart instance's own .container
                // reference would still point at the OLD, now-detached
                // node, so it would keep rendering invisibly into a element
                // no longer on the page (confirmed real regression: chart
                // vanished on Dashboard -> Images -> back to Dashboard).
                // destroy() releases its resize observer/tooltip before a
                // fresh instance is bound to the current, real DOM node.
                if (topVulnChart) {
                    topVulnChart.destroy();
                    topVulnChart = null;
                }
                topVulnChart = new Layer8DChart({
                    containerId: 'secscan-top-vuln-chart',
                    viewConfig: {
                        chartType: 'bar',
                        categoryField: 'imageName',
                        valueField: 'totalVulnCount',
                        aggregation: 'sum',
                        title: 'Top Images with Vulnerabilities',
                        // Full-width container would otherwise hit the
                        // default formula's 400px ceiling (layer8d-chart-
                        // core.js's height is normally width-derived) --
                        // fixed here (svg height, title/controls/padding
                        // add ~110px on top) so the whole dashboard fits
                        // the viewport without scrolling (measured).
                        // 290 still overflowed below the fold on shorter
                        // viewports -- reduced 20% (290 * 0.8 = 232).
                        height: 232
                    }
                });
                topVulnChart.init();
                topVulnChart.setData(top);
            }).catch(function(err) {
                console.error('Top vulnerabilities chart: failed to load', err);
            });
    }

    function updateScanButton(count) {
        const btn = document.getElementById('secscan-scan-images-btn');
        if (!btn) return;
        // Disabled while a scan is actively in progress too, not just
        // when nothing is selected (activeJobId set below) -- avoids
        // firing a second overlapping scan job from the same selection.
        btn.disabled = count === 0 || activeJobId !== null;
        if (activeJobId === null) {
            btn.textContent = count === 0 ? 'Scan Images' : 'Scan Images (' + count + ')';
        }
    }

    // --- Scan progress (live, via Layer8DProgressBar) ---------------------
    // activeJobId is module-level state, not DOM-scoped -- a scan started
    // from the Dashboard must still finish and clear the selection even if
    // the user navigates to Images while it's running; re-entering the
    // Dashboard re-attaches the progress bar for whatever job is still
    // active (see initializeSecScanDashboard below).

    let activeJobId = null;
    let progressBarHandle = null;

    // ScanJobs is the renamed, ORM-backed persistence service
    // (plans/scanjob-live-progress.md Phase 2) -- the stateless ScanJob
    // action service's own Get() is stubbed "not supported", so fetching a
    // job's current status must target /60/ScanJobs, not /60/ScanJob (the
    // POST-only endpoint scanSelected() below still uses). "register" in
    // the query text is what registers this session's live subscription
    // server-side (l8utils/plans/generic-websocket-change-notifications.md);
    // the protobuf type name in the query itself stays "ScanJob" either way.
    function fetchScanJob(scanJobId) {
        const q = encodeURIComponent(JSON.stringify({ text: "select * from ScanJob where scanJobId='" + scanJobId + "' register" }));
        return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ScanJobs?body=' + q))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) { return (data && data.list && data.list[0]) || null; });
    }

    // JobStatus enum (proto/secscan.proto): 1=QUEUED (never set anymore --
    // no poll/claim step left to queue behind) 2=RUNNING 3=COMPLETED
    // 4=FAILED 5=PARTIAL.
    function statusLabel(status) {
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
            label: statusLabel(job.status) + ': ' + done + ' / ' + total + ' image(s)' +
                (job.failedImages ? ' (' + job.failedImages + ' failed)' : ''),
            done: job.status === 3 || job.status === 4 || job.status === 5
        };
    }

    function attachProgressBar(scanJobId) {
        activeJobId = scanJobId;
        const wrap = document.getElementById('secscan-scan-progress');
        if (!wrap || typeof Layer8DProgressBar === 'undefined') return;
        if (progressBarHandle) progressBarHandle.detach();
        progressBarHandle = Layer8DProgressBar.attach(wrap, {
            modelType: 'ScanJob', // protobuf type name, not ServiceName (Decision 1)
            primaryKey: scanJobId,
            fetchCurrent: function() { return fetchScanJob(scanJobId); },
            getProgress: scanJobProgress,
            onDone: function() {
                activeJobId = null;
                progressBarHandle = null;
                SecScanImageSelection.clear();
                updateScanButton(SecScanImageSelection.count());
                loadStrip();
                setTimeout(function() { wrap.hidden = true; }, 3000);
            }
        });
    }

    function scanSelected() {
        const ids = SecScanImageSelection.getIds();
        if (ids.length === 0 || activeJobId !== null) return;
        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8DNotification.error('No customer context found for this session');
            return;
        }
        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ScanJob'), {
            method: 'POST',
            body: JSON.stringify({ customerId: customerId, imageRefIds: ids })
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Scan request failed')).then(function(t) {
                    throw new Error(t || 'Scan request failed');
                });
            }
            return resp.json();
        }).then(function(data) {
            // POST /60/ScanJob's response body is the same generic
            // {list, metadata} wrapper every query response uses (verified
            // live), not a bare ScanJob object.
            const job = data && data.list && data.list[0];
            if (!job || !job.scanJobId) {
                throw new Error('No scan job returned');
            }
            Layer8DNotification.success('Scanning ' + ids.length + ' image(s)');
            updateScanButton(SecScanImageSelection.count());
            attachProgressBar(job.scanJobId);
        }).catch(function(err) {
            console.error('Scan Images error:', err);
            Layer8DNotification.error('Failed to start scan: ' + err.message);
        });
    }

    Layer8SectionConfigs.register('dashboard', {
        title: 'Vulnerabilities Dashboard',
        subtitle: 'Overview, image ingestion, and scanning',
        // Real inline SVG (stroke="currentColor") instead of an emoji --
        // emoji render with their own fixed built-in colors on every
        // theme and can't be recolored via CSS.
        icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="M18 17V9M13 17V5M8 17v-3"/></svg>',
        modules: [],
        // .section-content/.main-content are flex containers with
        // overflow:hidden by framework design (base-core.css) -- normal
        // module content is a single self-scrolling Layer8DTable, but this
        // page's KPI cards + chart + toolbar + progress bar are a flat
        // stack with no scrolling of their own, so once their combined
        // height exceeds the available flex space they were silently
        // clipped at the bottom instead of scrolling. Wrapping them in our
        // own scrollable container (secscan-dashboard-content, below)
        // fixes this without touching shared l8ui/base-core.css files.
        customContent:
            '<div class="secscan-dashboard-content">' +
            '<div id="secscan-dashboard-kpi-strip" class="secscan-kpi-strip secscan-kpi-loading">Loading…</div>' +
            '<div class="secscan-dashboard-toolbar">' +
            '<button class="layer8d-btn layer8d-btn-primary layer8d-btn-small" id="secscan-add-images-btn">Add Images</button>' +
            '<button class="layer8d-btn layer8d-btn-primary layer8d-btn-small" id="secscan-scan-images-btn" disabled>Scan Images</button>' +
            '</div>' +
            // Empty on purpose -- Layer8DProgressBar.attach() populates this
            // container with its own generic markup.
            '<div id="secscan-scan-progress" class="secscan-scan-progress" hidden></div>' +
            '<div id="secscan-top-vuln-chart" class="secscan-top-vuln-chart"></div>' +
            '</div>'
    });

    let attached = false;

    window.initializeSecScanDashboard = function() {
        const addBtn = document.getElementById('secscan-add-images-btn');
        const scanBtn = document.getElementById('secscan-scan-images-btn');
        if (addBtn && !addBtn.dataset.secscanAttached) {
            addBtn.dataset.secscanAttached = '1';
            addBtn.addEventListener('click', function() {
                if (typeof SecScanAddImages !== 'undefined') SecScanAddImages.open();
            });
        }
        if (scanBtn && !scanBtn.dataset.secscanAttached) {
            scanBtn.dataset.secscanAttached = '1';
            scanBtn.addEventListener('click', scanSelected);
        }
        if (!attached) {
            attached = true;
            SecScanImageSelection.onChange(updateScanButton);
        }
        updateScanButton(SecScanImageSelection.count());
        // Re-entering the Dashboard mid-scan (navigated away and back) --
        // the progress container was just recreated, so re-attach.
        if (activeJobId) {
            attachProgressBar(activeJobId);
        }
        loadStrip();
    };

    return {};
})();
