/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// SecScan Section Configuration for Layer8SectionGenerator (PRD §11). One
// flat top-level section per sidebar nav item -- each has exactly one
// module with exactly one service, so no sub-tab/sub-nav bar has
// anything to switch between (see secscan.css for the CSS that hides
// that single-item chrome entirely, since Layer8SectionGenerator always
// renders it when a module/service list is non-empty regardless of
// length).
(function() {
    'use strict';

    Layer8SectionConfigs.register('images', {
        title: 'Images',
        subtitle: 'Grouped by image name, across every repo and tag',
        icon: '🖼️',
        initFn: 'initializeSecScanImages',
        modules: [
            {
                key: 'images', label: 'Images', icon: '🖼️', isDefault: true,
                services: [
                    { key: 'groups', label: 'Images', icon: '🖼️', isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('categories', {
        title: 'Categories',
        subtitle: 'Organize image groups by category',
        icon: '🏷️',
        initFn: 'initializeSecScanCategories',
        modules: [
            {
                key: 'categories', label: 'Categories', icon: '🏷️', isDefault: true,
                services: [
                    { key: 'categories', label: 'Categories', icon: '🏷️', isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('scanhistory', {
        title: 'Scan History',
        subtitle: 'Every Trivy scan job run against your images',
        icon: '🕒',
        initFn: 'initializeSecScanScanHistory',
        modules: [
            {
                key: 'scanhistory', label: 'Scan History', icon: '🕒', isDefault: true,
                services: [
                    { key: 'scanjobs', label: 'Scan History', icon: '🕒', isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('customers', {
        title: 'Customers',
        subtitle: 'Customer Catalog (opsadmin only)',
        icon: '🏢',
        initFn: 'initializeSecScanCustomers',
        modules: [
            {
                key: 'customers', label: 'Customers', icon: '🏢', isDefault: true,
                services: [
                    { key: 'customers', label: 'Customers', icon: '🏢', isDefault: true }
                ]
            }
        ]
    });
})();
