/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// SecScan Module Configuration (PRD §11). Flat top-level modules -- one
// per sidebar nav item (Images, Categories, Scan History, Customers), no
// nested sub-navigation: "ImageGroup" is a backend/ORM concept only, the
// UI-facing name is "Images", and there is no reason to bury Categories/
// Scan History/Customers a level deeper behind a "Vulnerability
// Management"/"Administration" tab bar when each is its own real,
// independent page.
(function() {
    'use strict';
    const svc = Layer8ModuleConfigFactory.service;
    const mod = Layer8ModuleConfigFactory.module;

    // opsadmin has no server-side row restriction to its own customer (PRD
    // §9 -- opsadmin must see every customer), so the customer-picker's
    // "focused view" for a picked customer is enforced ONLY here, client
    // side, via baseWhereClause. A real customer-role account (e.g.
    // local-user) is already correctly restricted server side to its own
    // customerId regardless of this. Evaluated as a function (not a plain
    // string) because the picked customer is only known at table-init
    // time, well after this static config runs at script-load time -- an
    // empty selection means "no customer chosen yet", which must NOT
    // resolve to an unfiltered/all-customers query.
    function customerScoped() {
        const cid = (typeof SecScan !== 'undefined' && SecScan.getCurrentCustomerId) ? SecScan.getCurrentCustomerId() : '';
        return cid ? "customerId='" + cid + "'" : null;
    }

    const groupsService = svc('groups', 'Images', 'icon-image', '/60/ImgGroup', 'ImageGroup', 'table', {
        chartType: 'bar', categoryField: 'imageName', valueField: 'newestCounts.critical',
        title: 'Critical Vulnerabilities by Image'
    }, ['chart']);
    groupsService.baseWhereClause = customerScoped;

    const categoriesService = svc('categories', 'Categories', 'icon-tag', '/60/ImgCat', 'ImageCategory');
    categoriesService.baseWhereClause = customerScoped;

    // Endpoint is /60/ScanJobs (the renamed, ORM-backed persistence
    // service) -- the new stateless /60/ScanJob action service's own
    // Get() is stubbed "not supported" (plans/scanjob-live-progress.md
    // Phase 2). modelName stays 'ScanJob', the protobuf type name,
    // unchanged either way.
    const scanjobsService = svc('scanjobs', 'Scan History', 'icon-history', '/60/ScanJobs', 'ScanJob');
    scanjobsService.baseWhereClause = customerScoped;
    // Latest scans first -- requestedAt (not completedAt) since it's
    // always set at job creation, while completedAt stays 0 for any
    // still-running job, which would otherwise sort every in-progress
    // scan to the bottom regardless of how recently it started.
    scanjobsService.defaultSort = { column: 'requestedAt', direction: 'desc' };

    Layer8ModuleConfigFactory.create({
        namespace: 'SecScan',
        modules: {
            // §11.1: table/chart view switch, the only Layer8DViewFactory
            // chart type used in this PRD -- severity comparison across
            // groups using each group's newest (latest scanned) counts.
            'images': mod('Images', 'icon-image', [groupsService]),
            'categories': mod('Categories', 'icon-tag', [categoriesService]),
            'scanhistory': mod('Scan History', 'icon-history', [scanjobsService]),
            // Customers is opsadmin-only and intentionally NOT
            // customer-scoped -- opsadmin manages every customer record
            // regardless of which one is picked for the focused view.
            'customers': mod('Customers', 'icon-building', [
                svc('customers', 'Customers', 'icon-building', '/60/Customer', 'Customer')
            ])
        },
        submodules: ['SecScanVuln', 'SecScanAdmin']
    });
})();
