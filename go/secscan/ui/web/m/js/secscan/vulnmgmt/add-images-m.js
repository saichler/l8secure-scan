/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile Add Images bulk ingestion (PRD §6.1/§11.7, parity with desktop's
// add-images.js): a free-text textarea, split into lines client-side and
// POSTed to the custom ImgRefAdd action service. Real, exact mobile
// precedent for a Layer8MPopup with raw content + custom onSave (not a
// generated form): l8learn's people-batch-process-m.js
// (L8BatchProcessMobile.openMultiUpload).
window.SecScanAddImages_M = (function() {
    'use strict';

    // Same 1MB cap as desktop's add-images.js, for the same reason.
    const MAX_FILE_BYTES = 1024 * 1024;

    function open() {
        const html = '<div class="mobile-form-section">' +
            '<label class="mobile-form-label" for="secscan-m-add-images-textarea">Image References (one per line)</label>' +
            '<div class="secscan-m-add-images-file-row">' +
            '<button type="button" class="mobile-popup-btn" id="secscan-m-add-images-file-btn">Load from file…</button>' +
            '<input type="file" id="secscan-m-add-images-file-input" ' +
            'accept=".txt,.csv,.list,text/plain" hidden>' +
            '<span id="secscan-m-add-images-file-note" class="secscan-m-add-images-file-note"></span>' +
            '</div>' +
            '<textarea id="secscan-m-add-images-textarea" rows="8" class="mobile-form-input" style="width:100%;" ' +
            'placeholder="registry.example.com/myorg/backend:v1.4.2"></textarea>' +
            '</div>';

        Layer8MPopup.show({
            title: 'Add Images',
            content: html,
            size: 'large',
            showFooter: true,
            saveButtonText: 'Add',
            onSave: function(popup) { submit(popup); },
            onShow: function(popup) { attachFileLoader(popup.body); }
        });
    }

    // Mobile twin of desktop add-images.js's attachFileLoader -- read
    // locally with FileReader and appended to the same textarea, so submit()
    // stays the only thing that talks to the server and nothing leaves the
    // device until Add is pressed. On a phone the file picker pulls from
    // Files/Drive/iCloud, which is the realistic way a reference list
    // arrives here at all.
    function attachFileLoader(body) {
        const btn = body.querySelector('#secscan-m-add-images-file-btn');
        const input = body.querySelector('#secscan-m-add-images-file-input');
        const note = body.querySelector('#secscan-m-add-images-file-note');
        const textarea = body.querySelector('#secscan-m-add-images-textarea');
        if (!btn || !input || !textarea) return;

        btn.addEventListener('click', function() { input.click(); });

        input.addEventListener('change', function() {
            const file = input.files && input.files[0];
            if (!file) return;
            if (file.size > MAX_FILE_BYTES) {
                input.value = '';
                Layer8MUtils.showError(file.name + ' is larger than 1 MB');
                return;
            }
            const reader = new FileReader();
            reader.onload = function() {
                // Cleared on every path so picking the SAME file again
                // still fires change the second time.
                input.value = '';
                const added = appendLines(textarea, String(reader.result || ''));
                if (note) {
                    note.textContent = added === 0
                        ? 'No image references found in ' + file.name
                        : added + ' loaded from ' + file.name;
                }
            };
            reader.onerror = function() {
                input.value = '';
                Layer8MUtils.showError('Could not read ' + file.name);
            };
            reader.readAsText(file);
        });
    }

    // Appends rather than replaces, and splits on \r?\n so a CRLF file
    // doesn't leave a stray \r on every reference -- see desktop's copy.
    function appendLines(textarea, text) {
        const lines = text.split(/\r?\n/).map(function(l) {
            return l.trim();
        }).filter(function(l) {
            return l.length > 0;
        });
        if (lines.length === 0) return 0;
        const existing = textarea.value.trim();
        textarea.value = (existing ? existing + '\n' : '') + lines.join('\n') + '\n';
        textarea.scrollTop = textarea.scrollHeight;
        return lines.length;
    }

    function submit(popup) {
        const textarea = popup.body.querySelector('#secscan-m-add-images-textarea');
        const raw = textarea ? textarea.value : '';
        const lines = raw.split('\n').map(function(l) { return l.trim(); }).filter(function(l) { return l.length > 0; });

        if (lines.length === 0) {
            Layer8MUtils.showError('Paste at least one image reference');
            return;
        }

        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8MUtils.showError('No customer context found for this session');
            return;
        }

        Layer8MAuth.post(Layer8MConfig.resolveEndpoint('/60/ImgRefAdd'), { customerId: customerId, imageRefStrings: lines })
            .then(function(data) {
                if (!data) return;
                Layer8MPopup.close();
                showSummary(data);
            }).catch(function(err) {
                console.error('Add Images (mobile) error:', err);
                Layer8MUtils.showError('Failed to add images: ' + err.message);
            });
    }

    // No skipped/errored line is ever silently dropped (PRD §11.5/§11.7).
    function showSummary(data) {
        const created = data.created || [];
        const skipped = data.skipped || [];
        const errors = data.errors || [];

        var html = '<div class="secscan-m-add-images-summary">' +
            '<p><strong>' + created.length + '</strong> created, <strong>' + skipped.length +
            '</strong> skipped, <strong>' + errors.length + '</strong> errors.</p>';

        function list(title, items) {
            if (items.length === 0) return '';
            var rows = items.map(function(it) {
                return '<li>' + Layer8MUtils.escapeHtml(it.ref) + ' — ' + Layer8MUtils.escapeHtml(it.reason) + '</li>';
            }).join('');
            return '<h4>' + title + '</h4><ul>' + rows + '</ul>';
        }

        html += list('Skipped (duplicates)', skipped);
        html += list('Errors', errors);
        html += '</div>';

        Layer8MPopup.show({
            title: 'Add Images Result',
            content: html,
            size: 'large',
            showFooter: true,
            showCancelButton: false,
            saveButtonText: 'Close',
            onSave: function() { Layer8MPopup.close(); }
        });
    }

    return { open: open };
})();
