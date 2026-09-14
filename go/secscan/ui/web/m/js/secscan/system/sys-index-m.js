/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile System module registry (PRD §11.7 parity with desktop's System
// section, l8ui/sys/l8sys-init.js): combines this project's own
// MobileSysHealth (health-columns-m.js) with L8Security, the shared
// framework's L8User/L8Role/L8Credentials columns/forms
// (l8ui/sys/security/l8security-{enums,columns,forms}.js -- loaded earlier
// on this page, same files desktop's app.html already loads, real
// framework-level types, not project-specific). 'MobileSystem' is added to
// l8ui/m/js/layer8m-nav-data.js's registry whitelist already (present
// before this project touched it -- a reserved, generic name).
(function() {
    'use strict';

    var modules = [MobileSysHealth, L8Security];

    function findModule(modelName) {
        for (var i = 0; i < modules.length; i++) {
            if (modules[i].columns && modules[i].columns[modelName]) return modules[i];
        }
        return null;
    }

    window.MobileSystem = {
        getFormDef: function(modelName) {
            var mod = findModule(modelName);
            return (mod && mod.forms && mod.forms[modelName]) ? mod.forms[modelName] : null;
        },
        getColumns: function(modelName) {
            var mod = findModule(modelName);
            return (mod && mod.columns && mod.columns[modelName]) ? mod.columns[modelName] : null;
        },
        getTransformData: function(modelName) {
            var mod = findModule(modelName);
            return (mod && mod.transformData) ? mod.transformData : null;
        },
        hasModel: function(modelName) {
            return findModule(modelName) !== null;
        },
        modules: { Health: MobileSysHealth, Security: L8Security }
    };
})();
