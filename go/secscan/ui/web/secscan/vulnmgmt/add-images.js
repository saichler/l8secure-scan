/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Add Images bulk ingestion (PRD §6.1/§11.5): a single free-text textarea,
// split into lines client-side and POSTed to the custom ImgRefAdd action
// service -- not a normal entity form (no 1:1 field mapping), so this is
// hand-built, not Layer8DForms.openAddForm.
window.SecScanAddImages = (function() {
    'use strict';

    function open() {
        const html = '<div class="layer8d-form-section">' +
            '<label for="secscan-add-images-textarea">Image References (one per line)</label>' +
            '<textarea id="secscan-add-images-textarea" rows="10" style="width:100%;" ' +
            'placeholder="registry.example.com/myorg/backend:v1.4.2"></textarea>' +
            '</div>';

        Layer8DPopup.show({
            title: 'Add Images',
            content: html,
            size: 'medium',
            showFooter: true,
            saveButtonText: 'Add',
            onSave: submit
        });
    }

    function submit() {
        const body = Layer8DPopup.getBody();
        const textarea = body ? body.querySelector('#secscan-add-images-textarea') : document.getElementById('secscan-add-images-textarea');
        const raw = textarea ? textarea.value : '';
        const lines = raw.split('\n').map(function(l) { return l.trim(); }).filter(function(l) { return l.length > 0; });

        if (lines.length === 0) {
            Layer8DNotification.error('Paste at least one image reference');
            return;
        }

        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8DNotification.error('No customer context found for this session');
            return;
        }

        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgRefAdd'), {
            method: 'POST',
            body: JSON.stringify({ customerId: customerId, imageRefStrings: lines })
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Request failed')).then(function(t) {
                    throw new Error(t || 'Request failed');
                });
            }
            return resp.json();
        }).then(function(data) {
            if (!data) return;
            Layer8DPopup.close();
            showSummary(data);
            if (typeof SecScan !== 'undefined' && SecScan.refreshCurrentTable) {
                SecScan.refreshCurrentTable();
            }
        }).catch(function(err) {
            console.error('Add Images error:', err);
            Layer8DNotification.error('Failed to add images: ' + err.message);
        });
    }

    // Per ReportInfraBugs "No Silent Fallbacks" -- every skipped/errored
    // line is shown to the user, never silently dropped (PRD §11.5).
    function showSummary(data) {
        const created = data.created || [];
        const skipped = data.skipped || [];
        const errors = data.errors || [];

        var html = '<div class="secscan-add-images-summary">' +
            '<p><strong>' + created.length + '</strong> created, <strong>' + skipped.length +
            '</strong> skipped, <strong>' + errors.length + '</strong> errors.</p>';

        function list(title, items) {
            if (items.length === 0) return '';
            var rows = items.map(function(it) {
                return '<li>' + Layer8DUtils.escapeHtml(it.ref) + ' — ' + Layer8DUtils.escapeHtml(it.reason) + '</li>';
            }).join('');
            return '<h4>' + title + '</h4><ul>' + rows + '</ul>';
        }

        html += list('Skipped (duplicates)', skipped);
        html += list('Errors', errors);
        html += '</div>';

        Layer8DPopup.show({
            title: 'Add Images Result',
            content: html,
            size: 'medium',
            showFooter: true,
            showCancelButton: false,
            saveButtonText: 'Close',
            onSave: function() { Layer8DPopup.close(); }
        });
    }

    return { open: open };
})();
