import {Page} from 'puppeteer';
import {newTest, GotifyTest} from './setup';
import {count, innerText, waitForExists, waitToDisappear, clearField} from './utils';
import {afterAll, beforeAll, describe, expect, it} from 'vitest';
import * as auth from './authentication';
import * as selector from './selector';

let page: Page;
let gotify: GotifyTest;

beforeAll(async () => {
    gotify = await newTest();
    page = gotify.page;
});

afterAll(async () => await gotify.close());

const $dialog = selector.form('#app-dialog');
const $tokenDialog = selector.form('#token-dialog');

const card = (id: number) => `.channel-card[data-channel-id="${id}"]`;

const waitForChannel =
    (id: number, name: string, description: string): (() => Promise<void>) =>
    async () => {
        await waitForExists(page, `${card(id)} .channel-name`, name);
        expect(await innerText(page, `${card(id)} .channel-description`)).toBe(description);
    };

const updateChannel =
    (id: number, data: {name?: string; description?: string}): (() => Promise<void>) =>
    async () => {
        await page.click(`${card(id)} .channel-actions`);
        await page.waitForSelector('.edit');
        await page.click('.edit');
        await page.waitForSelector($dialog.selector());

        if (data.name) {
            const nameSelector = $dialog.input('.name');
            await clearField(page, nameSelector);
            await page.type(nameSelector, data.name);
        }
        if (data.description) {
            const descSelector = $dialog.textarea('.description');
            await clearField(page, descSelector);
            await page.type(descSelector, data.description);
        }

        await page.click($dialog.button('.update'));
        await waitToDisappear(page, $dialog.selector());
    };

const createChannel =
    (name: string, description: string): (() => Promise<void>) =>
    async () => {
        await page.click('#create-app');
        await page.waitForSelector($dialog.selector());
        await page.type($dialog.input('.name'), name);
        await page.type($dialog.textarea('.description'), description);
        await page.click($dialog.button('.create'));
        await waitToDisappear(page, $dialog.selector());

        await page.waitForSelector($tokenDialog.p('.token'));
        const token = await innerText(page, $tokenDialog.p('.token'));
        expect(token.startsWith('gtfya.')).toBeTruthy();
        await page.click($tokenDialog.button('.finish'));
        await waitToDisappear(page, $tokenDialog.selector());
    };

describe('Channels', () => {
    it('does login', async () => await auth.login(page));

    it('navigates to Channels', async () => {
        await page.click('#navigate-apps');
        await waitForExists(page, selector.heading(), 'Channels');
    });

    it('has changed url', async () => {
        expect(page.url()).toContain('/channels');
    });

    it('does not have any Channels', async () => {
        expect(await count(page, '.channel-card')).toBe(0);
    });

    describe('create Channels', () => {
        it('server', createChannel('server', '#1'));
        it('desktop', createChannel('desktop', '#2'));
        it('raspberry', createChannel('raspberry', '#3'));
    });

    describe('has created Channels', () => {
        it('has three Channels', async () => {
            await page.waitForSelector(card(3));
            expect(await count(page, '.channel-card')).toBe(3);
        });
        it('has server Channel', waitForChannel(1, 'server', '#1'));
        it('has desktop Channel', waitForChannel(2, 'desktop', '#2'));
        it('has raspberry Channel', waitForChannel(3, 'raspberry', '#3'));
    });

    it('updates Channels', async () => {
        await updateChannel(1, {name: 'server_linux'})();
        await updateChannel(2, {description: 'kitchen_computer'})();
        await updateChannel(3, {name: 'raspberry_pi', description: 'home_pi'})();
    });

    it('has updated Channels', async () => {
        await waitForChannel(1, 'server_linux', '#1')();
        await waitForChannel(2, 'desktop', 'kitchen_computer')();
        await waitForChannel(3, 'raspberry_pi', 'home_pi')();
    });

    it('regenerates Channel token', async () => {
        await page.click(`${card(1)} .channel-actions`);
        await page.waitForSelector('.regenerate-token');
        await page.click('.regenerate-token');
        await page.waitForSelector(selector.$confirmDialog.selector());
        await page.click(selector.$confirmDialog.button('.confirm'));
        await waitToDisappear(page, selector.$confirmDialog.selector());

        await page.waitForSelector($tokenDialog.p('.token'));
        const token = await innerText(page, $tokenDialog.p('.token'));
        expect(token.startsWith('gtfya.')).toBeTruthy();
        await page.click($tokenDialog.button('.finish'));
        await waitToDisappear(page, $tokenDialog.selector());
    });

    it('deletes Channel', async () => {
        await page.click(`${card(2)} .channel-actions`);
        await page.waitForSelector('.delete');
        await page.click('.delete');
        await page.waitForSelector(selector.$confirmDialog.selector());
        await page.click(selector.$confirmDialog.button('.confirm'));
    });

    it('has deleted Channel', async () => {
        await waitToDisappear(page, card(2));
        expect(await count(page, '.channel-card')).toBe(2);
    });

    it('does logout', async () => await auth.logout(page));
});
