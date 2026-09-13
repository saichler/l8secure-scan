/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Client-side ImageCategory cache: the ImageGroup table's "Category" column
// (§11.1) needs to resolve categoryId -> name/color, but this ORM has no
// join (verified, PRD §8) and ImageGroup carries no denormalized category
// name of its own. Loaded once when the vulnmgmt section initializes
// (categories are a small dataset, §15: "4-6 per customer"); columns read
// this cache synchronously at render time. A row rendered before the cache
// populates shows "Uncategorized" until the next natural re-render
// (pagination/filter/sort) -- a minor, self-correcting cold-load race, not
// a functional gap.
window.SecScanVuln = window.SecScanVuln || {};
SecScanVuln.categoryCache = {};

SecScanVuln.loadCategoryCache = function() {
    var query = encodeURIComponent(JSON.stringify({ text: 'select * from ImageCategory' }));
    return makeAuthenticatedRequest(Layer8DConfig.resolveEndpoint('/60/ImgCat?body=' + query))
        .then(function(r) { return r ? r.json() : null; })
        .then(function(data) {
            SecScanVuln.categoryCache = {};
            (data && data.list || []).forEach(function(c) {
                SecScanVuln.categoryCache[c.categoryId] = c;
            });
        })
        .catch(function(err) {
            console.error('SecScanVuln: failed to load category cache', err);
        });
};

SecScanVuln.getCategory = function(categoryId) {
    return categoryId ? SecScanVuln.categoryCache[categoryId] : null;
};

SecScanVuln.getCategoryName = function(categoryId) {
    var c = SecScanVuln.getCategory(categoryId);
    return c ? c.name : 'Uncategorized';
};
