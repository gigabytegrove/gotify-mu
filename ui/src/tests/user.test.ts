import {Page} from 'puppeteer';
import {newTest, GotifyTest} from './setup';
import {clearField, clickByText, count, innerText, waitForExists, waitToDisappear} from './utils';
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

enum Col {
    Name = 1,
    DisplayName = 2,
    Role = 3,
    Created = 4,
    EditDelete = 5,
}

const $table = selector.table('#user-table');
const $dialog = selector.form('#add-edit-user-dialog');

describe('User', () => {
    it('does login', async () => await auth.login(page));
    it('navigates to users through window location', async () => {
        await page.goto(gotify.url + '/#/users');
        await waitForExists(page, selector.heading(), 'Users');
    });
    it('has changed url', async () => {
        expect(page.url()).toContain('/users');
    });
    it('has only admin user (the current one)', async () => {
        expect(await count(page, $table.rows())).toBe(1);
    });
    describe('create users', () => {
        const createUser =
            (name: string, password: string, isAdmin: boolean): (() => Promise<void>) =>
            async () => {
                await page.click('#create-user');
                await page.waitForSelector($dialog.selector());
                await page.type($dialog.input('.name'), name);
                await page.type($dialog.input('.password'), password);
                if (isAdmin) {
                    await page.click($dialog.input('.admin-rights'));
                }
                await page.click($dialog.button('.save-create'));
                await waitToDisappear(page, $dialog.selector());
            };
        it('nicories', createUser('nicories', 'nicories-pass-123', false));
        it('jmattheis', createUser('jmattheis', 'jmattheis-pass-123', true));
        it('dude', createUser('dude', 'dude-pass-123', false));
    });
    const hasUser =
        (name: string, isAdmin: boolean, row: number): (() => Promise<void>) =>
        async () => {
            expect(await innerText(page, $table.cell(row, Col.Name))).toBe(name);
            expect(await innerText(page, $table.cell(row, Col.Role))).toBe(isAdmin ? 'Administrator' : 'User');
        };

    describe('has created users', () => {
        it('has four users', async () => {
            await page.waitForSelector($table.row(4));
            expect(await count(page, $table.rows())).toBe(4);
        });
        it('has admin user', hasUser('admin', true, 1));
        it('has nicories user', hasUser('nicories', false, 2));
        it('has jmattheis user', hasUser('jmattheis', true, 3));
        it('has dude user', hasUser('dude', false, 4));
    });
    describe('edit users', () => {
        it('changes password of jmattheis', async () => {
            await page.click($table.cell(3, Col.EditDelete, '.edit'));
            await page.waitForSelector($dialog.selector());
            await page.type($dialog.input('.password'), 'unicorn-pass-123');
            await page.click($dialog.button('.save-create'));
            await waitToDisappear(page, $dialog.selector());
        });
        it('changed jmattheis', hasUser('jmattheis', true, 3));

        it('changes name of nicories', async () => {
            await page.click($table.cell(2, Col.EditDelete, '.edit'));

            await page.waitForSelector($dialog.selector());

            await clearField(page, $dialog.input('.name'));
            await page.type($dialog.input('.name'), 'nicolas');
            await page.click($dialog.button('.save-create'));
            await waitToDisappear(page, $dialog.selector());

            await waitForExists(page, $table.cell(2, Col.Name), 'nicolas');
        });
        it('changed nicories to nicolas', hasUser('nicolas', false, 2));

        it('makes dude admin', async () => {
            await page.click($table.cell(4, Col.EditDelete, '.edit'));

            await page.waitForSelector($dialog.selector());

            await page.click($dialog.input('.admin-rights'));
            await page.click($dialog.button('.save-create'));
            await waitToDisappear(page, $dialog.selector());

            await waitForExists(page, $table.cell(4, Col.Role), 'Administrator');
        });
        it('made dude admin', hasUser('dude', true, 4));
    });

    it('deletes dude', async () => {
        await page.click($table.cell(4, Col.EditDelete, '.delete'));

        await page.waitForSelector(selector.$confirmDialog.selector());
        await page.click(selector.$confirmDialog.button('.confirm'));
    });
    it('has deleted dude', async () => {
        await waitToDisappear(page, $table.row(4));
        expect(await count(page, $table.rows())).toBe(3);
    });
    it('changes password of current user', async () => {
        const $changepw = selector.form('#changepw-form');
        await page.waitForSelector('#user-menu-button');
        await page.click('#user-menu-button');
        await clickByText(page, '#user-menu [role="menuitem"]', 'Settings');
        await waitToDisappear(page, '#user-menu');
        await waitForExists(page, selector.heading(), 'Settings');
        const elevationPassword = '.elevation-password input';
        if (await page.$(elevationPassword)) {
            await page.type(elevationPassword, 'admin');
            await page.click('.elevation-submit');
        }
        await page.waitForSelector($changepw.selector());
        await page.type($changepw.input('.newpass'), 'changed-pass-123');
        const [response] = await Promise.all([
            page.waitForResponse(
                (candidate) =>
                    candidate.request().method() === 'POST' &&
                    candidate.url().endsWith('/current/user/password')
            ),
            page.click($changepw.button('.change')),
        ]);
        expect(response.status()).toBe(200);
    });
    it('does logout', async () => await auth.logout(page));
    it('can login with new password (admin)', async () =>
        await auth.login(page, 'admin', 'changed-pass-123'));
    it('does logout admin', async () => await auth.logout(page));

    it('can login with nicolas', async () => await auth.login(page, 'nicolas', 'nicories-pass-123'));
    it('does logout nicolas', async () => await auth.logout(page));
    it('can login with jmattheis', async () => await auth.login(page, 'jmattheis', 'unicorn-pass-123'));
    it('does logout jmattheis', async () => await auth.logout(page));
});
