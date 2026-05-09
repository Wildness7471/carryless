import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const pack = data.Pack || data.pack || {};
    const csrf = data.CSRFToken || data.csrf_token || '';
    const isEdit = !!pack.id;
    const action = isEdit ? `/packs/${escapeHtml(pack.id)}` : '/packs';

    container.innerHTML = `
        <div class="pack-form-page">
            <h1>${isEdit ? 'Edit Pack' : 'New Pack'}</h1>
            <form method="POST" action="${action}">
                <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" name="name" value="${escapeHtml(pack.name || '')}" required maxlength="200">
                </div>
                <div class="form-group">
                    <label>Note</label>
                    <textarea name="note">${escapeHtml(pack.note || '')}</textarea>
                </div>
                <button type="submit" class="btn btn-primary">${isEdit ? 'Save' : 'Create Pack'}</button>
                <a href="/packs" class="btn btn-secondary">Cancel</a>
            </form>
        </div>`;
}
