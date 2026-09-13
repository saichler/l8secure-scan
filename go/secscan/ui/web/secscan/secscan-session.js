/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Client-side customerId discovery (PRD §4). Every write in this app needs
// an explicit, trusted customerId in its request body -- the /auth
// response and bearer JWT originally carried no tenant info at all
// (verified). Fixed at the framework level (not a project-local
// workaround): L8User gained a real `customer` attribute, threaded through
// L8Token -> AuthToken (l8secure) -> sessionStorage.userCustomer (l8ui's
// login flow), the same way `portal` already worked. This reads that
// directly -- customer-role users must be provisioned with L8User.customer
// set to their own customerId (Phase 6 mock data, and any future real
// provisioning).
//
// Dependency note: this needs l8ui's login JS to actually persist
// userCustomer (l8ui commit 10d7fc7, not yet pushed/pulled into this
// project's l8ui submodule as of this writing -- see plans/PROGRESS.md).
window.SecScan = window.SecScan || {};

SecScan.getCurrentCustomerId = function() {
    return sessionStorage.getItem('userCustomer') || '';
};
