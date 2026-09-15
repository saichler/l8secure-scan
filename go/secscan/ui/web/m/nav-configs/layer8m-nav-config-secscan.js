/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile navigation configuration (PRD §11.7): two sections mirroring
// desktop's 2-section structure (vulnmgmt: groups/categories/scanjobs,
// admin: customers), plus a minimal System section (Health Monitor +
// L8Security's Users/Roles/Credentials, the same shared framework columns
// desktop's System section already uses via l8ui/sys/security/). Giving
// each module exactly ONE subModules entry makes Layer8MNav auto-skip
// straight to that submodule's services list (verified, layer8m-nav.js) --
// admin/system additionally auto-skip straight into their single service's
// table since each has exactly one service too.
(function() {
    'use strict';

    window.LAYER8M_NAV_CONFIG = window.LAYER8M_NAV_CONFIG || {};

    LAYER8M_NAV_CONFIG.modules = [
        { key: 'dashboard', label: 'Dashboard', icon: 'dashboard', hasSubModules: false },
        { key: 'vulnmgmt', label: 'Vulnerability Management', icon: 'security', hasSubModules: true },
        { key: 'admin', label: 'Administration', icon: 'settings', hasSubModules: true },
        { key: 'system', label: 'System', icon: 'system', hasSubModules: true }
    ];

    LAYER8M_NAV_CONFIG.vulnmgmt = {
        subModules: [
            { key: 'vulnmgmt', label: 'Vulnerability Management', icon: 'security' }
        ],
        services: {
            vulnmgmt: [
                {
                    key: 'groups', label: 'Image Groups', icon: 'default',
                    endpoint: '/60/ImgGroup', model: 'ImageGroup', idField: 'imageGroupId',
                    customInit: 'SecScanGroupsView_M', customContainer: 'secscan-m-groups-view-container',
                    subtitle: 'Tap a group to view details'
                },
                {
                    key: 'categories', label: 'Categories', icon: 'default',
                    endpoint: '/60/ImgCat', model: 'ImageCategory', idField: 'categoryId',
                    onAdd: function() {
                        SecScanAddCategory_M.open(function() {
                            var t = window._Layer8MNavActiveTable;
                            if (t) t.refresh();
                        });
                    }
                },
                {
                    // Endpoint is /60/ScanJobs (the renamed, ORM-backed
                    // persistence service) -- the new stateless /60/ScanJob
                    // action service's own Get() is stubbed "not supported"
                    // (plans/scanjob-live-progress.md Phase 2). model stays
                    // 'ScanJob', the protobuf type name, unchanged either way.
                    key: 'scanjobs', label: 'Scan History', icon: 'default',
                    endpoint: '/60/ScanJobs', model: 'ScanJob', idField: 'scanJobId', readOnly: true,
                    onRowClick: function(item) {
                        Layer8MNavCrud.showRecordDetails(
                            { label: 'Scan Job', model: 'ScanJob', endpoint: '/60/ScanJobs', idField: 'scanJobId' },
                            MobileSecScanVuln.forms.ScanJob,
                            item
                        );
                    }
                }
            ]
        }
    };

    LAYER8M_NAV_CONFIG.admin = {
        subModules: [
            { key: 'admin', label: 'Administration', icon: 'settings' }
        ],
        services: {
            admin: [
                { key: 'customers', label: 'Customers', icon: 'default', endpoint: '/60/Customer', model: 'Customer', idField: 'customerId' }
            ]
        }
    };

    LAYER8M_NAV_CONFIG.system = {
        subModules: [
            { key: 'health', label: 'Health', icon: 'health' },
            { key: 'security', label: 'Security', icon: 'security' }
        ],
        services: {
            health: [
                { key: 'health-monitor', label: 'Health Monitor', icon: 'health', endpoint: '/0/Health', model: 'L8Health', idField: 'service', readOnly: true }
            ],
            security: [
                { key: 'users', label: 'Users', icon: 'default', endpoint: '/73/users', model: 'L8User', idField: 'userId' },
                { key: 'roles', label: 'Roles', icon: 'default', endpoint: '/74/roles', model: 'L8Role', idField: 'roleId' },
                { key: 'credentials', label: 'Credentials', icon: 'default', endpoint: '/75/Creds', model: 'L8Credentials', idField: 'id' }
            ]
        }
    };

    LAYER8M_NAV_CONFIG.icons = {
        dashboard: '&#x1F4CA;',
        security: '&#x1F6E1;&#xFE0F;',
        settings: '&#x2699;&#xFE0F;',
        system: '&#x1F527;',
        health: '&#x1F49A;',
        back: '&#x2190;',
        default: '&#x1F4C4;'
    };

    LAYER8M_NAV_CONFIG.getIcon = function(key) {
        return this.icons[key] || this.icons['default'];
    };
})();
