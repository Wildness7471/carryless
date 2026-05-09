import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const categories = data.Categories || data.categories || [];
    const csrf = data.CSRFToken || data.csrf_token || '';

    const rows = categories.map(cat => `
        <tr>
            <td>${escapeHtml(cat.name)}</td>
            <td>${cat.item_count || 0} items</td>
            <td>
                <a href="/categories/${cat.id}/edit" class="btn btn-secondary btn-sm">Edit</a>
                <form method="POST" action="/categories/${cat.id}/delete" style="display:inline"
                      onsubmit="return confirm('Delete category ${escapeHtml(cat.name)}?')">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <button type="submit" class="btn btn-danger btn-sm">Delete</button>
                </form>
            </td>
        </tr>`).join('');

    container.innerHTML = `
        <div class="categories-page">
            <div class="page-header">
                <h1>Categories</h1>
                <a href="/categories/new" class="btn btn-primary">
                    <i class="fas fa-plus"></i> New Category
                </a>
            </div>

            <table class="data-table">
                <thead>
                    <tr><th>Name</th><th>Items</th><th></th></tr>
                </thead>
                <tbody>
                    ${rows || '<tr><td colspan="3">No categories yet. <a href="/categories/new">Create one!</a></td></tr>'}
                </tbody>
            </table>
        </div>`;
}
