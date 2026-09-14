/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Post-login customer picker: a logged-in user with no L8User.customer
// scope (e.g. the opsadmin role, PRD §4 -- "customer left empty for this
// role") has nothing for SecScan.getCurrentCustomerId() to return, so
// every customer-scoped action (dashboard KPIs, Add Images, Export CSV,
// Scan Selected) just failed with "No customer context found for this
// session". This popup lets such a user pick one customer to view for
// the rest of the browser session, writing the SAME sessionStorage
// userCustomer key secscan-session.js reads -- everything downstream
// keeps working exactly as it does for a real customer-scoped login.
//
// Uses Layer8DReferencePicker, the same searchable picker component
// group-detail.js's category picker already uses (attach an input,
// endpoint/modelName/idColumn/displayColumn, onChange(id)) -- not a
// hand-rolled <select>, matching this project's established pattern for
// "pick one row from another entity's table".
window.SecScanCustomerPicker = (function() {
    'use strict';

    // Two independent call sites need to wait on this (app.js's initial
    // loadSection('vulnmgmt') and secscan-init.js's module setup) -- both
    // run before a customer is necessarily picked, so this must be safe to
    // call more than once concurrently: queue callbacks and show the
    // popup only once, rather than stacking a second popup on a second
    // call while the first is still open.
    let popupOpen = false;
    const waiters = [];

    // No-op if a customer is already scoped (the normal case for a real
    // customer-role login) -- callback runs synchronously in that case.
    function checkAndPrompt(onReady) {
        if (SecScan.getCurrentCustomerId()) {
            onReady();
            return;
        }
        waiters.push(onReady);
        if (popupOpen) return;
        popupOpen = true;
        show(function() {
            popupOpen = false;
            const pending = waiters.splice(0, waiters.length);
            pending.forEach(function(fn) { fn(); });
        });
    }

    function show(onReady) {
        const html = '<p class="secscan-customer-picker-intro">Your account is not scoped to a single ' +
            'customer. Select which customer\'s data to view for this session:</p>' +
            '<div id="secscan-customer-picker-wrap"><input type="text" id="secscan-customer-picker-input" ' +
            'class="layer8d-form-input" placeholder="Click to select a customer..." readonly></div>';

        Layer8DPopup.show({
            title: 'Select Customer',
            content: html,
            size: 'small',
            showFooter: false,
            onShow: function(body) {
                const input = body.querySelector('#secscan-customer-picker-input');
                Layer8DReferencePicker.attach(input, {
                    endpoint: Layer8DConfig.resolveEndpoint('/60/Customer'),
                    modelName: 'Customer',
                    idColumn: 'customerId',
                    displayColumn: 'name',
                    baseWhereClause: 'isActive=true',
                    title: 'Select Customer',
                    emptyMessage: 'No active customers exist yet. Create one under Customer Management.',
                    onChange: function(id) {
                        if (!id) return;
                        sessionStorage.setItem('userCustomer', id);
                        Layer8DPopup.close();
                        onReady();
                    }
                });
                Layer8DReferencePicker.open(input);
            }
        });
    }

    return { checkAndPrompt: checkAndPrompt };
})();
