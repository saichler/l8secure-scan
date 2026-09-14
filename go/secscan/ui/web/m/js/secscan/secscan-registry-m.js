/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile module registry (PRD §11.7): Layer8MNavData.getServiceColumns/
// getServiceFormDef/getServiceTransformData only check a hardcoded whitelist
// of known registry globals in l8ui/m/js/layer8m-nav-data.js -- 'MobileSecScan'
// was added there (l8ui commit ea46cc8) alongside every other real project's
// own entry (MobileHCM, MobileAlarms, etc.), the established convention.
Layer8MModuleRegistry.create('MobileSecScan', {
    'Vulnerability Management': MobileSecScanVuln,
    'Administration': MobileSecScanAdmin
});
