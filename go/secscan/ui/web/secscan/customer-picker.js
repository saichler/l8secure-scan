/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Post-login customer picker: a logged-in user with no L8User.customer
// scope (e.g. the opsadmin role, PRD §4 -- "customer left empty for this
// role") has no trusted customerId for SecScan.getCurrentCustomerId() to
// return, so every customer-scoped screen/action in this app (dashboard
// KPIs, Add Images, Export CSV, Scan Selected) has nothing to work with.
// This popup lets such a user pick one customer to view for the rest of
// the browser session, writing the SAME sessionStorage.userCustomer key
// secscan-session.js reads -- everything downstream keeps working exactly
// as it does for a real customer-scoped login, no other code needed to
// know the difference.
window.SecScanCustomerPicker = (function() {
    'use strict';

    // Runs once per app.html load, before the rest of the app initializes
    // (secscan-init.js calls this first). No-op if a customer is already
    // scoped (the normal case for a real customer-role login).
    function checkAndPrompt(onReady) {
        if (SecScan.getCurrentCustomerId()) {
            onReady();
            return;
        }
        fetchCustomers().then(function(customers) {
            show(customers, onReady);
        }).catch(function(err) {
            console.error('CustomerPicker: failed to load customers', err);
            show([], onReady);
        });
    }

    function fetchCustomers() {
        const query = encodeURIComponent(JSON.stringify({ text: "select * from Customer where isActive=true" }));
        return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/Customer?body=' + query))
            .then(function(r) { return r ? r.json() : null; })
            .then(function(data) { return (data && data.list) || []; });
    }

    function show(customers, onReady) {
        const html = customers.length
            ? '<p class="secscan-customer-picker-intro">Your account is not scoped to a single customer. ' +
              'Select which customer\'s data to view for this session:</p>' +
              '<select id="secscan-customer-picker-select" class="layer8d-form-input">' +
              customers.map(function(c) {
                  return '<option value="' + Layer8DUtils.escapeHtml(c.customerId) + '">' +
                      Layer8DUtils.escapeHtml(c.name || c.customerId) + '</option>';
              }).join('') +
              '</select>'
            : '<p class="secscan-customer-picker-intro">No active customers exist yet. Create one under ' +
              'Customer Management before using the dashboard.</p>';

        Layer8DPopup.show({
            title: 'Select Customer',
            content: html,
            size: 'small',
            showFooter: false,
            onShow: function(body) {
                if (!customers.length) return;
                const footer = document.createElement('div');
                footer.className = 'probler-popup-footer';
                const btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'btn btn-primary';
                btn.textContent = 'Continue';
                btn.addEventListener('click', function() {
                    const select = body.querySelector('#secscan-customer-picker-select');
                    if (!select || !select.value) return;
                    sessionStorage.setItem('userCustomer', select.value);
                    Layer8DPopup.close();
                    onReady();
                });
                footer.appendChild(btn);
                body.parentElement.appendChild(footer);
            }
        });
    }

    return { checkAndPrompt: checkAndPrompt };
})();
