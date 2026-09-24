import {Page} from 'puppeteer';
import {waitForExists} from './utils';
import {expect} from 'vitest';
import * as selector from './selector';

const $loginForm = selector.form('#login-form');

export const login = async (page: Page, user = 'admin', pass = 'admin'): Promise<void> => {
    await waitForExists(page, selector.heading(), 'Sign in');
    expect(page.url()).toContain('/login');
    await page.type($loginForm.input('.name'), user);
    await page.type($loginForm.input('.password'), pass);
    await page.click($loginForm.button('.login'));
    await waitForExists(page, selector.heading(), 'Dashboard');
};

export const logout = async (page: Page): Promise<void> => {
    await page.waitForSelector('#user-menu-button');
    await page.click('#user-menu-button');
    await page.waitForSelector('#logout');
    await page.click('#logout');
    await waitForExists(page, selector.heading(), 'Sign in');
    expect(page.url()).toContain('/login');
};
