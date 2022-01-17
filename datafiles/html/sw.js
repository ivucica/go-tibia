console.log('Script loaded!')
var cacheStorageKey = 'gotweb-fe-3'

var cacheList = [
    "/app/",
    "/favicon.ico",

    "/app/main.wasm",
    "/app/wasm_exec.js",
    "/app/go-tibia.png",
    "/app/go-tibia-192.png",
    "/app/go-tibia-512.png",
    "/app/Tibia.spr",
    "/app/Tibia.pic",
    "/app/Tibia.dat",
    "/app/map.otbm", // Locally renderable map.
    "/app/items.otb",
    "/app/items.xml",
    "/app/outfits.xml"
]

self.addEventListener('install', function(e) {
    console.log('Cache event!')
    e.waitUntil(
        caches.open(cacheStorageKey).then(function(cache) {
            console.log('Adding to Cache:', cacheList)
            return cache.addAll(cacheList)
        }).then(function() {
            console.log('Skip waiting!')
            return self.skipWaiting()
        })
    )
})

self.addEventListener('activate', function(e) {
    console.log('Activate event')
    if (caches) {
            console.log('caches keys: ')
            console.log(caches.keys())
        e.waitUntil(
            caches.keys().then(cacheNames => {
                return cacheNames.map(key => {
                    if (key !== cacheStorageKey) {
                        return caches.delete(name)
                    }
                })
            }).then(() => {
                console.log('Clients claims.')
                return self.clients.claim()
            })
        )
    } else {
        console.log('no caches yet');
    }
})

self.addEventListener('fetch', function(e) {
    console.log('Fetch event:', e.request.url)
    //if (e.request.url.endsWith('/main.wasm')) {
    //    console.log('force fetch for wasm');
    //    return fetch(e.request.url);
    //}
    e.respondWith(
        caches.match(e.request).then(function(response) {
            if (response != null) {
                console.log('Using cache for:', e.request.url)
                return response
            }
            console.log('Fallback to fetch:', e.request.url)
            return fetch(e.request.url)
        })
    )
})
