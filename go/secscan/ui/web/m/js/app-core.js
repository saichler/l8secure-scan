/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
*/
/**
 * Mobile App Core - Navigation and initialization (PRD §11.7)
 * Adapted from ../l8erp/go/erp/ui/web/m/js/app-core.js (the real mobile
 * app-shell precedent): trimmed to this project's own needs -- a single
 * SECTIONS entry ('dashboard', the Layer8MNav mount point every
 * module/system view renders into, per that config's dashboard.html), no
 * currency/exchange-rate caches or /permissions fetch (secscan has no such
 * concepts), and no Layer8DModuleFilter wiring (ModconfigFailureNoLogout --
 * this project has no ModConfig service, matching desktop's js/app.js).
 */
(function() {
    'use strict';

    const SECTIONS = {
        'dashboard': 'sections/dashboard.html'
    };

    let currentSection = 'dashboard';
    let sectionCache = {};

    window.showErrorAndLogout = function(message, detail) {
        if (typeof Layer8MAuth !== 'undefined') {
            Layer8MAuth.showErrorAndLogout(message, detail);
        } else {
            alert(message + (detail ? '\n\n' + detail : ''));
            window.location.href = '/l8ui/login/';
        }
    };

    window.MobileApp = {
        async init() {
            if (!Layer8MAuth.requireAuth()) return;

            await Layer8MConfig.load();

            // Connect real-time WebSocket for live data updates -- reuses
            // the same desktop Layer8DWebSocket already loaded here, never
            // previously initialized on mobile
            // (plans/scanjob-live-progress.md Phase 5-m).
            if (typeof Layer8DWebSocket !== 'undefined') {
                Layer8DWebSocket.init();
            }

            this.updateUserInfo();

            // No server-side ModConfig service in this project
            // (ModconfigFailureNoLogout) -- mark any shared filter code as
            // loaded so it never blocks, matching desktop's js/app.js.
            if (typeof Layer8DModuleFilter !== 'undefined') {
                Layer8DModuleFilter._loaded = true;
            }

            this.initSidebar();

            document.getElementById('refresh-btn')?.addEventListener('click', () => {
                this.loadSection(currentSection, true);
            });

            const hash = window.location.hash.slice(1);
            const section = SECTIONS[hash] ? hash : 'dashboard';
            await this.loadSection(section);

            window.addEventListener('hashchange', () => {
                const newSection = window.location.hash.slice(1);
                if (SECTIONS[newSection] && newSection !== currentSection) {
                    this.loadSection(newSection);
                }
            });
        },

        updateUserInfo() {
            const username = Layer8MAuth.getUsername();
            const initial = username.charAt(0).toUpperCase();
            document.getElementById('user-name').textContent = username;
            document.getElementById('user-avatar').textContent = initial;
        },

        initSidebar() {
            const menuToggle = document.getElementById('menu-toggle');
            document.getElementById('sidebar-overlay')?.addEventListener('click', () => this.closeSidebar());
            menuToggle?.addEventListener('click', () => this.openSidebar());

            document.querySelectorAll('.sidebar-item[data-section]').forEach(item => {
                item.addEventListener('click', async (e) => {
                    e.preventDefault();
                    const section = item.dataset.section;
                    const module = item.dataset.module;
                    this.closeSidebar();
                    await this.loadSection(section);
                    if (module && window.Layer8MNav) {
                        Layer8MNav.navigateToModule(module);
                    }
                });
            });
        },

        openSidebar() {
            document.getElementById('sidebar')?.classList.add('open');
            document.getElementById('sidebar-overlay')?.classList.add('visible');
            document.body.style.overflow = 'hidden';
        },

        closeSidebar() {
            document.getElementById('sidebar')?.classList.remove('open');
            document.getElementById('sidebar-overlay')?.classList.remove('visible');
            document.body.style.overflow = '';
        },

        async loadSection(section, forceReload = false) {
            const sectionUrl = SECTIONS[section];
            if (!sectionUrl) {
                console.error('Unknown section:', section);
                return;
            }

            this.updateNavState(section);

            const contentArea = document.getElementById('content-area');
            if (!contentArea) return;

            contentArea.style.opacity = '0.5';

            try {
                if (!forceReload && sectionCache[section]) {
                    contentArea.innerHTML = sectionCache[section];
                } else {
                    const response = await fetch(sectionUrl + '?t=' + Date.now());
                    if (!response.ok) throw new Error('Failed to load section');
                    const html = await response.text();
                    sectionCache[section] = html;
                    contentArea.innerHTML = html;
                }

                this.executeScripts(contentArea);
                this.initSection(section);

                currentSection = section;
                window.location.hash = section;
                contentArea.scrollTop = 0;
            } catch (error) {
                console.error('Error loading section:', error);
                contentArea.innerHTML = `
                    <div class="nav-empty-state">
                        <div class="nav-empty-state-icon">&#x26A0;&#xFE0F;</div>
                        <h3>Failed to load</h3>
                        <p>Please try again</p>
                        <button class="mobile-popup-btn mobile-popup-btn-save" onclick="MobileApp.loadSection('${section}', true)">Retry</button>
                    </div>
                `;
            }

            contentArea.style.opacity = '1';
        },

        updateNavState(section) {
            document.querySelectorAll('.sidebar-item').forEach(item => {
                item.classList.remove('active');
                if (item.dataset.section === section && !item.dataset.module) {
                    item.classList.add('active');
                }
            });
        },

        executeScripts(container) {
            const scripts = container.querySelectorAll('script');
            scripts.forEach(oldScript => {
                const newScript = document.createElement('script');
                Array.from(oldScript.attributes).forEach(attr => {
                    newScript.setAttribute(attr.name, attr.value);
                });
                newScript.textContent = oldScript.textContent;
                oldScript.parentNode.replaceChild(newScript, oldScript);
            });
        },

        initSection(section) {
            const initFunctions = { 'dashboard': 'initMobileDashboard' };
            const initFn = initFunctions[section];
            if (initFn && typeof window[initFn] === 'function') {
                window[initFn]();
            }
        },

        getCurrentSection() {
            return currentSection;
        },

        logout() {
            Layer8MAuth.logout();
        }
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => MobileApp.init());
    } else {
        MobileApp.init();
    }

})();
