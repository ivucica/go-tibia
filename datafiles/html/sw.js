console.log('Script loaded!')
var cacheStorageKey = '%GO-TIBIA-CACHE-STORAGE-KEY%'
var cacheStorageKeyTibiaData = '%GO-TIBIA-DATA-CACHE-STORAGE-KEY%'

var cachesSpec = [
    {
        "name": cacheStorageKey,
        "urls": [
            "/app/",
            "/favicon.ico",
            "/app/manifest.json",

            "/app/tfont.css",
            "/app/martel.ttf",
            "/app/verdana.ttf",
            "/app/ariblk.ttf",

            "/app/main.wasm",
            "/app/wasm_exec.js",
            "/app/go-tibia.png",
            "/app/go-tibia-192.png",
            "/app/go-tibia-512.png",

            "/app/map.otbm", // Locally renderable map.
            "/app/items.otb",
            "/app/items.xml",
            "/app/outfits.xml"
        ]
    },
    {
        "name": cacheStorageKeyTibiaData,
        "urls": [
            "/app/Tibia.spr",
            "/app/Tibia.pic",
            "/app/Tibia.dat",

            /* sed -E 's@.+(/pic/4?[^\)]+).+@\1@' tfont.css | grep \/pic | sort | uniq | sed -E 's@(.+)@            "\1",@' */
            "/pic/1",
            "/pic/4?x=0&y=0&w=96&h=96",
            "/pic/4?x=106&y=183&w=8&h=17",
            "/pic/4?x=114&y=183&w=96&h=17",
            "/pic/4?x=174&y=138&w=34&h=20",
            "/pic/4?x=174&y=158&w=34&h=20",
            "/pic/4?x=256&y=0&w=8&h=96",
            "/pic/4?x=2&y=193&w=96&h=4",
            "/pic/4?x=308&y=96&w=12&h=12",
            "/pic/4?x=98&y=193&w=8&h=4",
            "/pic/4#xywh=0,0,96,96",
            "/pic/4#xywh=106,183,8,17",
            "/pic/4#xywh=174,138,34,20",
            "/pic/4#xywh=174,158,34,20",
            "/pic/4#xywh=256,0,8,96",
            "/pic/4#xywh=308,96,12,12",
            "/pic/4#xywh=98,193,8,4",
        ]
    }
];

var cachesNames = cachesSpec.map((cache) => cache.name);

var total = 0
var loaded = 0

self.addEventListener('install', function(e) {
    console.log('Install event: ' + cacheStorageKey + ', ' + cacheStorageKeyTibiaData)
    console.log(e)
    e.waitUntil(
        caches.keys().then(function(keys) {
            return Promise.all(cachesSpec.map(function (cacheSpec) {
                if (keys.indexOf(cacheSpec.name) === -1) {
                    return caches.open(cacheSpec.name).then(function (cache) {
                        // This cache is not downloaded yet.
                        total += cacheSpec.urls.length
                        console.log('Installing cache ' + cacheSpec.name + ' (total queue: ' + total + ')')
                        //return cache.addAll(cacheSpec.urls)
                        return Promise.all(cacheSpec.urls.map(function (url) {
                            return cache.add(url).then(function(/*undefined*/) {
                                return loadedMore(e)
                            })
                        }))
                    })
                } else {
                    console.log('Cache ' + cacheSpec.name + ' already installed')
                    return Promise.resolve(true)
                }
            })).then(function () {
                // Reconsider: it may be better to request a reload.
                console.log('Install complete, invoking skipWaiting to immediately install new service worker')
                return this.skipWaiting();
            })
        })
    );
})

self.addEventListener('activate', function(e) {
    console.log('Activating new cache')
    if (caches) {
        e.waitUntil(
            caches.keys().then(function (keys) {
                return Promise.all(keys.map(function (key) {
                    if (cachesNames.indexOf(key) === -1) {
                        console.log('Uninstalling ' + key)
                        return caches.delete(key)
                    }
                    // If delete is not required, return no promise that needs to be resolved.
                }))
            }).then(() => {
                return self.clients.claim()
            })
        )
    } else {
        console.log('No caches can be activated yet.')
    }
})

