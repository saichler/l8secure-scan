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

    // Cap on a loaded file. A reference is well under 200 bytes, so 1MB is
    // thousands of images -- far past any real list, while still small
    // enough that reading it can't lock up the tab.
    const MAX_FILE_BYTES = 1024 * 1024;

    function open() {
        const html = '<div class="layer8d-form-section">' +
            '<label for="secscan-add-images-textarea">Image References (one per line)</label>' +
            '<div class="secscan-add-images-file-row">' +
            '<button type="button" class="l8-btn l8-btn-small" id="secscan-add-images-file-btn">Load from file…</button>' +
            '<input type="file" id="secscan-add-images-file-input" ' +
            'accept=".txt,.csv,.list,text/plain" hidden>' +
            '<span id="secscan-add-images-file-note" class="secscan-add-images-file-note"></span>' +
            '</div>' +
            '<textarea id="secscan-add-images-textarea" rows="10" style="width:100%;" ' +
            'placeholder="registry.example.com/myorg/backend:v1.4.2"></textarea>' +
            '</div>';

        Layer8DPopup.show({
            title: 'Add Images',
            content: html,
            size: 'medium',
            showFooter: true,
            saveButtonText: 'Add',
            onSave: submit,
            onShow: attachFileLoader
        });
    }

    // The list of references usually comes out of a cluster dump or a
    // spreadsheet export rather than somebody's clipboard, so it can be
    // loaded from a file instead of pasted. Read locally with FileReader
    // and appended to the same textarea: submit() below stays the only
    // thing that talks to the server, so this needs no upload endpoint and
    // nothing leaves the browser until Add is pressed.
    function attachFileLoader(body) {
        const btn = body.querySelector('#secscan-add-images-file-btn');
        const input = body.querySelector('#secscan-add-images-file-input');
        const note = body.querySelector('#secscan-add-images-file-note');
        const textarea = body.querySelector('#secscan-add-images-textarea');
        if (!btn || !input || !textarea) return;

        btn.addEventListener('click', function() { input.click(); });

        input.addEventListener('change', function() {
            const file = input.files && input.files[0];
            if (!file) return;
            if (file.size > MAX_FILE_BYTES) {
                input.value = '';
                Layer8DNotification.error(file.name + ' is larger than 1 MB');
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
                Layer8DNotification.error('Could not read ' + file.name);
            };
            reader.readAsText(file);
        });
    }

    // Appends rather than replaces, so a second file adds to the list and
    // anything already typed survives. Splits on \r?\n so a CRLF file off
    // a Windows box doesn't leave a stray \r on every reference, and drops
    // blanks here so the count reported back is the number of references
    // actually added -- submit() trims and filters again regardless.
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
