/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile-flavored loader for the SAME SecScanVuln.categoryCache the
// decorated ImageGroup "Category" column (groups-columns-m.js, via the
// desktop groups-columns.js render callback) reads via SecScanVuln.getCategory
// -- desktop's own loader (../../../secscan/vulnmgmt/category-cache.js) uses
// makeAuthenticatedRequest/Layer8DConfig, which don't exist on this page, so
// this reimplements just the fetch using Layer8MAuth/Layer8MConfig while
// writing to the exact same shared cache object/functions (no duplicate
// cache, no separate getCategory implementation).
window.SecScanVuln = window.SecScanVuln || {};
SecScanVuln.categoryCache = SecScanVuln.categoryCache || {};

SecScanVuln.loadCategoryCache = function() {
    var query = encodeURIComponent(JSON.stringify({ text: 'select * from ImageCategory' }));
    return Layer8MAuth.get(Layer8MConfig.resolveEndpoint('/60/ImgCat?body=' + query))
        .then(function(data) {
            SecScanVuln.categoryCache = {};
            (data && data.list || []).forEach(function(c) {
                SecScanVuln.categoryCache[c.categoryId] = c;
            });
        })
        .catch(function(err) {
            console.error('SecScanVuln (mobile): failed to load category cache', err);
        });
};

SecScanVuln.getCategory = SecScanVuln.getCategory || function(categoryId) {
    return categoryId ? SecScanVuln.categoryCache[categoryId] : null;
};

SecScanVuln.getCategoryName = SecScanVuln.getCategoryName || function(categoryId) {
    var c = SecScanVuln.getCategory(categoryId);
    return c ? c.name : 'Uncategorized';
};
