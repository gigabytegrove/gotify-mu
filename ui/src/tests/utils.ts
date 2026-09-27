import {ElementHandle, JSHandle, Page} from 'puppeteer';

export const innerText = async (page: ElementHandle | Page, selector: string): Promise<string> => {
    const element = await page.$(selector);
    const handle = await element!.getProperty('innerText');
    const value = await handle.jsonValue();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return (value as any).toString().trim();
};

const isVisible = (element: Element): boolean => {
    const html = element as HTMLElement;
    const style = window.getComputedStyle(html);
    return (
        style.display !== 'none' &&
        style.visibility !== 'hidden' &&
        html.getClientRects().length > 0
    );
};

export const clickByText = async (page: Page, selector: string, text: string): Promise<void> => {
    await waitForExists(page, selector, text);
    text = text.toLowerCase();
    await page.evaluate(
        (_selector, _text) => {
            const element = Array.from(document.querySelectorAll(_selector)).find((candidate) => {
                const html = candidate as HTMLElement;
                const style = window.getComputedStyle(html);
                return (
                    style.display !== 'none' &&
                    style.visibility !== 'hidden' &&
                    html.getClientRects().length > 0 &&
                    candidate.textContent?.toLowerCase().trim() === _text
                );
            }) as HTMLButtonElement | undefined;
            if (!element) throw new Error(`No visible element found for text: ${_text}`);
            element.click();
        },
        selector,
        text
    );
};

export const count = async (page: Page, selector: string): Promise<number> =>
    page.$$(selector).then((elements) => elements.length);

export const waitToDisappear = async (page: Page, selector: string): Promise<JSHandle> =>
    page.waitForFunction((_selector: string) => !document.querySelector(_selector), {}, selector);

export const waitForCount = async (
    page: Page,
    selector: string,
    amount: number
): Promise<JSHandle> =>
    page.waitForFunction(
        (_selector: string, _amount: number) =>
            document.querySelectorAll(_selector).length === _amount,
        {},
        selector,
        amount
    );

export const waitForExists = async (page: Page, selector: string, text: string): Promise<void> => {
    text = text.toLowerCase();
    await page.waitForFunction(
        (_selector: string, _text: string) =>
            Array.from(document.querySelectorAll(_selector)).some((element) => {
                const html = element as HTMLElement;
                const style = window.getComputedStyle(html);
                return (
                    style.display !== 'none' &&
                    style.visibility !== 'hidden' &&
                    html.getClientRects().length > 0 &&
                    element.textContent!.toLowerCase().trim() === _text
                );
            }),
        {},
        selector,
        text
    );
};

export const clearField = async (element: ElementHandle | Page, selector: string) => {
    const elementHandle = await element.$(selector);
    if (!elementHandle) {
        throw 'element handle not set';
    }
    await elementHandle.click();
    await elementHandle.focus();
    // click three times to select all
    await elementHandle.click({count: 3});
    await elementHandle.press('Backspace');
};

export enum ClientCol {
    Name = 1,
    Status = 2,
    ElevationEnds = 3,
    ExpiresIn = 4,
    LastSeen = 5,
    Created = 6,
    Elevate = 7,
    Edit = 7,
    Delete = 7,
}
