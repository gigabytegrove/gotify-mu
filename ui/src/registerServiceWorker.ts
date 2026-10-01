const serviceWorkerSupported = () => 'serviceWorker' in navigator && window.isSecureContext;

export function register() {
    if (!serviceWorkerSupported()) return;

    window.addEventListener('load', () => {
        const workerUrl = new URL('sw.js', document.baseURI).toString();
        navigator.serviceWorker
            .register(workerUrl, {scope: './'})
            .then((registration) => {
                registration.addEventListener('updatefound', () => {
                    const worker = registration.installing;
                    if (!worker) return;
                    worker.addEventListener('statechange', () => {
                        if (worker.state === 'installed' && navigator.serviceWorker.controller) {
                            window.dispatchEvent(new CustomEvent('monita-pwa-update-ready'));
                        }
                    });
                });
            })
            .catch((error) => {
                console.warn('Monita PWA service worker registration failed', error);
            });
    });
}

export function unregister() {
    if (!('serviceWorker' in navigator)) return;
    void navigator.serviceWorker.ready.then((registration) => registration.unregister());
}
