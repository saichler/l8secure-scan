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
    }

    function updateScanButton(count) {
        const btn = document.getElementById('secscan-scan-images-btn');
        if (!btn) return;
        btn.disabled = count === 0;
        btn.textContent = count === 0 ? 'Scan Images' : 'Scan Images (' + count + ')';
    }

    function scanSelected() {
        const ids = SecScanImageSelection.getIds();
        if (ids.length === 0) return;
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
            Layer8DNotification.success('Scanning ' + ids.length + ' image(s) — check Scan History for progress');
            SecScanImageSelection.clear();
        }).catch(function(err) {
            console.error('Scan Images error:', err);
            Layer8DNotification.error('Failed to queue scan: ' + err.message);
        });
    }

    Layer8SectionConfigs.register('dashboard', {
        title: 'Vulnerabilities Dashboard',
        subtitle: 'Overview, image ingestion, and scanning',
        icon: '📈',
        modules: [],
        customContent:
            '<div id="secscan-dashboard-kpi-strip" class="secscan-kpi-strip secscan-kpi-loading">Loading…</div>' +
            '<div class="secscan-dashboard-toolbar">' +
            '<button class="layer8d-btn layer8d-btn-primary layer8d-btn-small" id="secscan-add-images-btn">Add Images</button>' +
            '<button class="layer8d-btn layer8d-btn-primary layer8d-btn-small" id="secscan-scan-images-btn" disabled>Scan Images</button>' +
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
        loadStrip();
    };

    return {};
})();
