import { escapeHtml, formatWeight } from '../utils.js';

export async function render(container, data) {
    const packs = data.Packs || data.packs || [];

    if (packs.length === 0) {
        container.innerHTML = '<p>Select packs from the packs list to compare them.</p>';
        return;
    }

    const headers = packs.map(p => `<th>${escapeHtml(p.name)}</th>`).join('');

    // Collect all item names across all packs
    const allItemNames = new Set();
    packs.forEach(p => {
        (p.items || []).forEach(pi => {
            const name = pi.item ? pi.item.name : '';
            if (name) allItemNames.add(name);
        });
    });

    const itemRows = Array.from(allItemNames).sort().map(name => {
        const cells = packs.map(p => {
            const pi = (p.items || []).find(i => i.item && i.item.name === name);
            return pi
                ? `<td><span class="weight-badge" data-weight="${pi.item.weight_grams}">${formatWeight(pi.item.weight_grams)}</span></td>`
                : '<td>—</td>';
        }).join('');
        return `<tr><td>${escapeHtml(name)}</td>${cells}</tr>`;
    }).join('');

    const totalRow = `<tr>
        <td><strong>Total</strong></td>
        ${packs.map(p => {
            const total = (p.items || []).reduce((s, pi) => s + (pi.item ? (pi.item.weight_grams || 0) : 0), 0);
            return `<td><strong class="weight-badge" data-weight="${total}">${formatWeight(total)}</strong></td>`;
        }).join('')}
    </tr>`;

    container.innerHTML = `
        <div class="compare-page">
            <div class="page-header">
                <h1>Compare Packs</h1>
                <a href="/packs" class="btn btn-secondary">Back to Packs</a>
            </div>
            <div class="compare-table-wrapper">
                <table class="data-table compare-table">
                    <thead><tr><th>Item</th>${headers}</tr></thead>
                    <tbody>${itemRows}${totalRow}</tbody>
                </table>
            </div>
        </div>`;
}
