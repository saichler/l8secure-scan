/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// SecScan Module Configuration (PRD §11)
(function() {
    'use strict';
    const svc = Layer8ModuleConfigFactory.service;
    const mod = Layer8ModuleConfigFactory.module;
    Layer8ModuleConfigFactory.create({
        namespace: 'SecScan',
        modules: {
            'vulnmgmt': mod('Vulnerability Management', 'icon-shield', [
                // §11.1: table/chart view switch, the only Layer8DViewFactory
                // chart type used in this PRD -- severity comparison across
                // groups using each group's newest (latest scanned) counts.
                svc('groups', 'Image Groups', 'icon-image', '/60/ImgGroup', 'ImageGroup', 'table', {
                    chartType: 'bar', categoryField: 'imageName', valueField: 'newestCounts.critical',
                    title: 'Critical Vulnerabilities by Group'
                }, ['chart']),
                svc('categories', 'Categories', 'icon-tag', '/60/ImgCat', 'ImageCategory'),
                svc('scanjobs', 'Scan History', 'icon-history', '/60/ScanJob', 'ScanJob')
            ]),
            'admin': mod('Administration', 'icon-settings', [
                svc('customers', 'Customers', 'icon-building', '/60/Customer', 'Customer')
            ])
        },
        submodules: ['SecScanVuln', 'SecScanAdmin']
    });
})();
