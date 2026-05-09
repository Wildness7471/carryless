import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const item = data.Item || data.item || {};
    const categories = data.Categories || data.categories || [];
    const csrf = data.CSRFToken || data.csrf_token || '';
    const isEdit = !!item.id;
    const action = isEdit ? `/inventory/items/${item.id}` : '/inventory/items';

    const categoryOptions = categories.map(cat => `
        <option value="${cat.id}" ${item.category_id === cat.id ? 'selected' : ''}>
            ${escapeHtml(cat.name)}
        </option>`).join('');

    container.innerHTML = `
        <div class="item-form-page">
            <h1>${isEdit ? 'Edit Item' : 'New Item'}</h1>
            <form method="POST" action="${action}">
                <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" name="name" value="${escapeHtml(item.name || '')}" required maxlength="200">
                </div>
                <div class="form-group">
                    <label>Category</label>
                    <select name="category_id" required>
                        ${categoryOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label>Weight (grams)</label>
                    <input type="number" name="weight" value="${item.weight_grams || 0}" min="0" required>
                </div>
                <div class="form-group">
                    <label>Description</label>
                    <textarea name="description">${escapeHtml(item.description || '')}</textarea>
                </div>
                <div class="form-group">
                    <label>URL</label>
                    <input type="url" name="url" value="${escapeHtml(item.url || '')}">
                </div>
                <button type="submit" class="btn btn-primary">${isEdit ? 'Save' : 'Create Item'}</button>
                <a href="/inventory" class="btn btn-secondary">Cancel</a>
            </form>
        </div>`;
}
