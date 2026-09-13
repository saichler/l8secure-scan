/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Client-side customerId discovery (PRD §4). Every write in this app needs
// an explicit, trusted customerId in its request body -- the /auth
// response and bearer JWT carry no tenant info at all (verified). Resolved
// by repurposing L8User.portal (normally just a post-login redirect path)
// to also carry the id: customer-role users are provisioned with
// portal = "app.html?customerId=<id>", so the id rides along in the same
// sessionStorage value the real login flow already persists
// (userPortal) -- parsed back out here, not used raw.
window.SecScan = window.SecScan || {};

SecScan.getCurrentCustomerId = function() {
    var portal = sessionStorage.getItem('userPortal') || '';
    var qIndex = portal.indexOf('?');
    if (qIndex === -1) {
        return '';
    }
    var params = new URLSearchParams(portal.substring(qIndex + 1));
    return params.get('customerId') || '';
};
