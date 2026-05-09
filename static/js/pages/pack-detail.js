import { escapeHtml, formatWeight } from '../utils.js';

function categorySection(categoryName, items, packID, csrf) {
    const rows = items.map(pi => `
        <tr>
            <td>
                ${pi.is_worn ? '<i class="fas fa-user" title="Worn"></i> ' : ''}
                ${escapeHtml(pi.item ? pi.item.name : pi.name || '')}
            </td>
            <td>${escapeHtml(pi.item ? (pi.item.description || '') : '')}</td>
            <td>
                <span class="weight-badge" data-weight="${pi.item ? pi.item.weight_grams : 0}">
                    ${formatWeight(pi.item ? (pi.item.weight_grams || 0) : 0)}
                </span>
            </td>
            <td>
                <button class="btn btn-danger btn-sm" onclick="removeItem('${escapeHtml(packID)}', ${pi.item ? pi.item.id : pi.item_id}, '${escapeHtml(csrf)}')">
                    <i class="fas fa-times"></i>
                </button>
            </td>
        </tr>`).join('');

    return `
        <div class="category-section">
            <h3 class="category-heading">${escapeHtml(categoryName)}</h3>
            <table class="data-table">
                <tbody>${rows}</tbody>
            </table>
        </div>`;
}

export async function render(container, data) {
    const pack = data.Pack || data.pack || {};
    const items = pack.items || [];
    const csrf = data.CSRFToken || data.csrf_token || '';
    const packID = pack.id || '';

    // Group items by category
    const groups = {};
    for (const pi of items) {
        const cat = (pi.item && pi.item.category && pi.item.category.name) || 'Uncategorized';
        (groups[cat] = groups[cat] || []).push(pi);
    }

    const categorySections = Object.keys(groups).sort()
        .map(cat => categorySection(cat, groups[cat], packID, csrf))
        .join('');

    const totalWeight = items.reduce((sum, pi) => sum + (pi.item ? (pi.item.weight_grams || 0) : 0), 0);
    const wornWeight = items.filter(pi => pi.is_worn)
        .reduce((sum, pi) => sum + (pi.item ? (pi.item.weight_grams || 0) : 0), 0);

    container.innerHTML = `
        <div class="pack-detail-page">
            <div class="page-header">
                <h1>${escapeHtml(pack.name || '')}</h1>
                <div class="page-actions">
                    <a href="/packs/${escapeHtml(packID)}/edit" class="btn btn-secondary">Edit</a>
                    <a href="/packs" class="btn btn-secondary">Back to Packs</a>
                </div>
            </div>

            <div class="pack-stats">
                <span class="weight-badge" data-weight="${totalWeight}">Total: ${formatWeight(totalWeight)}</span>
                ${wornWeight > 0 ? `<span class="weight-badge">Worn: ${formatWeight(wornWeight)}</span>` : ''}
                <span>${items.length} items</span>
            </div>

            ${pack.note ? `<p class="pack-note">${escapeHtml(pack.note)}</p>` : ''}

            <div class="pack-items">
                ${categorySections || '<p>No items in this pack. <a href="/inventory">Add items</a> to your inventory first.</p>'}
            </div>
        </div>`;

    // Expose removeItem globally for inline handlers
    window.removeItem = async (packId, itemId, csrfToken) => {
        if (!confirm('Remove item from pack?')) return;
        const resp = await fetch(`/packs/${packId}/items/${itemId}`, {
            method: 'DELETE',
            headers: { 'X-CSRF-Token': csrfToken, Accept: 'application/json' },
        });
        if (resp.ok) {
            window.dispatchEvent(new PopStateEvent('popstate'));
        }
    };
}
