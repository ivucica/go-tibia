console.log('Script loaded!')
var cacheStorageKey = '%GO-TIBIA-CACHE-STORAGE-KEY%'
var cacheStorageKeyTibiaData = '%GO-TIBIA-DATA-CACHE-STORAGE-KEY%'

var cacheList = [
    "/app/",
    "/favicon.ico",

    "/app/main.wasm",
    "/app/wasm_exec.js",
    "/app/go-tibia.png",
    "/app/go-tibia-192.png",
    "/app/go-tibia-512.png",

    "/app/map.otbm", // Locally renderable map.
    "/app/items.otb",
    "/app/items.xml",
    "/app/outfits.xml"
];

var cacheListTibiaData = [
    "/app/Tibia.spr",
    "/app/Tibia.pic",
    "/app/Tibia.dat",
]

self.addEventListener('install', function(e) {
    console.log('Cache event!')
    e.waitUntil(
        caches.open(cacheStorageKey).then(function(cache) {
            console.log('Adding to Cache:', cacheList)
            return cache.addAll(cacheList)
        }).then(function() {
            caches.open(cacheStorageKeyTibiaData).then(function(cache) {
                console.log('Adding to Cache:', cacheListTibiaData)
                return cache.addAll(cacheListTibiaData)
            }).then(function() {
                console.log('Skip waiting big data!')
                return self.skipWaiting()
            })
        }).then(function() {
            console.log('Skip waiting main!')
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
                    if (key !== cacheStorageKey && key != cacheStorageKeyTibiaData) {
                        return caches.delete(key)
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
