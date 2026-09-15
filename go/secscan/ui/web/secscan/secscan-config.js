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
    Layer8ModuleConfigFactory.create({
        namespace: 'SecScan',
        modules: {
            // §11.1: table/chart view switch, the only Layer8DViewFactory
            // chart type used in this PRD -- severity comparison across
            // groups using each group's newest (latest scanned) counts.
            'images': mod('Images', 'icon-image', [
                svc('groups', 'Images', 'icon-image', '/60/ImgGroup', 'ImageGroup', 'table', {
                    chartType: 'bar', categoryField: 'imageName', valueField: 'newestCounts.critical',
                    title: 'Critical Vulnerabilities by Image'
                }, ['chart'])
            ]),
            'categories': mod('Categories', 'icon-tag', [
                svc('categories', 'Categories', 'icon-tag', '/60/ImgCat', 'ImageCategory')
            ]),
            'scanhistory': mod('Scan History', 'icon-history', [
                // Endpoint is /60/ScanJobs (the renamed, ORM-backed
                // persistence service) -- the new stateless /60/ScanJob
                // action service's own Get() is stubbed "not supported"
                // (plans/scanjob-live-progress.md Phase 2). modelName stays
                // 'ScanJob', the protobuf type name, unchanged either way.
                svc('scanjobs', 'Scan History', 'icon-history', '/60/ScanJobs', 'ScanJob')
            ]),
            'customers': mod('Customers', 'icon-building', [
                svc('customers', 'Customers', 'icon-building', '/60/Customer', 'Customer')
            ])
        },
        submodules: ['SecScanVuln', 'SecScanAdmin']
    });
})();
