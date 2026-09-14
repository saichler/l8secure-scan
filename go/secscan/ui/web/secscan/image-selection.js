/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Cross-page/cross-popup image-ref selection for scanning. Checkboxes in
// Group Detail's ref table (group-detail.js) add/remove imageRefIds here;
// the Dashboard's "Scan Images" button (dashboard-page.js) reads the
// current set and triggers the scan -- selection must survive closing one
// Group Detail popup and opening another, and navigating to the Dashboard
// section, all without a page reload (a plain SPA in-memory Map is enough
// since this app never reloads the page between those actions).
window.SecScanImageSelection = (function() {
    'use strict';

    const selected = new Map(); // imageRefId -> display label
    const listeners = [];

    function notify() {
        listeners.forEach(function(fn) { fn(count()); });
    }

    function add(id, label) {
        selected.set(id, label || id);
        notify();
    }

    function remove(id) {
        selected.delete(id);
        notify();
    }

    function has(id) {
        return selected.has(id);
    }

    function toggle(id, label) {
        if (has(id)) {
            remove(id);
        } else {
            add(id, label);
        }
    }

    function clear() {
        selected.clear();
        notify();
    }

    function getIds() {
        return Array.from(selected.keys());
    }

    function getEntries() {
        return Array.from(selected.entries()).map(function(e) { return { id: e[0], label: e[1] }; });
    }

    function count() {
        return selected.size;
    }

    // onChange(fn): fn(newCount) called after every add/remove/toggle/clear
    // -- lets both the Group Detail popup (selection-count hint) and the
    // Dashboard (Scan Images button label/disabled state) stay in sync
    // regardless of which one made the change.
    function onChange(fn) {
        listeners.push(fn);
    }

    return { add: add, remove: remove, has: has, toggle: toggle, clear: clear, getIds: getIds, getEntries: getEntries, count: count, onChange: onChange };
})();
