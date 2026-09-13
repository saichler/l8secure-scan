/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

window.SecScanAdmin = window.SecScanAdmin || {};

(function() {
    'use strict';
    const f = Layer8FormFactory;

    // PRD §11.6: the only place Customer rows are created (opsadmin only).
    SecScanAdmin.forms = SecScanAdmin.forms || {};
    SecScanAdmin.forms.Customer = f.form('Customer', [
        f.section('Customer Details', [
            ...f.text('name', 'Name', true),
            ...f.checkbox('isActive', 'Active')
        ])
    ]);
})();
