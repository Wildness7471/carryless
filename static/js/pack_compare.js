// pack_compare.js — drag-and-drop + move-all between packs on the compare page.
// Depends on apiRequest() from app.js and the CSRF global set by the template.

'use strict';

let draggedRow = null;
let draggedPackID = null;
let draggedItemID = null;

// ── Drag-and-drop: items ──────────────────────────────────────────────────────

function handleItemDragStart(event) {
    const row = event.currentTarget;
    draggedRow = row;
    draggedPackID = row.dataset.packId;
    draggedItemID = row.dataset.itemId;
    row.classList.add('dragging');
    event.dataTransfer.effectAllowed = 'move';
}

async function handleItemDrop(event, targetPackID) {
    event.preventDefault();
    if (!draggedRow || draggedPackID === targetPackID) return;

    await moveItem(draggedPackID, draggedItemID, targetPackID, draggedRow);

    draggedRow = null;
    draggedPackID = null;
    draggedItemID = null;
}

// ── Drag-and-drop: columns ────────────────────────────────────────────────────

let draggedColIndex = null;

document.addEventListener('dragstart', (e) => {
    const col = e.target.closest('.compare-col');
    if (!col || e.target.closest('.compare-item-row')) return;
    draggedColIndex = [...col.parentElement.children].indexOf(col);
    col.style.opacity = '.5';
});

document.addEventListener('dragend', (e) => {
    const col = e.target.closest('.compare-col');
    if (col) col.style.opacity = '';
    draggedColIndex = null;
    document.querySelectorAll('.compare-col').forEach(c => c.classList.remove('drag-over'));
});

function handleColumnDrop(event, targetIndex) {
    event.preventDefault();
    if (draggedColIndex === null || draggedColIndex === targetIndex) return;
    const grid = document.getElementById('compare-grid');
    const cols = [...grid.children];
    const from = cols[draggedColIndex];
    const to = cols[targetIndex];
    if (!from || !to) return;
    // Swap DOM positions
    const fromNext = from.nextSibling;
    const toNext = to.nextSibling;
    grid.insertBefore(from, toNext);
    grid.insertBefore(to, fromNext);
    draggedColIndex = null;
}

document.addEventListener('dragover', (e) => {
    const col = e.target.closest('.compare-col');
    document.querySelectorAll('.compare-col').forEach(c => c.classList.remove('drag-over'));
    if (col) col.classList.add('drag-over');
});

// ── Move-all buttons ──────────────────────────────────────────────────────────

document.addEventListener('click', async (e) => {
    const btn = e.target.closest('.move-btn');
    if (!btn) return;

    const cols = [...document.querySelectorAll('.compare-col')];
    const fromIdx = parseInt(btn.dataset.from, 10);
    const toIdx = parseInt(btn.dataset.to, 10);
    const fromCol = cols[fromIdx];
    const toCol = cols[toIdx];
    if (!fromCol || !toCol) return;

    const fromPackID = fromCol.dataset.packId;
    const toPackID = toCol.dataset.packId;

    const rows = [...fromCol.querySelectorAll('.compare-item-row')];
    if (rows.length === 0) return;

    btn.disabled = true;
    btn.textContent = 'Moving…';

    for (const row of rows) {
        await moveItem(fromPackID, row.dataset.itemId, toPackID, row);
    }

    btn.disabled = false;
    btn.innerHTML = `Move all &rarr; ${escapeHtml(toCol.querySelector('h2').textContent.trim())}`;
    updateColumnStats();
});

// ── Core move logic ───────────────────────────────────────────────────────────

async function moveItem(fromPackID, itemID, toPackID, row) {
    // 1. Add to target pack
    const addRes = await apiRequest(`/packs/${toPackID}/items`, {
        method: 'POST',
        body: new URLSearchParams({ item_id: itemID, csrf_token: CSRF }),
    });
    if (!addRes.ok) {
        const d = await addRes.json().catch(() => ({}));
        console.warn('Failed to add item to target pack:', d.error || addRes.status);
        row.classList.remove('dragging');
        return;
    }
    // 2. Remove from source pack — need the pack_item id (row dataset)
    const packItemID = row.dataset.packItemId;
    const delRes = await apiRequest(`/packs/${fromPackID}/items/${packItemID}`, {
        method: 'DELETE',
        body: new URLSearchParams({ csrf_token: CSRF }),
    });

    if (delRes.ok) {
        // Move the DOM row to the target column
        const targetCol = document.querySelector(`.compare-col[data-pack-id="${toPackID}"]`);
        const targetItems = targetCol ? targetCol.querySelector('.compare-items') : null;
        if (targetItems) {
            row.dataset.packId = toPackID;
            targetItems.appendChild(row);
        } else {
            row.remove();
        }
    } else {
        const d = await delRes.json().catch(() => ({}));
        console.warn('Failed to remove item from source pack:', d.error || delRes.status);
    }

    row.classList.remove('dragging');
    updateColumnStats();
}

// ── Weight display helpers ────────────────────────────────────────────────────

function updateColumnStats() {
    document.querySelectorAll('.compare-col').forEach(col => {
        const rows = col.querySelectorAll('.compare-item-row');
        let total = 0;
        let count = 0;
        rows.forEach(r => {
            const weightEl = r.querySelector('.item-weight');
            if (weightEl) {
                const g = parseInt(weightEl.dataset.weight || '0', 10);
                const itemCount = parseInt(r.querySelector('.item-count')?.textContent?.replace('×','') || '1', 10);
                total += g * itemCount;
                count += itemCount;
            }
        });
        const statTotal = col.querySelector('.stat-total');
        const statBase = col.querySelector('.stat-base');
        if (statTotal) {
            statTotal.dataset.weight = total;
            statTotal.textContent = `${total}g total`;
        }
        if (statBase) {
            statBase.dataset.weight = total;
            statBase.textContent = `${total}g base`;
        }
        const statCount = col.querySelector('.stat-chip:last-child');
        if (statCount && !statCount.classList.contains('stat-worn')) {
            statCount.textContent = `${count} items`;
        }
    });

    // Re-run unit conversion from app.js if available
    if (typeof window.updateWeightDisplays === 'function') {
        window.updateWeightDisplays();
    }
}
