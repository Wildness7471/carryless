import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const pack = data.Pack || data.pack || {};
    const shares = data.Shares || data.shares || [];
    const csrf = data.CSRFToken || data.csrf_token || '';

    const shareRows = shares.map(s => `
        <tr>
            <td>${escapeHtml(s.username || s.email || '')}</td>
            <td>${escapeHtml(s.permission || 'view')}</td>
            <td>
                <form method="POST" action="/packs/${escapeHtml(pack.id)}/shares/${s.shared_with_user_id}/delete"
                      style="display:inline">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <button type="submit" class="btn btn-danger btn-sm">Revoke</button>
                </form>
            </td>
        </tr>`).join('') || '<tr><td colspan="3">No shares yet.</td></tr>';

    container.innerHTML = `
        <div class="pack-shares-page">
            <div class="page-header">
                <h1>Share: ${escapeHtml(pack.name || '')}</h1>
                <a href="/packs/${escapeHtml(pack.id)}" class="btn btn-secondary">Back to Pack</a>
            </div>

            <section>
                <h2>Add collaborator</h2>
                <form id="shareForm">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <div class="form-group">
                        <input type="text" name="username_or_email" placeholder="Username or email" required>
                    </div>
                    <div class="form-group">
                        <select name="permission">
                            <option value="view">View</option>
                            <option value="edit">Edit</option>
                        </select>
                    </div>
                    <button type="submit" class="btn btn-primary">Share</button>
                </form>
            </section>

            <section>
                <h2>Current shares</h2>
                <table class="data-table">
                    <thead><tr><th>User</th><th>Permission</th><th></th></tr></thead>
                    <tbody>${shareRows}</tbody>
                </table>
            </section>
        </div>`;

    document.getElementById('shareForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const form = e.target;
        const body = new URLSearchParams(new FormData(form));
        const resp = await fetch(`/packs/${pack.id}/shares`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded', Accept: 'application/json' },
            body: body.toString(),
        });
        if (resp.ok) {
            window.dispatchEvent(new PopStateEvent('popstate'));
        }
    });
}
