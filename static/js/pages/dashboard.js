import { escapeHtml, formatWeight } from '../utils.js';

export async function render(container, data) {
    const { Stats: s, RecentPacks: packs } = data;
    const stats = s || {};

    const recentPacksHTML = (packs && packs.length)
        ? packs.map(p => `
            <tr class="clickable-row" onclick="window.history.pushState(null,'','/packs/${escapeHtml(p.id)}');window.dispatchEvent(new PopStateEvent('popstate'))">
                <td>${escapeHtml(p.name)}</td>
                <td>${p.item_count || 0} items</td>
                <td class="weight-badge" data-weight="${p.total_weight}">${formatWeight(p.total_weight || 0)}</td>
            </tr>`).join('')
        : '<tr><td colspan="3">No packs yet. <a href="/packs/new">Create one!</a></td></tr>';

    container.innerHTML = `
        <div class="dashboard-page">
            <div class="page-header">
                <div class="header-stats">
                    <span class="header-stat"><strong>${stats.total_packs || 0}</strong> packs</span>
                    <span class="header-stat"><strong>${stats.total_items || 0}</strong> items</span>
                    <span class="header-stat"><strong>${stats.total_categories || 0}</strong> categories</span>
                    <span class="header-stat" data-weight="${stats.total_weight || 0}">
                        <strong>${formatWeight(stats.total_weight || 0)}</strong> total
                    </span>
                </div>
            </div>

            ${(stats.items_to_verify > 0) ? `
            <div class="alert alert-warning">
                <i class="fas fa-exclamation-triangle"></i>
                ${stats.items_to_verify} items need weight verification
            </div>` : ''}

            <div class="dashboard-grid">
                <section class="dashboard-section">
                    <h2>Recent Packs</h2>
                    <table class="data-table">
                        <thead><tr><th>Name</th><th>Items</th><th>Weight</th></tr></thead>
                        <tbody>${recentPacksHTML}</tbody>
                    </table>
                    <a href="/packs" class="btn btn-secondary">All packs</a>
                </section>

                <section class="dashboard-section">
                    <h2>Pack stats</h2>
                    ${stats.lightest_pack ? `
                    <div class="stat-row">
                        <span>Lightest pack</span>
                        <strong>${escapeHtml(stats.lightest_pack.name)}</strong>
                        <span class="weight-badge" data-weight="${stats.lightest_weight}">${formatWeight(stats.lightest_weight)}</span>
                    </div>` : ''}
                    ${stats.heaviest_pack ? `
                    <div class="stat-row">
                        <span>Heaviest pack</span>
                        <strong>${escapeHtml(stats.heaviest_pack.name)}</strong>
                        <span class="weight-badge" data-weight="${stats.heaviest_weight}">${formatWeight(stats.heaviest_weight)}</span>
                    </div>` : ''}
                    ${!stats.lightest_pack && !stats.heaviest_pack ? '<p>Add items to packs to see stats.</p>' : ''}
                </section>
            </div>
        </div>`;
}
