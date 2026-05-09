import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const category = data.Category || data.category || {};
    const csrf = data.CSRFToken || data.csrf_token || '';
    const isEdit = !!category.id;
    const action = isEdit ? `/categories/${category.id}` : '/categories';

    container.innerHTML = `
        <div class="category-form-page">
            <h1>${isEdit ? 'Edit Category' : 'New Category'}</h1>
            <form method="POST" action="${action}">
                <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" name="name" value="${escapeHtml(category.name || '')}" required maxlength="100">
                </div>
                <button type="submit" class="btn btn-primary">${isEdit ? 'Save' : 'Create Category'}</button>
                <a href="/categories" class="btn btn-secondary">Cancel</a>
            </form>
        </div>`;
}
