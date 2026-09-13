/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanVuln = window.SecScanVuln || {};

(function() {
    'use strict';
    const f = Layer8FormFactory;

    // customerId is never a form field (PRD §4/§9 -- always injected from
    // the logged-in session, never user-typed). See secscan-init.js for how
    // it's attached before POST.
    SecScanVuln.forms = SecScanVuln.forms || {};
    SecScanVuln.forms.ImageCategory = f.form('Category', [
        f.section('Category Details', [
            ...f.text('name', 'Name', true),
            ...f.colorCode('colorCode', 'Color')
        ])
    ]);
})();
