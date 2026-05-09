import { escapeHtml, formatWeight } from '../utils.js';

export async function render(container, data) {
    const items = data.Items || data.items || [];
    const csrf = data.CSRFToken || data.csrf_token || '';

    const rows = items.map(item => `
        <tr>
            <td>${escapeHtml(item.name)}</td>
            <td>${escapeHtml(item.category ? item.category.name : '')}</td>
            <td><span class="weight-badge" data-weight="${item.weight_grams}">${formatWeight(item.weight_grams || 0)}</span></td>
            <td>${escapeHtml(item.description || '')}</td>
            <td onclick="event.stopPropagation()">
                <a href="/inventory/items/${item.id}/edit" class="btn btn-secondary btn-sm">Edit</a>
                <form method="POST" action="/inventory/items/${item.id}/delete" style="display:inline"
                      onsubmit="return confirm('Delete item ${escapeHtml(item.name)}?')">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <button type="submit" class="btn btn-danger btn-sm">Delete</button>
                </form>
            </td>
        </tr>`).join('');

    container.innerHTML = `
        <div class="inventory-page">
            <div class="page-header">
                <h1>Inventory</h1>
                <div>
                    <a href="/inventory/items/new" class="btn btn-primary">
                        <i class="fas fa-plus"></i> New Item
                    </a>
                    <a href="/inventory/export" class="btn btn-secondary">Export CSV</a>
                </div>
            </div>

            <table class="data-table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Category</th>
                        <th>Weight</th>
                        <th>Description</th>
                        <th></th>
                    </tr>
                </thead>
                <tbody>
                    ${rows || '<tr><td colspan="5">No items yet. <a href="/inventory/items/new">Add your first item!</a></td></tr>'}
                </tbody>
            </table>
        </div>`;
}
