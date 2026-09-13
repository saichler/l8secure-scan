/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// SecScan Section Configuration for Layer8SectionGenerator (PRD §11)
(function() {
    'use strict';

    Layer8SectionConfigs.register('vulnmgmt', {
        title: 'Vulnerability Management',
        subtitle: 'Image Groups, Categories & Scan History',
        icon: '🛡️',
        initFn: 'initializeSecScanVuln',
        modules: [
            {
                key: 'vulnmgmt', label: 'Vulnerability Management', icon: '🛡️', isDefault: true,
                services: [
                    { key: 'groups', label: 'Image Groups', icon: '🖼️', isDefault: true },
                    { key: 'categories', label: 'Categories', icon: '🏷️' },
                    { key: 'scanjobs', label: 'Scan History', icon: '🕒' }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('admin', {
        title: 'Administration',
        subtitle: 'Customer Catalog (opsadmin only)',
        icon: '⚙️',
        initFn: 'initializeSecScanAdmin',
        modules: [
            {
                key: 'admin', label: 'Administration', icon: '⚙️', isDefault: true,
                services: [
                    { key: 'customers', label: 'Customers', icon: '🏢', isDefault: true }
                ]
            }
        ]
    });
})();
