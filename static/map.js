// Map functionality for the Parts Pile site
// Uses Leaflet for map rendering and HTMX for server communication

// Global map instance
let map = null;
let markers = [];
let currentBounds = null;
let mapInitialized = false;
let isInitializing = false;
let searchDebounceTimer = null;
let isInitialSetup = true;

function initMap() {
    console.log('[map.js] initMap called');
    
    // Prevent multiple initializations
    if (mapInitialized || isInitializing) {
        console.log('[map.js] Map already initialized or initializing, skipping');
        return;
    }
    
    isInitializing = true;
    isInitialSetup = true;
    
    // Check if Leaflet is available
    if (typeof L === 'undefined') {
        console.error('[map.js] Leaflet not loaded');
        isInitializing = false;
        return;
    }
    
    // Check if map container exists
    const container = document.getElementById('map-container');
    if (!container) {
        console.error('[map.js] Map container not found');
        isInitializing = false;
        return;
    }
    
    console.log('[map.js] Initializing map...');
    
    try {
        // Initialize map with default view (Europe center where ads are located)
        map = L.map('map-container').setView([48.99, 7.44], 6);
        
        // Add tile layer (OpenStreetMap)
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);
        
        console.log('[map.js] Map initialized, adding markers...');
        
        // Add ads as markers
        addAdMarkers();
        
        // Listen for map move/zoom events with debouncing
        let moveTimeout;
        map.on('moveend', function() {
            // Don't trigger search during initial setup
            if (isInitialSetup) {
                console.log('[map.js] Skipping moveend during initial setup');
                return;
            }
            clearTimeout(moveTimeout);
            moveTimeout = setTimeout(() => updateBoundingBox(true), 500);
        });
        map.on('zoomend', function() {
            // Don't trigger search during initial setup
            if (isInitialSetup) {
                console.log('[map.js] Skipping zoomend during initial setup');
                return;
            }
            clearTimeout(moveTimeout);
            moveTimeout = setTimeout(() => updateBoundingBox(true), 500);
        });
        
        // Initial bounding box update (but don't trigger search)
        updateBoundingBox(false);
        
        // Mark initial setup as complete after a short delay
        setTimeout(() => {
            isInitialSetup = false;
            console.log('[map.js] Initial setup complete, user interactions will now trigger searches');
        }, 1000);
        
        mapInitialized = true;
        isInitializing = false;
        console.log('[map.js] Map initialization complete');
        
        // Force a resize to ensure the map renders properly
        setTimeout(() => {
            if (map) {
                map.invalidateSize();
                console.log('[map.js] Map size invalidated');
            }
        }, 100);
        
    } catch (error) {
        console.error('[map.js] Error initializing map:', error);
        isInitializing = false;
    }
}

function addAdMarkers() {
    console.log('[map.js] addAdMarkers called');
    
    if (!map) {
        console.error('[map.js] Map not initialized');
        return;
    }
    
    // Clear existing markers
    markers.forEach(marker => {
        try {
            map.removeLayer(marker);
        } catch (e) {
            console.warn('[map.js] Error removing marker:', e);
        }
    });
    markers = [];
    
    // Get ads data from server-rendered HTML
    const adElements = document.querySelectorAll('[data-ad-id]');
    console.log('[map.js] Found', adElements.length, 'ad elements');
    
    adElements.forEach((element, index) => {
        const adId = element.dataset.adId;
        const lat = parseFloat(element.dataset.lat);
        const lon = parseFloat(element.dataset.lon);
        const title = element.dataset.title;
        const price = element.dataset.price;
        
        console.log(`[map.js] Ad ${index + 1}: ID=${adId}, lat=${lat}, lon=${lon}, title=${title}`);
        
        if (lat && lon && !isNaN(lat) && !isNaN(lon)) {
            try {
                const popupContent = `
                    <div class="map-popup">
                        <h4>${title}</h4>
                        <p class="price">$${price}</p>
                        <button onclick="viewAd(${adId})" class="btn btn-primary btn-sm">View Details</button>
                    </div>
                `;
                
                const marker = L.marker([lat, lon])
                    .bindPopup(popupContent)
                    .addTo(map);
                markers.push(marker);
                console.log(`[map.js] Added marker for ad ${adId} at [${lat}, ${lon}]`);
            } catch (error) {
                console.error(`[map.js] Error adding marker for ad ${adId}:`, error);
            }
        } else {
            console.warn(`[map.js] Skipping ad ${adId} - invalid coordinates: lat=${lat}, lon=${lon}`);
        }
    });
    
    console.log('[map.js] Total markers added:', markers.length);
    
    // Fit map to markers if we have any (but don't trigger search)
    if (markers.length > 0) {
        try {
            const group = new L.featureGroup(markers);
            map.fitBounds(group.getBounds().pad(0.1));
            console.log('[map.js] Map fitted to markers');
        } catch (error) {
            console.error('[map.js] Error fitting map to markers:', error);
        }
    }
}