self.addEventListener('fetch', function(e) {
    console.log('Fetch event:', e.request.url)

    //if (e.request.url.endsWith('/main.wasm')) {
    //    console.log('force fetch for wasm');
    //    return fetch(e.request.url);
    //}

    const url = new URL(e.request.url)
    const scope = self.registration.scope

    // n.b. We could omit e.respondWith(), thus resulting in direct network
    // request without lookup into cache or the indirect invocation of fetch().
    //
    // We could examine:
    // - e.request.method=='GET',
    // - e.request.headers.get('accept').indexOf('some/mime-type') !== -1

    // For online-first resources, we could prioritize fetch(e.request), then
    // .catch(function(e) { ... response ... }) to return an offline page
    // instead.

    e.respondWith(smartFetch(e))
})

self.addEventListener('message', event => {
    // service worker received a message, for example from one of the 'window'
    // clients such as various tabs.
    console.log('sw.js received message', event.data);
    //
    // We could reply to a message like so:
    // https://stackoverflow.com/a/30226117/39974
});

function simpleFetch(e) {
    // TODO: Remove, since we use 'smartFetch' now.
    return caches.match(e.request).then(function(response) {
        if (response != null) {
            // Found in some cache. Returning the promise containing the cached response.
            console.log('Using cache for:', e.request.url)
            return response
        }
        // Not found in a cache.
        console.log('Fallback to fetch:', e.request.url)
        //return fetch(e.request.url)
        return fetchAndStore(e.request.url)

        // Alternative: assuming we want to store the response in another cache:
        // cache.open('another-cache').then(function (response) {
        //   fetch(e.request.url).then(function (response) {
        //      return cache.put(url, response)
        //   })
        // })
        //
        // caches.match() also accepts options, incl ignoreSearch, ignoreMethod, ignoreVary and cacheName
        //
        // There's also caches.matchAll() / cache.matchAll() where we can pass '/images/' and then:
        //  cache.matchAll('/images/', then(function(response) {
        //    response.forEach(function(element, index, array) {
        //      cache.delete(element)
        //    })
        // }))
    })
}

function matchCachesIndividually(e) {
    return caches.keys().then(function(cacheKeys) {
        return Promise.all(cacheKeys.map(function(cacheKey) {
            console.log('fetching', e.request.url, ' -- opening cache', cacheKey)
            return caches.open(cacheKey).then(function (cache) {
                console.log('matching', e.request.url, ' to cache', cacheKey)
                return cache.match(e.request).then(function(response) {
                    if (response != null) {
                        // Found in some cache. Returning the promise containing the cached response.
                        console.log('Using cache', cacheKey, 'for:', e.request.url)
                        return response
                    }
                    console.log('Did not find', e.request.url,'in matched cache', cacheKey)
                    //return Promise.resolve(null) //Promise.reject(new Error('Did not find ' + e.request.url + ' in cache ' + cacheKey))
                })
            })
        }))
    })
}


function handleSharePost(e) {
    // 'fetch' handler for POST requests for the share target.
    //
    // Expected to be already identified as a POST request to the share-target
    // URL in the calling handler for 'fetch' events in the service worker.
    //
    // Posts messages to all clients (open windows/tabs) with the shared data.
    //
    // The client pages are expected to listen for 'message' events and handle
    // the shared data appropriately.
    e.request.formData().then(formData => {
        const title = formData.get('title') || '';
        const text = formData.get('text') || '';
        const file = formData.get('file');

        let fileData = null;
        let fileReaderPromise = null;

        if (file) {
            fileReaderPromise = new Promise((resolve, reject) => {
                const reader = new FileReader();
                reader.onload = () => resolve(reader.result);
                reader.onerror = reject;
                reader.readAsDataURL(file);
            });
        } else {
            fileReaderPromise = Promise.resolve(null);
        }
        
        fileReaderPromise.then(fileDataURL => {
            clients.matchAll({type: "window", includeUncontrolled: true}).then(function(clientList) {
                for (var i = 0; i < clientList.length; i++) {
                    clientList[i].postMessage({
                        type: 'share',
                        title: title,
                        text: text,
                        file: fileDataURL,
                    });
                }
            });
        });
    });
    // NOTE: if the window has been opened (and on ChromeOS it does get opened),
    // we should either switch focus to an existing window and close the newly
    // open window, or handle a redirect here and handle it in local window, or
    // find a way to tell the browser/OS that the share should be handled in the
    // SW + existing window if one is already open.
    //
    // For now, saying that nothing needs to happen is fine, user gets bad
    // experience but this is for testing only anyway.
    return new Response(null, { status: 204 });
}

