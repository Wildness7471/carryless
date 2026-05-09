import { renderHeader } from './components/header.js';
import { renderFooter } from './components/footer.js';

// Page module registry — maps URL patterns to dynamic imports.
const routes = [
    { pattern: /^\/dashboard$/, loader: () => import('./pages/dashboard.js') },
    { pattern: /^\/packs\/compare/, loader: () => import('./pages/pack-compare.js') },
    { pattern: /^\/packs\/new$/, loader: () => import('./pages/pack-form.js') },
    { pattern: /^\/packs\/[^/]+\/edit$/, loader: () => import('./pages/pack-form.js') },
    { pattern: /^\/packs\/[^/]+\/shares$/, loader: () => import('./pages/pack-shares.js') },
    { pattern: /^\/packs\/[^/]+$/, loader: () => import('./pages/pack-detail.js') },
    { pattern: /^\/packs$/, loader: () => import('./pages/packs.js') },
    { pattern: /^\/inventory\/items\/new$/, loader: () => import('./pages/item-form.js') },
    { pattern: /^\/inventory\/items\/[^/]+\/edit$/, loader: () => import('./pages/item-form.js') },
    { pattern: /^\/inventory$/, loader: () => import('./pages/inventory.js') },
    { pattern: /^\/categories\/new$/, loader: () => import('./pages/category-form.js') },
    { pattern: /^\/categories\/[^/]+\/edit$/, loader: () => import('./pages/category-form.js') },
    { pattern: /^\/categories$/, loader: () => import('./pages/categories.js') },
    { pattern: /^\/trips\/new$/, loader: () => import('./pages/trip-form.js') },
    { pattern: /^\/trips\/[^/]+\/edit$/, loader: () => import('./pages/trip-form.js') },
    { pattern: /^\/trips\/[^/]+$/, loader: () => import('./pages/trip-detail.js') },
    { pattern: /^\/trips$/, loader: () => import('./pages/trips.js') },
    { pattern: /^\/account$/, loader: () => import('./pages/account.js') },
];

const content = document.getElementById('app-content');
const header = document.getElementById('app-header');
const footer = document.getElementById('app-footer');

async function navigate(path) {
    showLoading();

    const route = routes.find(r => r.pattern.test(path));
    if (!route) {
        content.innerHTML = '<div class="error-page"><h2>Page not found</h2></div>';
        return;
    }

    try {
        const resp = await fetch(path, { headers: { Accept: 'application/json' } });
        if (resp.status === 302 || resp.redirected) {
            window.location.href = '/login';
            return;
        }
        if (resp.status === 401) {
            window.location.href = '/login';
            return;
        }

        const data = await resp.json();
        const mod = await route.loader();
        await mod.render(content, data);
        document.title = data.Title || 'Carryless';

        // Re-render header so nav state stays current
        if (data.User) {
            renderHeader(header, data.User, data.CSRFToken);
        }
    } catch (err) {
        content.innerHTML = `<div class="error-page"><h2>Failed to load page</h2><p>${err.message}</p></div>`;
    }
}

function showLoading() {
    content.innerHTML = '<div class="loading-spinner"><i class="fas fa-spinner fa-spin"></i> Loading...</div>';
}

// Intercept same-origin <a> clicks.
document.addEventListener('click', (e) => {
    const a = e.target.closest('a');
    if (!a) return;
    if (a.origin !== window.location.origin) return;
    if (a.target === '_blank') return;
    const path = a.pathname + a.search;
    // Let auth/public pages through to the server.
    if (/^\/(login|register|activate|p\/|t\/|terms|privacy)/.test(path)) return;
    e.preventDefault();
    window.history.pushState(null, '', path);
    navigate(path);
});

// Back/forward navigation.
window.addEventListener('popstate', () => {
    navigate(window.location.pathname + window.location.search);
});

// Boot: render header/footer then navigate to current URL.
(async () => {
    renderFooter(footer);
    navigate(window.location.pathname + window.location.search);
})();
