import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const trips = data.Trips || data.trips || [];
    const csrf = data.CSRFToken || data.csrf_token || '';

    function formatDate(d) {
        if (!d) return '';
        return new Date(d).toLocaleDateString();
    }

    const rows = trips.map(trip => `
        <tr class="clickable-row"
            onclick="window.history.pushState(null,'','/trips/${escapeHtml(trip.id)}');window.dispatchEvent(new PopStateEvent('popstate'))">
            <td>${escapeHtml(trip.name)}</td>
            <td>${escapeHtml(trip.location || '')}</td>
            <td>${trip.start_date ? formatDate(trip.start_date) : ''}</td>
            <td>${trip.end_date ? formatDate(trip.end_date) : ''}</td>
            <td onclick="event.stopPropagation()">
                <a href="/trips/${escapeHtml(trip.id)}/edit" class="btn btn-secondary btn-sm">Edit</a>
                <form method="POST" action="/trips/${escapeHtml(trip.id)}/delete" style="display:inline"
                      onsubmit="return confirm('Delete trip ${escapeHtml(trip.name)}?')">
                    <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                    <button type="submit" class="btn btn-danger btn-sm">Delete</button>
                </form>
            </td>
        </tr>`).join('');

    container.innerHTML = `
        <div class="trips-page">
            <div class="page-header">
                <h1>Trips</h1>
                <a href="/trips/new" class="btn btn-primary">
                    <i class="fas fa-plus"></i> New Trip
                </a>
            </div>

            <table class="data-table">
                <thead>
                    <tr><th>Name</th><th>Location</th><th>Start</th><th>End</th><th></th></tr>
                </thead>
                <tbody>
                    ${rows || '<tr><td colspan="5">No trips yet. <a href="/trips/new">Plan one!</a></td></tr>'}
                </tbody>
            </table>
        </div>`;
}
