import {describe, expect, it} from 'vitest';
import {classifyUpdate, compareVersions, latestPublishedRelease, parseVersion} from './release';

describe('release update helpers', () => {
    it('parses release versions with or without v prefix', () => {
        expect(parseVersion('0.2.0')).toEqual([0, 2, 0]);
        expect(parseVersion('v1.12.3')).toEqual([1, 12, 3]);
    });

    it('does not treat development build names as semantic releases', () => {
        expect(parseVersion('master-1ae117f')).toBeNull();
        expect(parseVersion('master-local')).toBeNull();
    });

    it('compares semantic release versions', () => {
        expect(compareVersions('0.1.9', '0.2.0')).toBe(-1);
        expect(compareVersions('0.2.0', 'v0.2.0')).toBe(0);
        expect(compareVersions('0.3.0', '0.2.0')).toBe(1);
    });

    it('classifies update state', () => {
        expect(classifyUpdate('0.1.0', '0.2.0')).toBe('available');
        expect(classifyUpdate('0.2.0', '0.2.0')).toBe('current');
        expect(classifyUpdate('0.3.0', '0.2.0')).toBe('newer');
        expect(classifyUpdate('master-local', '0.2.0')).toBe('development');
    });

    it('includes prereleases while ignoring drafts', () => {
        const release = latestPublishedRelease([
            {
                tag_name: 'v0.3.0',
                name: 'draft',
                html_url: 'https://example.invalid/draft',
                draft: true,
                prerelease: false,
                published_at: null,
                assets: [],
            },
            {
                tag_name: 'v0.2.0',
                name: 'Gotify MU v0.2.0',
                html_url: 'https://example.invalid/release',
                draft: false,
                prerelease: true,
                published_at: null,
                assets: [],
            },
        ]);
        expect(release?.tag_name).toBe('v0.2.0');
    });
});
