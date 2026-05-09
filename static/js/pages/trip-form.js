import { escapeHtml } from '../utils.js';

function dateStr(d) {
    if (!d) return '';
    return new Date(d).toISOString().split('T')[0];
}

export async function render(container, data) {
    const trip = data.Trip || data.trip || {};
    const csrf = data.CSRFToken || data.csrf_token || '';
    const isEdit = !!trip.id;
    const action = isEdit ? `/trips/${escapeHtml(trip.id)}` : '/trips';

    container.innerHTML = `
        <div class="trip-form-page">
            <h1>${isEdit ? 'Edit Trip' : 'New Trip'}</h1>
            <form method="POST" action="${action}">
                <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" name="name" value="${escapeHtml(trip.name || '')}" required maxlength="200">
                </div>
                <div class="form-group">
                    <label>Location</label>
                    <input type="text" name="location" value="${escapeHtml(trip.location || '')}">
                </div>
                <div class="form-group">
                    <label>Description</label>
                    <textarea name="description">${escapeHtml(trip.description || '')}</textarea>
                </div>
                <div class="form-group">
                    <label>Start Date</label>
                    <input type="date" name="start_date" value="${dateStr(trip.start_date)}">
                </div>
                <div class="form-group">
                    <label>End Date</label>
                    <input type="date" name="end_date" value="${dateStr(trip.end_date)}">
                </div>
                <button type="submit" class="btn btn-primary">${isEdit ? 'Save' : 'Create Trip'}</button>
                <a href="/trips" class="btn btn-secondary">Cancel</a>
            </form>
        </div>`;
}