function smartFetch(e) {
    if (e.request.method === 'POST' && (
        e.request.url.includes('/app/_share-target-handler') ||
        e.request.url.includes('/app/share-target-handler') // v1, when it was a relative URL, should be removed
    )) {
        // Note:
        // - only appears when sharing the file types we declared in manifest.json
        // - a window is opened for /app/_share-target-handler, service worker
        //   might not be the only thing intercepting this
        return handleSharePost(e);
    }

    return matchCachesIndividually(e).then(function (response) {
        if (!response) {
            console.warn('Response from caches is unexpectedly null; falling back to fetch')
            return fetchAndStore(e)
        }
        var response = response.filter(function (itm) { return !!itm })
        if (response.length == 0) {
            console.warn('All responses from caches are unexpectedly falsy; falling back to fetch')
            return fetchAndStore(e)
        }
        return response[0]
    }).catch(function (err) {
        console.error(err)
    })
}

function fetchAndStore(e) {
    return fetch(e.request.url).then(function(response) {
        // TODO: do not store if a header prevents us from doing so
        // TODO: only cache if in cachesSpec

        if (!response.ok) {
            throw new TypeError('bad response status')
        }

        var url = e.request.url
        if (!url.startsWith('chrome-extension://')) {
            if (url.endsWith('Tibia.dat') || url.endsWith('Tibia.pic') || url.endsWith('Tibia.spr') || url.startsWith('/pic/')) {
                console.warn('Override: Storing ' + url + ' for later use into', cacheStorageKeyTibiaData)
                caches.open(cacheStorageKeyTibiaData).then(function(cacheTD) {
                    return cacheTD.put(url, response).then(_ => console.log('Stored', url, 'into TD cache', cacheStorageKeyTibiaData))
                })
                return response.clone()
            } else {
                console.warn('Override: Storing ' + url + ' for later use into', cacheStorageKey)
                caches.open(cacheStorageKey).then(function(cacheMain) {
                    return cacheMain.put(url, response).then(_ => console.log('Stored', url, 'into main cache', cacheStorageKey))
                })
                return response.clone()
            }

            // n.b. We can also just return response without cache.put if it's a noncacheable request.

            console.warn('Storing', url, 'for later use into', cacheKey)
            cache.put(url, response).then(_ => console.log('Stored', url, 'into', cacheKey))
            return response.clone()
        } else {
            // Seriously. Sometimes Chrome sends chrome-extension:// URLs through here.
            // Ideally we should not even use e.respondWith in this case.
            return response;
        }

    }).then(function(response) {
        if (false && e.clientId) {
            loadedMore()
        }
        return response
    }).catch(function(e) {
        console.error('Fetch failed:', e)
    })
}


function loadedMore(e) {
    loaded += 1
    console.log('Loaded ' + loaded + ' out of ' + total)

    if (e.clientId) {
        self.clients.get(e.clientId).then(function(client) {
            if (client) {
                client.postMessage({loaded, total})
            } else {
                console.warn('Client ' + e.clientId + ' not found')
            }
        })
    }

    // Notify all other clients of the install progress.

    clients.matchAll({type: "window", includeUncontrolled: true}).then(function(clientList) {
        console.log(clientList)
        for (var i = 0; i < clientList.length; i++) {
            var client = clientList[i];
            client.postMessage({loaded, total})
        }
    })
}
