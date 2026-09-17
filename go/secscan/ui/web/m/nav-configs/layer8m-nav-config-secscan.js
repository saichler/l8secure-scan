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

    // opsadmin has no server-side row restriction to its own customer (PRD
    // §9), so the customer-picker's "focused view" is enforced here, client
    // side, mirroring desktop's secscan-config.js customerScoped(). A real
    // customer-role account is already correctly restricted server side
    // regardless of this. A function (not a plain string) because the
    // picked customer is only known once the post-login picker resolves,
    // well after this static config runs at script-load time.
    function customerScoped() {
        const cid = (typeof SecScan !== 'undefined' && SecScan.getCurrentCustomerId) ? SecScan.getCurrentCustomerId() : '';
        return cid ? "customerId='" + cid + "'" : null;
    }
    LAYER8M_NAV_CONFIG.customerScoped = customerScoped;

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
                    baseWhereClause: customerScoped,
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
                    baseWhereClause: customerScoped,
                    // Latest scans first -- requestedAt, not completedAt,
                    // same reasoning as desktop's secscan-config.js
                    // (completedAt stays 0 for any still-running job).
                    defaultSort: { column: 'requestedAt', direction: 'desc' },
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

    // Real inline SVG (stroke="currentColor") instead of an emoji glyph --
    // emoji render with their own fixed built-in colors on every theme and
    // can't be recolored via CSS (.nav-card-icon svg already themes stroke
    // via var(--layer8d-primary), it just never had real SVG to apply to).
    LAYER8M_NAV_CONFIG.icons = {
        dashboard: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="M18 17V9M13 17V5M8 17v-3"/></svg>',
        security: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/></svg>',
        settings: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1.03 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 8.98 19.4a1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.7 1.7 0 0 0 .34-1.87 1.7 1.7 0 0 0-1.56-1.03H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 8.98a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.7 1.7 0 0 0 1.87.34H9a1.7 1.7 0 0 0 1.03-1.56V3a2 2 0 1 1 4 0v.09a1.7 1.7 0 0 0 1.03 1.56 1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.7 1.7 0 0 0-.34 1.87V9c.22.63.8 1.03 1.56 1.03H21a2 2 0 1 1 0 4h-.09a1.7 1.7 0 0 0-1.51 1Z"/></svg>',
        system: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>',
        health: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 8-4-16-3 8H2"/></svg>',
        back: '&#x2190;',
        default: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/></svg>'
    };

    LAYER8M_NAV_CONFIG.getIcon = function(key) {
        return this.icons[key] || this.icons['default'];
    };
})();
