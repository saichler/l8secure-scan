/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Custom "Edit Category" flow (PRD §11.2), mirroring add-category.js: the
// generic Layer8DForms.openEditForm pipeline has no hook to inject
// customerId into the PUT body either (customerId is a trusted,
// client-supplied value, §4/§9, never a form field) -- every edit through
// the generic pipeline failed server-side with "CustomerId is required"
// (ImageCategoryServiceCallback.Require). Add already had this override;
// Edit never did.
window.SecScanEditCategory = (function() {
    'use strict';

    function open(categoryId, onSuccess) {
        Layer8DPopup.show({
            title: 'Edit Category',
            content: '<div style="text-align:center;padding:40px;color:#718096;">Loading...</div>',
            size: 'medium',
            showFooter: false
        });

        Layer8DFormsData.fetchRecord(
            Layer8DConfig.resolveEndpoint('/60/ImgCat'), 'categoryId', categoryId, 'ImageCategory'
        ).then(function(record) {
            if (!record) {
                Layer8DPopup.close();
                Layer8DNotification.error('Category not found');
                return;
            }
            showForm(categoryId, record, onSuccess);
        }).catch(function(err) {
            Layer8DPopup.close();
            Layer8DNotification.error('Failed to load category: ' + err.message);
        });
    }

    function showForm(categoryId, record, onSuccess) {
        const formDef = SecScanVuln.forms.ImageCategory;
        const content = Layer8DForms.generateFormHtml(formDef, record);

        Layer8DPopup.close();
        Layer8DPopup.show({
            title: 'Edit Category',
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
                save(categoryId, record.customerId, onSuccess);
            }
        });
    }

    function save(categoryId, customerId, onSuccess) {
        const formDef = SecScanVuln.forms.ImageCategory;
        const data = Layer8DForms.collectFormData(formDef);
        if (!data) return;

        if (!data.name || !String(data.name).trim()) {
            Layer8DNotification.error('Name is required');
            return;
        }

        const entity = {
            categoryId: categoryId,
            customerId: customerId,
            name: data.name,
            colorCode: data.colorCode || ''
        };

        makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgCat'), {
            method: 'PUT',
            body: JSON.stringify(entity)
        }).then(function(resp) {
            if (!resp || !resp.ok) {
                return (resp ? resp.text() : Promise.resolve('Save failed')).then(function(t) {
                    throw new Error(t || 'Save failed');
                });
            }
            Layer8DPopup.close();
            Layer8DNotification.success('Category updated');
            if (onSuccess) onSuccess();
        }).catch(function(err) {
            console.error('Edit Category error:', err);
            Layer8DNotification.error('Failed to update category: ' + err.message);
        });
    }

    return { open: open };
})();
