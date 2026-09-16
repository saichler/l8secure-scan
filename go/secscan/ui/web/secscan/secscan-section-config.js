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

    // Real inline SVG (stroke="currentColor") instead of an emoji glyph --
    // emoji render with their own fixed, built-in colors on every platform
    // and can't be recolored via CSS, so they stayed the same "colorful"
    // look regardless of theme (confirmed real user complaint in noir).
    // These render wherever `icon:` is used (section hero .l8-icon, module
    // tab .tab-icon, subnav .subnav-icon -- layer8-section-generator.js
    // interpolates the string directly as markup, so a full <svg> string
    // works exactly like an emoji character did).
    const ICON = {
        images: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>',
        categories: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41 13.42 20.58a2 2 0 0 1-2.83 0L2.59 12.59a2 2 0 0 1 0-2.83l7.17-7.17A2 2 0 0 1 11.17 2H18a2 2 0 0 1 2 2v6.83a2 2 0 0 1-.59 1.41Z"/><circle cx="7.5" cy="7.5" r="1"/></svg>',
        scanhistory: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>',
        customers: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 22V4a1 1 0 0 1 1-1h8a1 1 0 0 1 1 1v18"/><path d="M6 12H4a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h2"/><path d="M18 9h2a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1h-2"/><path d="M10 6h4M10 10h4M10 14h4M10 18h4"/></svg>'
    };

    Layer8SectionConfigs.register('images', {
        title: 'Images',
        subtitle: 'Grouped by image name, across every repo and tag',
        icon: ICON.images,
        initFn: 'initializeSecScanImages',
        modules: [
            {
                key: 'images', label: 'Images', icon: ICON.images, isDefault: true,
                services: [
                    { key: 'groups', label: 'Images', icon: ICON.images, isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('categories', {
        title: 'Categories',
        subtitle: 'Organize image groups by category',
        icon: ICON.categories,
        initFn: 'initializeSecScanCategories',
        modules: [
            {
                key: 'categories', label: 'Categories', icon: ICON.categories, isDefault: true,
                services: [
                    { key: 'categories', label: 'Categories', icon: ICON.categories, isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('scanhistory', {
        title: 'Scan History',
        subtitle: 'Every Trivy scan job run against your images',
        icon: ICON.scanhistory,
        initFn: 'initializeSecScanScanHistory',
        modules: [
            {
                key: 'scanhistory', label: 'Scan History', icon: ICON.scanhistory, isDefault: true,
                services: [
                    { key: 'scanjobs', label: 'Scan History', icon: ICON.scanhistory, isDefault: true }
                ]
            }
        ]
    });

    Layer8SectionConfigs.register('customers', {
        title: 'Customers',
        subtitle: 'Customer Catalog (opsadmin only)',
        icon: ICON.customers,
        initFn: 'initializeSecScanCustomers',
        modules: [
            {
                key: 'customers', label: 'Customers', icon: ICON.customers, isDefault: true,
                services: [
                    { key: 'customers', label: 'Customers', icon: ICON.customers, isDefault: true }
                ]
            }
        ]
    });
})();
