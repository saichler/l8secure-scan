/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Custom "Add Category" flow (PRD §11.2): the generic Layer8DForms
// pipeline has no hook to inject customerId (a trusted, client-supplied
// value, §4/§9, never a form field) into the POST body, so this is a
// direct popup + fetch, not Layer8DForms.openAddForm.
window.SecScanAddCategory = (function() {
    'use strict';

    function open(onSuccess) {
        const formDef = SecScanVuln.forms.ImageCategory;
        const content = Layer8DForms.generateFormHtml(formDef, {});

        Layer8DPopup.show({
            title: 'Add Category',
            content: content,
            size: 'medium',
            showFooter: true,
            saveButtonText: 'Save',
            onShow: function(body) {
                // InlinePopupRenderingParity: attachDatePickers also attaches
                // input formatters + reference pickers.
                if (typeof Layer8DFormsPickers !== 'undefined') {
                    Layer8DFormsPickers.attachDatePickers(body);
                } else if (typeof Layer8DForms !== 'undefined' && Layer8DForms.attachDatePickers) {
                    Layer8DForms.attachDatePickers(body);
                }
            },
            onSave: function() {
                save(onSuccess);
            }
        });
    }

    function save(onSuccess) {
        const formDef = SecScanVuln.forms.ImageCategory;
        const data = Layer8DForms.collectFormData(formDef);
        if (!data) return;

        if (!data.name || !String(data.name).trim()) {
            Layer8DNotification.error('Name is required');
            return;
        }

        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8DNotification.error('No customer context found for this session');
            return;
        }

        const entity = { customerId: customerId, name: data.name, colorCode: data.colorCode || '' };

        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgCat'), {
            method: 'POST',
            body: JSON.stringify(entity)
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Save failed')).then(function(t) {
                    throw new Error(t || 'Save failed');
                });
            }
            Layer8DPopup.close();
            Layer8DNotification.success('Category created');
            if (onSuccess) onSuccess();
        }).catch(function(err) {
            console.error('Add Category error:', err);
            Layer8DNotification.error('Failed to create category: ' + err.message);
        });
    }

    return { open: open };
})();
