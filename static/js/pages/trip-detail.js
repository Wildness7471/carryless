import { escapeHtml } from '../utils.js';

export async function render(container, data) {
    const trip = data.Trip || data.trip || {};
    const csrf = data.CSRFToken || data.csrf_token || '';

    function formatDate(d) {
        if (!d) return '';
        return new Date(d).toLocaleDateString();
    }

    const checklist = (trip.checklist_items || []).map(item => `
        <li class="${item.is_checked ? 'checked' : ''}">
            <form method="POST" action="/trips/${escapeHtml(trip.id)}/checklist/${item.id}/toggle" style="display:inline">
                <input type="hidden" name="csrf_token" value="${escapeHtml(csrf)}">
                <button type="submit" class="btn-check">${item.is_checked ? '✓' : '○'}</button>
            </form>
            ${escapeHtml(item.content)}
        </li>`).join('') || '<li>No checklist items yet.</li>';

    const transport = (trip.transport_steps || []).map(step => `
        <div class="transport-step">
            <span class="transport-type">${escapeHtml(step.transport_type)}</span>
            <span>${escapeHtml(step.departure_place)} → ${escapeHtml(step.arrival_place)}</span>
        </div>`).join('') || '<p>No transport steps yet.</p>';

    container.innerHTML = `
        <div class="trip-detail-page">
            <div class="page-header">
                <h1>${escapeHtml(trip.name || '')}</h1>
                <div class="page-actions">
                    <a href="/trips/${escapeHtml(trip.id)}/edit" class="btn btn-secondary">Edit</a>
                    <a href="/trips" class="btn btn-secondary">Back to Trips</a>
                </div>
            </div>

            ${trip.location ? `<p><i class="fas fa-map-marker-alt"></i> ${escapeHtml(trip.location)}</p>` : ''}
            ${trip.start_date ? `<p>${formatDate(trip.start_date)}${trip.end_date ? ' – ' + formatDate(trip.end_date) : ''}</p>` : ''}
            ${trip.description ? `<p>${escapeHtml(trip.description)}</p>` : ''}

            <div class="trip-sections">
                <section>
                    <h2>Checklist</h2>
                    <ul class="checklist">${checklist}</ul>
                </section>

                <section>
                    <h2>Transport</h2>
                    ${transport}
                </section>
            </div>
        </div>`;
}
