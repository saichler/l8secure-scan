/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile "Add Category" flow (PRD §11.7, parity with desktop's
// add-category.js): reuses the shared MobileSecScanVuln.forms.ImageCategory
// definition via Layer8MForms.renderForm, but posts by hand (not
// Layer8MNavCrud.openServiceForm's generic POST) so customerId -- a
// trusted, client-supplied value, never a form field -- can be injected.
// Wired as the 'categories' service's onAdd override in the nav-config.
window.SecScanAddCategory_M = (function() {
    'use strict';

    function open(onSuccess) {
        const formDef = MobileSecScanVuln.forms.ImageCategory;
        const content = Layer8MForms.renderForm(formDef, {});

        Layer8MPopup.show({
            title: 'Add Category',
            content: content,
            size: 'large',
            showFooter: true,
            saveButtonText: 'Save',
            onShow: function(popup) {
                Layer8MForms.initFormFields(popup.body, formDef);
            },
            onSave: function(popup) {
                save(popup.body, onSuccess);
            }
        });
    }

    function save(body, onSuccess) {
        const errors = Layer8MForms.validateForm(body);
        if (errors.length > 0) {
            Layer8MForms.showErrors(body, errors);
            return;
        }

        const data = Layer8MForms.getFormData(body);
        const customerId = SecScan.getCurrentCustomerId();
        if (!customerId) {
            Layer8MUtils.showError('No customer context found for this session');
            return;
        }

        const entity = { customerId: customerId, name: data.name, colorCode: data.colorCode || '' };

        Layer8MAuth.post(Layer8MConfig.resolveEndpoint('/60/ImgCat'), entity)
            .then(function() {
                Layer8MPopup.close();
                Layer8MUtils.showSuccess('Category created');
                if (onSuccess) onSuccess();
            }).catch(function(err) {
                console.error('Add Category (mobile) error:', err);
                Layer8MUtils.showError('Failed to create category: ' + err.message);
            });
    }

    return { open: open };
})();
