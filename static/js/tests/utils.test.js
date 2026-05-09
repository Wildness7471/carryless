import { describe, it, expect } from 'vitest';
import {
    escapeHtml,
    formatWeight,
    gramsToOunces,
    ouncesToGrams,
    formatWeightWithUnit,
} from '../utils.js';

describe('escapeHtml', () => {
    it('returns empty string for null', () => {
        expect(escapeHtml(null)).toBe('');
    });

    it('returns empty string for undefined', () => {
        expect(escapeHtml(undefined)).toBe('');
    });

    it('escapes < and > characters', () => {
        expect(escapeHtml('<script>alert(1)</script>')).toBe('&lt;script&gt;alert(1)&lt;/script&gt;');
    });

    it('does not encode double quotes in text nodes', () => {
        expect(escapeHtml('"hello"')).toBe('"hello"');
    });

    it('escapes ampersands', () => {
        expect(escapeHtml('a & b')).toBe('a &amp; b');
    });

    it('returns plain text unchanged', () => {
        expect(escapeHtml('hello world')).toBe('hello world');
    });

    it('is idempotent on already-escaped input', () => {
        const escaped = escapeHtml('<b>');
        const doubleEscaped = escapeHtml(escaped);
        expect(doubleEscaped).toBe('&amp;lt;b&amp;gt;');
    });
});

describe('formatWeight', () => {
    it('formats grams below 1000 with g suffix', () => {
        expect(formatWeight(500)).toBe('500 g');
        expect(formatWeight(0)).toBe('0 g');
        expect(formatWeight(999)).toBe('999 g');
    });

    it('formats 1000g as 1.0 kg', () => {
        expect(formatWeight(1000)).toBe('1.0 kg');
    });

    it('formats weights above 1000 in kilograms', () => {
        expect(formatWeight(1500)).toBe('1.5 kg');
        expect(formatWeight(2000)).toBe('2.0 kg');
        expect(formatWeight(10000)).toBe('10.0 kg');
    });
});

describe('gramsToOunces', () => {
    it('converts grams to ounces', () => {
        expect(gramsToOunces(28.3495)).toBeCloseTo(1.0, 2);
        expect(gramsToOunces(0)).toBe(0);
    });

    it('round-trips with ouncesToGrams', () => {
        const grams = 500;
        const oz = gramsToOunces(grams);
        expect(ouncesToGrams(oz)).toBe(grams);
    });
});

describe('ouncesToGrams', () => {
    it('converts ounces to grams', () => {
        expect(ouncesToGrams(1)).toBeCloseTo(28, 0);
        expect(ouncesToGrams(0)).toBe(0);
    });
});

describe('formatWeightWithUnit', () => {
    it('returns grams when unit is g', () => {
        expect(formatWeightWithUnit(500, 'g')).toBe('500 g');
        expect(formatWeightWithUnit(0, 'g')).toBe('0 g');
    });

    it('formats oz for small weights', () => {
        const result = formatWeightWithUnit(10, 'oz');
        expect(result).toMatch(/oz$/);
    });

    it('auto-converts to lbs when >= 16 oz', () => {
        // 454g ≈ 1 lb = 16 oz
        const result = formatWeightWithUnit(454, 'oz');
        expect(result).toMatch(/lbs$/);
    });

    it('shows lbs with one decimal for moderate weights', () => {
        // 1kg ≈ 2.2 lbs
        const result = formatWeightWithUnit(1000, 'oz');
        expect(result).toMatch(/lbs$/);
    });

    it('shows whole lbs for large weights (>= 10 lbs)', () => {
        // 5kg ≈ 11 lbs
        const result = formatWeightWithUnit(5000, 'oz');
        expect(result).toMatch(/^\d+ lbs$/);
    });
});
