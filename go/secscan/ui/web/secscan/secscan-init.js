/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

(function() {
    'use strict';

    // A user with no customer scope (opsadmin, PRD §4) must pick one
    // before any customer-scoped module initializes -- otherwise every
    // primary action just fails with "No customer context found for this
    // session" (dashboard KPIs, Add Images, Export CSV, Scan Selected all
    // read SecScan.getCurrentCustomerId() independently). Deferring the
    // two Layer8DModuleFactory.create() calls below until a customer is
    // confirmed avoids rendering those broken states at all for such a
    // user; real customer-role logins already have a customer and this
    // resolves synchronously (checkAndPrompt calls onReady immediately).
    SecScanCustomerPicker.checkAndPrompt(function() {
        initModules();
    });

    function initModules() {
    // One Layer8ModuleConfigFactory namespace ('SecScan', secscan-config.js)
    // holds all four modules' modules{}/submodules[] config; one
    // Layer8DModuleFactory.create() call per flat top-level SECTION attaches
    // navigation/CRUD for it -- each call's sectionSelector must equal its
    // own defaultModule (ModuleInitSectionSelector). All calls share the
    // same 'SecScan' namespace for CRUD/forms facade (harmless to attach
    // repeatedly) but validate only their own submodule.
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

    Layer8DModuleFactory.create({
        namespace: 'SecScan',
        defaultModule: 'customers',
        defaultService: 'customers',
        sectionSelector: 'customers',
        initializerName: 'initializeSecScanCustomers',
        requiredNamespaces: ['SecScanAdmin']
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
    }
})();
