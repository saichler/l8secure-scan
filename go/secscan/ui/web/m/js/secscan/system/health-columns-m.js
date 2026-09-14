/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/

// Mobile L8Health columns + transform (no desktop equivalent -- L8Health is
// rendered by l8ui/sys/health/l8health.js's own desktop-only widget code).
// Field shape (alias, stats.{rxMsgCount,txMsgCount,memoryUsage,cpuUsage,
// lastMsgTime}, startTime) is generic-framework (l8types L8Health), the
// same real shape every mobile Layer 8 project's own hand-authored health
// columns use (verified against ../l8erp/go/erp/ui/web/m/js/sys/sys-columns.js).
window.MobileSysHealth = window.MobileSysHealth || {};

(function() {
    'use strict';

    var col = window.Layer8ColumnFactory;

    function formatBytes(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        var sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        var i = Math.floor(Math.log(bytes) / Math.log(1024));
        if (i === 0) return bytes + ' B';
        return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + sizes[i];
    }

    function formatUptime(startTime) {
        if (!startTime || startTime === 0 || startTime === '0') return '00:00:00';
        var startMs = typeof startTime === 'string' ? parseInt(startTime, 10) : startTime;
        var s = Math.floor((Date.now() - startMs) / 1000);
        if (s < 0) return '00:00:00';
        return String(Math.floor(s / 3600)).padStart(2, '0') + ':' +
               String(Math.floor((s % 3600) / 60)).padStart(2, '0') + ':' +
               String(s % 60).padStart(2, '0');
    }

    function formatLastPulse(lastMsgTime) {
        if (!lastMsgTime || lastMsgTime === 0 || lastMsgTime === '0') return '00:00:00';
        var ms = typeof lastMsgTime === 'string' ? parseInt(lastMsgTime, 10) : lastMsgTime;
        var s = Math.floor((Date.now() - ms) / 1000);
        if (s < 0) return '00:00:00';
        return String(Math.floor(s / 3600)).padStart(2, '0') + ':' +
               String(Math.floor((s % 3600) / 60)).padStart(2, '0') + ':' +
               String(s % 60).padStart(2, '0');
    }

    MobileSysHealth.transformData = function(item) {
        if (!item.stats) return null;
        return {
            service: item.alias || 'Unknown',
            rx: (item.stats.rxMsgCount || 0).toLocaleString(),
            tx: (item.stats.txMsgCount || 0).toLocaleString(),
            memory: formatBytes(item.stats.memoryUsage || 0),
            cpuPercent: (item.stats.cpuUsage || 0).toFixed(2) + '%',
            upTime: formatUptime(item.startTime),
            lastPulse: formatLastPulse(item.stats.lastMsgTime)
        };
    };

    MobileSysHealth.columns = {
        L8Health: [
            Object.assign({}, col.custom('service', 'Service', null, { sortKey: 'alias', filterKey: 'alias' })[0], { primary: true }),
            ...col.custom('cpuPercent', 'CPU %', null, { sortKey: 'stats.cpuUsage', filterKey: 'stats.cpuUsage' }),
            ...col.custom('memory', 'Memory', null, { sortKey: 'stats.memoryUsage', filterKey: 'stats.memoryUsage' }),
            ...col.custom('rx', 'RX', null, { sortKey: 'stats.rxMsgCount', filterKey: 'stats.rxMsgCount' }),
            ...col.custom('tx', 'TX', null, { sortKey: 'stats.txMsgCount', filterKey: 'stats.txMsgCount' }),
            ...col.custom('upTime', 'Up Time', null, { sortKey: 'startTime', filterKey: 'startTime' }),
            ...col.custom('lastPulse', 'Last Pulse', null, { sortKey: 'stats.lastMsgTime', filterKey: 'stats.lastMsgTime' })
        ]
    };
})();
