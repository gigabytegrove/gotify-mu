export const RELEASES_API =
    'https://api.github.com/repos/gigabytegrove/gotify-mu/releases?per_page=10';

export interface ReleaseAsset {
    name: string;
    browser_download_url: string;
    size: number;
}

export interface PublishedRelease {
    tag_name: string;
    name: string | null;
    html_url: string;
    draft: boolean;
    prerelease: boolean;
    published_at: string | null;
    assets: ReleaseAsset[];
}

export type UpdateClassification = 'available' | 'current' | 'newer' | 'development';

const SEMVER = /^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$/i;

export const parseVersion = (value: string): [number, number, number] | null => {
    const match = value.trim().match(SEMVER);
    if (!match) return null;
    return [Number(match[1]), Number(match[2]), Number(match[3])];
};

export const compareVersions = (left: string, right: string): number | null => {
    const a = parseVersion(left);
    const b = parseVersion(right);
    if (!a || !b) return null;

    for (let index = 0; index < 3; index += 1) {
        if (a[index] > b[index]) return 1;
        if (a[index] < b[index]) return -1;
    }
    return 0;
};

export const classifyUpdate = (
    currentVersion: string,
    publishedVersion: string
): UpdateClassification => {
    const comparison = compareVersions(currentVersion, publishedVersion);
    if (comparison === null) return 'development';
    if (comparison < 0) return 'available';
    if (comparison > 0) return 'newer';
    return 'current';
};

export const latestPublishedRelease = (releases: PublishedRelease[]): PublishedRelease | null =>
    releases.find((release) => !release.draft) ?? null;
