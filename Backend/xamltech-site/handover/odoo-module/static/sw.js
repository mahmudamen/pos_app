// XamlTech Service Worker v5
// CRITICAL: Never cache HTML pages (fixes language switching blocking)
// Only caches static assets (images, fonts, CSS, JS)
var CACHE_NAME = 'xt-v5';
var OFFLINE_URL = '/xamltech_addons_store/static/offline.html';
var PRECACHE = [
    '/xamltech_addons_store/static/img/logo.png',
    '/xamltech_addons_store/static/img/favicon.png',
    OFFLINE_URL
];

// Install: precache offline page + core assets
self.addEventListener('install', function (e) {
    e.waitUntil(
        caches.open(CACHE_NAME)
            .then(function (cache) { return cache.addAll(PRECACHE); })
    );
    self.skipWaiting();
});

// Activate: clean old caches
self.addEventListener('activate', function (e) {
    e.waitUntil(
        caches.keys().then(function (keys) {
            return Promise.all(
                keys.filter(function (k) { return k !== CACHE_NAME; })
                    .map(function (k) { return caches.delete(k); })
            );
        })
    );
    self.clients.claim();
});

// Fetch: NEVER cache HTML, only cache /static/ assets
self.addEventListener('fetch', function (e) {
    var request = e.request;
    var url = request.url;

    // Skip non-GET
    if (request.method !== 'GET') return;

    // NEVER intercept navigation (HTML pages)
    // This is the key fix for language switching
    if (request.mode === 'navigate') {
        e.respondWith(
            fetch(request).catch(function () {
                return caches.match(OFFLINE_URL);
            })
        );
        return;
    }

    // NEVER cache Odoo backend, API, jsonrpc
    if (url.indexOf('/web/') > -1 ||
        url.indexOf('/odoo/') > -1 ||
        url.indexOf('/jsonrpc') > -1 ||
        url.indexOf('/website/translations') > -1 ||
        url.indexOf('/web/webclient/') > -1) {
        return;
    }

    // Only cache static assets
    if (url.indexOf('/static/') > -1) {
        e.respondWith(
            caches.match(request).then(function (cached) {
                if (cached) return cached;
                return fetch(request).then(function (response) {
                    if (response.ok && response.status === 200) {
                        var clone = response.clone();
                        caches.open(CACHE_NAME).then(function (cache) {
                            cache.put(request, clone);
                        });
                    }
                    return response;
                }).catch(function () {
                    // For images, return placeholder
                    if (url.match(/\.(png|jpg|jpeg|gif|svg|webp)$/)) {
                        return new Response('', { status: 404 });
                    }
                    return new Response('', { status: 404 });
                });
            })
        );
        return;
    }

    // Everything else: network only (no cache)
});

// Listen for messages (cache clear from JS)
self.addEventListener('message', function (e) {
    if (e.data === 'xt-clear-cache') {
        caches.keys().then(function (keys) {
            keys.forEach(function (k) { caches.delete(k); });
        });
    }
});