function updateBoundingBox(triggerSearch = false) {
    if (!map) {
        console.log('[map.js] updateBoundingBox called but map not initialized');
        return;
    }
    
    try {
        const bounds = map.getBounds();
        currentBounds = bounds;
        
        console.log('[map.js] Map bounds:', bounds.toString());
        
        // Update hidden inputs
        const minLatInput = document.getElementById('min-lat');
        const maxLatInput = document.getElementById('max-lat');
        const minLonInput = document.getElementById('min-lon');
        const maxLonInput = document.getElementById('max-lon');
        
        if (minLatInput && maxLatInput && minLonInput && maxLonInput) {
            minLatInput.value = bounds.getSouth();
            maxLatInput.value = bounds.getNorth();
            minLonInput.value = bounds.getWest();
            maxLonInput.value = bounds.getEast();
        }
        
        // Only trigger search if explicitly requested and not during initial setup
        if (triggerSearch && !isInitialSetup) {
            triggerMapSearch();
        }
    } catch (error) {
        console.error('[map.js] Error updating bounding box:', error);
    }
}

function triggerMapSearch() {
    // Clear any existing debounce timer
    if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer);
    }
    
    // Debounce the search request
    searchDebounceTimer = setTimeout(() => {
        try {
            const searchBox = document.getElementById('searchBox');
            const query = searchBox ? searchBox.value : '';
            
            // Build URL with current search + bounding box
            const params = new URLSearchParams();
            if (query) params.append('q', query);
            params.append('view', 'map');
            
            const minLatInput = document.getElementById('min-lat');
            const maxLatInput = document.getElementById('max-lat');
            const minLonInput = document.getElementById('min-lon');
            const maxLonInput = document.getElementById('max-lon');
            
            if (minLatInput && maxLatInput && minLonInput && maxLonInput) {
                params.append('minLat', minLatInput.value);
                params.append('maxLat', maxLatInput.value);
                params.append('minLon', minLonInput.value);
                params.append('maxLon', maxLonInput.value);
            }
            
            console.log('[map.js] Triggering search with params:', params.toString());
            
            // Use HTMX to update results
            htmx.ajax('GET', `/search?${params.toString()}`, {
                target: '#searchResults',
                swap: 'outerHTML'
            });
        } catch (error) {
            console.error('[map.js] Error triggering search:', error);
        }
    }, 1000); // 1 second debounce
}

function viewAd(adId) {
    // Navigate to ad detail page
    window.location.href = `/ad/${adId}`;
}

function fitMapToMarkers() {
    if (markers.length > 0 && map) {
        try {
            const group = new L.featureGroup(markers);
            map.fitBounds(group.getBounds().pad(0.1));
        } catch (error) {
            console.error('[map.js] Error fitting map to markers:', error);
        }
    }
}

// Helper function to check if we should initialize map
function shouldInitMap() {
    const mapContainer = document.getElementById('map-container');
    const mapView = document.getElementById('map-view');
    return mapContainer && mapView && !mapInitialized && !isInitializing;
}

// Helper function to attempt map initialization with retries
function attemptMapInit() {
    console.log('[map.js] Attempting map initialization...');
    
    if (shouldInitMap()) {
        console.log('[map.js] Conditions met, initializing map...');
        setTimeout(initMap, 50); // Small delay to ensure DOM is ready
    } else {
        console.log('[map.js] Conditions not met - container:', !!document.getElementById('map-container'), 
                   'view:', !!document.getElementById('map-view'), 
                   'initialized:', mapInitialized,
                   'initializing:', isInitializing);
    }
}

// Initialize map when DOM is ready
document.addEventListener('htmx:load', function() {
    console.log('[map.js] htmx:load event fired');
    attemptMapInit();
});

// Re-initialize map when view changes to map
document.addEventListener('htmx:afterSwap', function() {
    console.log('[map.js] htmx:afterSwap event fired');
    // Reset initialization state when content is swapped
    mapInitialized = false;
    isInitializing = false;
    isInitialSetup = true;
    map = null;
    markers = [];
    currentBounds = null;
    
    // Clear any pending search requests
    if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer);
        searchDebounceTimer = null;
    }
    
    // Wait a bit for DOM to be updated
    setTimeout(attemptMapInit, 100);
});

// Also try to initialize on regular DOMContentLoaded
document.addEventListener('DOMContentLoaded', function() {
    console.log('[map.js] DOMContentLoaded event fired');
    attemptMapInit();
});

// Additional initialization for map view specifically
document.addEventListener('htmx:afterRequest', function(event) {
    console.log('[map.js] htmx:afterRequest event fired');
    // Check if we're in map view and try to initialize
    setTimeout(attemptMapInit, 200); // Longer delay for after request
});

// Listen for map view button clicks specifically
document.addEventListener('click', function(event) {
    // Check if the clicked element is the map view button
    if (event.target && event.target.closest && event.target.closest('[hx-vals*="map"]')) {
        console.log('[map.js] Map view button clicked, will attempt initialization after swap');
        // The actual initialization will happen in htmx:afterSwap
    }
});

// Try to initialize map every second for the first 10 seconds after page load
let initAttempts = 0;
const maxInitAttempts = 10;
const initInterval = setInterval(function() {
    initAttempts++;
    console.log(`[map.js] Init attempt ${initAttempts}/${maxInitAttempts}`);
    
    if (shouldInitMap()) {
        console.log('[map.js] Map container found on interval, initializing map...');
        initMap();
    }
    
    if (initAttempts >= maxInitAttempts || mapInitialized) {
        clearInterval(initInterval);
        console.log('[map.js] Stopping init attempts');
    }
}, 1000); 