import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const user = data.User || data.user || {};
    const csrf = data.CSRFToken || data.csrf_token || '';

    container.innerHTML = `
        <div class="account-page">
            <h1>Account Settings</h1>

            <section class="account-section">
                <h2>Change Username</h2>
                <form method="POST" action="/account/username">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <div class="form-group">
                        <label>Username</label>
                        <input type="text" name="username" value="${escapeHtml(user.username || '')}" required minlength="2" maxlength="50">
                    </div>
                    <button type="submit" class="btn btn-primary">Save Username</button>
                </form>
            </section>

            <section class="account-section">
                <h2>Change Password</h2>
                <form method="POST" action="/account/password">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <div class="form-group">
                        <label>Current Password</label>
                        <input type="password" name="current_password" required>
                    </div>
                    <div class="form-group">
                        <label>New Password</label>
                        <input type="password" name="new_password" required minlength="8">
                    </div>
                    <div class="form-group">
                        <label>Confirm New Password</label>
                        <input type="password" name="confirm_password" required>
                    </div>
                    <button type="submit" class="btn btn-primary">Change Password</button>
                </form>
            </section>

            <section class="account-section">
                <h2>Currency</h2>
                <form method="POST" action="/account/currency">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <div class="form-group">
                        <label>Currency Symbol</label>
                        <select name="currency">
                            ${['$','€','¥','£','₹','₩','¢','R'].map(c =>
                                `<option value="${c}" ${(user.currency||'$') === c ? 'selected' : ''}>${c}</option>`
                            ).join('')}
                        </select>
                    </div>
                    <button type="submit" class="btn btn-primary">Save Currency</button>
                </form>
            </section>
        </div>`;
}
