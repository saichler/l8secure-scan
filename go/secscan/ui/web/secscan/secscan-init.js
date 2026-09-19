/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

(function() {
    'use strict';

    // Exposed globally rather than self-invoked here: this whole file runs
    // synchronously at <script> parse time, BEFORE app.js's
    // DOMContentLoaded handler has awaited Layer8DConfig.load() -- calling
    // SecScanCustomerPicker.checkAndPrompt() (and therefore
    // Layer8DConfig.resolveEndpoint()) from here used to run against the
    // default, unloaded apiPrefix ('', not '/scan'), a real bug that broke
    // every fetch the customer picker and every module's table made.
    // app.js calls window.initializeSecScanModules() itself, from inside
    // its own already-config-loaded, already-customer-gated flow.
    // Customers is opsadmin-only and intentionally NOT customer-scoped
    // (secscan-config.js) -- split out from initializeSecScanModules so
    // app.js can register it BEFORE a customer is picked, not just after.
    // Real, confirmed bug otherwise: with zero customers seeded, the
    // picker has nothing to select, SecScanCustomerPicker.checkAndPrompt's
    // callback (which every OTHER module's registration waited behind)
    // never fires, and Customer Management -- the one place opsadmin
    // could actually create the first customer -- never initializes
    // either. A dead end with no way out except editing the database
    // directly. Safe to call twice (idempotent, same as every other
    // Layer8DModuleFactory.create() call already sharing the 'SecScan'
    // namespace) -- initializeSecScanModules() below no longer calls it
    // a second time, so in practice it only ever runs once anyway.
    window.initializeSecScanCustomersModule = function() {
        Layer8DModuleFactory.create({
            namespace: 'SecScan',
            defaultModule: 'customers',
            defaultService: 'customers',
            sectionSelector: 'customers',
            initializerName: 'initializeSecScanCustomers',
            requiredNamespaces: ['SecScanAdmin']
        });
    };

    window.initializeSecScanModules = function() {
    // One Layer8ModuleConfigFactory namespace ('SecScan', secscan-config.js)
    // holds all four modules' modules{}/submodules[] config; one
    // Layer8DModuleFactory.create() call per flat top-level SECTION attaches
    // navigation/CRUD for it -- each call's sectionSelector must equal its
    // own defaultModule (ModuleInitSectionSelector). All calls share the
    // same 'SecScan' namespace for CRUD/forms facade (harmless to attach
    // repeatedly) but validate only their own submodule.
    //
    // Images/Categories/Scan History all use baseWhereClause: customerScoped
    // (secscan-config.js) -- a falsy baseWhereClause is dropped entirely by
    // the generic query builder (layer8d-table-data.js), not treated as
    // "match nothing", so these three genuinely must stay gated behind a
    // picked customer (app.js) or they'd show every customer's data
    // unfiltered. Customers itself has no such scoping and is registered
    // separately, unconditionally, by initializeSecScanCustomersModule
    // above.
    Layer8DModuleFactory.create({
        namespace: 'SecScan',
        defaultModule: 'images',
        defaultService: 'groups',
        sectionSelector: 'images',
        initializerName: 'initializeSecScanImages',
        requiredNamespaces: ['SecScanVuln']
    });

    Layer8DModuleFactory.create({
        namespace: 'SecScan',
        defaultModule: 'categories',
        defaultService: 'categories',
        sectionSelector: 'categories',
        initializerName: 'initializeSecScanCategories',
        requiredNamespaces: ['SecScanVuln']
    });

    Layer8DModuleFactory.create({
        namespace: 'SecScan',
        defaultModule: 'scanhistory',
        defaultService: 'scanjobs',
        sectionSelector: 'scanhistory',
        initializerName: 'initializeSecScanScanHistory',
        requiredNamespaces: ['SecScanVuln']
    });

    // Custom CRUD handlers (SpecialCases pattern): capture the generic
    // handler BEFORE overwriting, delegate everything except the models
    // this project handles custom, per PRD:
    // - ImageGroup row click -> Group Detail popup (§11.3), not the
    //   generic read-only details modal.
    // - ImageCategory Add -> customerId must be injected (§4/§9); the
    //   generic openAddForm pipeline has no field-injection hook.
    var origShowDetails = SecScan._showDetailsModal;
    SecScan._showDetailsModal = function(service, item, itemId) {
        if (service.model === 'ImageGroup' && typeof SecScanGroupDetail !== 'undefined') {
            SecScanGroupDetail.open(item.imageGroupId);
            return;
        }
        origShowDetails.call(SecScan, service, item, itemId);
    };

    var origOpenAdd = SecScan._openAddModal;
    SecScan._openAddModal = function(service) {
        if (service.model === 'ImageCategory' && typeof SecScanAddCategory !== 'undefined') {
            SecScanAddCategory.open(function() { SecScan.refreshCurrentTable(); });
            return;
        }
        origOpenAdd.call(SecScan, service);
    };

    // - ImageCategory Edit -> same customerId-injection gap as Add, just
    //   never given the same override (edit-category.js).
    var origOpenEdit = SecScan._openEditModal;
    SecScan._openEditModal = function(service, id) {
        if (service.model === 'ImageCategory' && typeof SecScanEditCategory !== 'undefined') {
            SecScanEditCategory.open(id, function() { SecScan.refreshCurrentTable(); });
            return;
        }
        origOpenEdit.call(SecScan, service, id);
    };
    };
})();
