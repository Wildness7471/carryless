import { escapeHtml, formatWeight } from '../utils.js';

function packRow(p, csrf) {
    return `
        <tr class="clickable-row${p.is_locked ? ' locked-pack' : ''}"
            data-href="/packs/${escapeHtml(p.id)}"
            onclick="window.history.pushState(null,'','/packs/${escapeHtml(p.id)}');window.dispatchEvent(new PopStateEvent('popstate'))">
            <td>
                <div class="pack-name-cell">
                    ${p.is_locked ? '<i class="fas fa-lock" title="Locked"></i> ' : ''}
                    ${escapeHtml(p.name)}
                </div>
            </td>
            <td>${p.item_count || 0} items</td>
            <td>
                <span class="weight-badge" data-weight="${p.base_weight}">${formatWeight(p.base_weight || 0)}</span>
            </td>
            <td onclick="event.stopPropagation()">
                <form method="POST" action="/packs/${escapeHtml(p.id)}/delete"
                      onsubmit="return confirm('Delete pack ${escapeHtml(p.name)}?')">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <button type="submit" class="btn btn-danger btn-sm">Delete</button>
                </form>
            </td>
        </tr>`;
}

export async function render(container, data) {
    const packs = data.Packs || data.packs || [];
    const shared = data.SharedPacks || data.shared_packs || [];
    const csrf = data.CSRFToken || data.csrf_token || '';

    const packsHTML = packs.length
        ? packs.map(p => packRow(p, csrf)).join('')
        : '<tr><td colspan="4">No packs yet. <a href="/packs/new">Create one!</a></td></tr>';

    const sharedHTML = shared.length
        ? shared.map(p => `
            <tr class="clickable-row"
                onclick="window.history.pushState(null,'','/packs/${escapeHtml(p.id)}');window.dispatchEvent(new PopStateEvent('popstate'))">
                <td>${escapeHtml(p.name)}</td>
                <td>${p.item_count || 0} items</td>
                <td><span class="weight-badge" data-weight="${p.base_weight}">${formatWeight(p.base_weight || 0)}</span></td>
                <td>${escapeHtml(p.user_permission || 'view')}</td>
            </tr>`).join('')
        : '';

    container.innerHTML = `
        <div class="packs-page">
            <div class="page-header">
                <h1>Packs</h1>
                <a href="/packs/new" class="btn btn-primary">
                    <i class="fas fa-plus"></i> New Pack
                </a>
            </div>

            <div class="packs-table">
                <table class="data-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Items</th>
                            <th>Base Weight</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>${packsHTML}</tbody>
                </table>
            </div>

            ${shared.length ? `
            <h2>Shared with me</h2>
            <div class="packs-table">
                <table class="data-table">
                    <thead><tr><th>Name</th><th>Items</th><th>Weight</th><th>Permission</th></tr></thead>
                    <tbody>${sharedHTML}</tbody>
                </table>
            </div>` : ''}
        </div>`;
}
