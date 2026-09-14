/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Dashboard KPI strip + toolbar (PRD §11.1). Layer8DWidget.renderEnhancedStatsGrid
// is tied to a different app's own "DashboardStats.navigateToSection"
// convention (hardcodes value=0, verified by reading its source) -- not
// usable here; this calls the lower-level Layer8DWidget.render(kpi, value,
// opts) directly instead, once per real computed KPI number.
//
// Layer8DTable has no generic "extra toolbar button" slot (verified: only
// a single onAdd button). The Add Images button and this KPI strip are
// injected directly above the 'groups' table's container after the
// images section renders, following the
// {moduleKey}-{serviceKey}-table-container id convention (AddingModule).
// The Vulnerabilities Report export lives on its own Reports nav section
// (reports-section.js), not here -- it's a cross-image report, not an
// Images-table action.
window.SecScanDashboardKpis = (function() {
    'use strict';

    const GROUPS_CONTAINER_ID = 'images-groups-table-container';

    function countOf(query) {
        const q = encodeURIComponent(JSON.stringify({ text: query }));
        return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgGroup?body=' + q))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) {
                return (data && data.metadata && data.metadata.keyCount && data.metadata.keyCount.counts && data.metadata.keyCount.counts.Total) || 0;
            })
            .catch(function() { return 0; });
    }

    // countOfEndpoint issues the count query against the given service
    // endpoint (ImageRef/ImageRefCve live under different services than
    // ImageGroup, so the query's "from" type and the endpoint must match).
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
            countOfEndpoint('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' limit 1 page 1"),
            // ScanStatus_PENDING = 1 (bare integer, never a quoted name -- verified L8Query rule)
            countOfEndpoint('/60/ImageRef', "select * from ImageRef where customerId='" + customerId + "' and scanStatus=1 limit 1 page 1"),
            // Severity_CRITICAL = 4
            countOfEndpoint('/60/ImgRefCve', "select * from ImageRefCve where customerId='" + customerId + "' and severity=4 limit 1 page 1"),
            countOfEndpoint('/60/ImgGroup', "select * from ImageGroup where customerId='" + customerId + "' and scannedRefCount=0 limit 1 page 1")
        ]).then(function(results) {
            return {
                totalGroups: results[0],
                pendingScans: results[1],
                criticalCves: results[2],
                unscannedGroups: results[3]
            };
        });
    }

    function renderStrip(kpis) {
        const cards = [
            Layer8DWidget.render({ label: 'Images', icon: 'icon-image' }, kpis.totalGroups, {}),
            Layer8DWidget.render({ label: 'Pending Scans', icon: 'icon-clock' }, kpis.pendingScans, {}),
            Layer8DWidget.render({ label: 'Critical CVEs', icon: 'icon-alert' }, kpis.criticalCves, {}),
            Layer8DWidget.render({ label: 'Groups Not Yet Scanned', icon: 'icon-question' }, kpis.unscannedGroups, {})
        ];
        return '<div class="secscan-kpi-strip">' + cards.join('') + '</div>';
    }

    function renderToolbar() {
        return '<div class="secscan-groups-toolbar">' +
            '<button class="layer8d-btn layer8d-btn-primary layer8d-btn-small" id="secscan-add-images-btn">Add Images</button>' +
            '</div>';
    }

    function inject() {
        const container = document.getElementById(GROUPS_CONTAINER_ID);
        if (!container || container.dataset.secscanKpisAttached) {
            return;
        }
        container.dataset.secscanKpisAttached = '1';

        const wrap = document.createElement('div');
        wrap.className = 'secscan-dashboard-top';
        wrap.innerHTML = '<div class="secscan-kpi-strip secscan-kpi-loading">Loading…</div>' + renderToolbar();
        container.parentElement.insertBefore(wrap, container);

        wrap.querySelector('#secscan-add-images-btn').addEventListener('click', function() {
            if (typeof SecScanAddImages !== 'undefined') SecScanAddImages.open();
        });

        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            wrap.querySelector('.secscan-kpi-strip').textContent = 'No customer context found for this session.';
            return;
        }
        loadKpis(customerId).then(function(kpis) {
            const stripEl = wrap.querySelector('.secscan-kpi-strip');
            if (stripEl) stripEl.outerHTML = renderStrip(kpis);
        }).catch(function(err) {
            console.error('Dashboard KPIs: failed to load', err);
        });
    }

    // Section HTML loads via fetch+innerHTML (sections.js), then
    // initializeSecScanImages() runs synchronously, but Layer8DTable's own
    // internal construction of the groups table happens on a schedule this
    // file doesn't control -- poll briefly for the container rather than
    // assume a fixed delay is always enough.
    function injectWhenReady(attemptsLeft) {
        const container = document.getElementById(GROUPS_CONTAINER_ID);
        if (container) {
            inject();
            return;
        }
        if (attemptsLeft > 0) {
            setTimeout(function() { injectWhenReady(attemptsLeft - 1); }, 150);
        }
    }

    return { injectWhenReady: injectWhenReady };
})();
