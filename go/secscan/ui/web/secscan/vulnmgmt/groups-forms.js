/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const f = Layer8FormFactory;

    function ro(fields) {
        return fields.map(function(field) { field.readOnly = true; return field; });
    }

    // No "Add Group" form -- ImageGroup rows only ever emerge from ImageRef
    // ingestion (PRD §9). Row click opens the custom Group Detail popup
    // (group-detail.js), not this generic form; this exists as a fallback
    // for DataCompletenessPipeline / any generic "view" pipeline that might
    // still reach it. Category is the one field this app lets a user
    // change, but it's edited via a reference picker in Group Detail's
    // header, not through this form.
    SecScanVuln.forms = SecScanVuln.forms || {};
    SecScanVuln.forms.ImageGroup = f.form('Image Group', [
        f.section('Group Details', [
            ...ro(f.text('imageName', 'Name')),
            ...ro(f.reference('categoryId', 'Category', 'ImageCategory')),
            ...ro(f.number('imageRefCount', 'Image Ref Count'))
        ])
    ]);
})();
