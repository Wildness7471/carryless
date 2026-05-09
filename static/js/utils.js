// Pure utility functions — exported for testing and reuse.

export function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

export function formatWeight(grams) {
    if (grams >= 1000) {
        return (grams / 1000).toFixed(1) + ' kg';
    }
    return grams + ' g';
}

export function gramsToOunces(grams) {
    return grams * 0.035274;
}

export function ouncesToGrams(ounces) {
    return Math.round(ounces / 0.035274);
}

export function formatWeightWithUnit(grams, unit) {
    if (unit === 'oz') {
        const oz = gramsToOunces(grams);
        if (oz >= 16) {
            const lbs = oz / 16;
            if (lbs >= 10) {
                return Math.round(lbs) + ' lbs';
            } else {
                return lbs.toFixed(1) + ' lbs';
            }
        }
        if (oz < 1) {
            return oz.toFixed(3) + ' oz';
        } else if (oz < 10) {
            return oz.toFixed(2) + ' oz';
        } else {
            return oz.toFixed(1) + ' oz';
        }
    }
    return grams + ' g';
}
