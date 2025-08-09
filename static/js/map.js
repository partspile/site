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
let hasEverBeenInitialized = false;

function initMap() {
    console.log('[map.js] initMap called');
    
    // Prevent multiple initializations
    if (mapInitialized || isInitializing) {
        console.log('[map.js] Map already initialized or initializing, just updating markers');
        if (mapInitialized) {
            // Just update markers without fitting bounds
            addAdMarkers();
        }
        return;
    }
    
    isInitializing = true;
    // Only set isInitialSetup to true if this is the very first initialization
    isInitialSetup = !hasEverBeenInitialized;
    
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
        hasEverBeenInitialized = true;
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
                
                // Store the ad ID on the marker for tracking
                marker._adId = adId;
                
                markers.push(marker);
                console.log(`[map.js] Added marker for ad ${adId} at [${lat}, ${lon}]`);
            } catch (error) {
                console.error(`[map.js] Error adding marker for ad ${adId}:`, error);
            }
        } else {
            console.warn(`[map.js] Skipping ad ${adId} - invalid coordinates: lat=${lat}, lon=${lon}`);
        }
    });
    
    console.log('[map.js] Total markers added (v2):', markers.length);
    
    // Only fit bounds on the very first page load to establish initial view
    // After that, user controls the map view completely
    console.log(`[map.js] addAdMarkers: markers=${markers.length}, hasEverBeenInitialized=${hasEverBeenInitialized}`);
    if (markers.length > 0 && !hasEverBeenInitialized) {
        try {
            const group = new L.featureGroup(markers);
            map.fitBounds(group.getBounds().pad(0.1));
            console.log('[map.js] Initial view: fitted map to show all ads');
        } catch (error) {
            console.error('[map.js] Error fitting map to markers:', error);
        }
    } else {
        console.log(`[map.js] Updated ${markers.length} markers, keeping user's chosen view (hasEverBeenInitialized=${hasEverBeenInitialized})`);
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
            
            // Use direct fetch to get ads and update markers only
            fetch(`/api/search?${params.toString()}`)
                .then(response => response.json())
                .then(data => {
                    console.log('[map.js] Search completed, got', data.ads?.length || 0, 'ads');
                    // Update markers directly without HTMX
                    updateMarkersFromData(data.ads || []);
                })
                .catch(error => {
                    console.error('[map.js] Search failed:', error);
                });
        } catch (error) {
            console.error('[map.js] Error triggering search:', error);
        }
    }, 1000); // 1 second debounce
}

function updateMarkersFromData(ads) {
    console.log('[map.js] Updating markers from API data:', ads.length, 'ads');
    
    // Create a set of new ad IDs for quick lookup
    const newAdIds = new Set();
    const newAdData = new Map();
    
    ads.forEach(ad => {
        if (ad.latitude && ad.longitude) {
            const lat = parseFloat(ad.latitude);
            const lon = parseFloat(ad.longitude);
            
            if (!isNaN(lat) && !isNaN(lon) && lat !== 0 && lon !== 0) {
                newAdIds.add(ad.id);
                newAdData.set(ad.id, { ad, lat, lon });
            }
        }
    });
    
    // Remove markers that are no longer in the results
    // But keep existing markers that are still in the new results (smoother UX)
    markers = markers.filter(marker => {
        const adId = marker._adId;
        if (adId && !newAdIds.has(adId)) {
            // This marker is no longer in results, remove it
            map.removeLayer(marker);
            console.log(`[map.js] Removed marker for ad ${adId} (no longer in results)`);
            return false;
        }
        return true;
    });
    
    // Add new markers that don't already exist
    const existingAdIds = new Set(markers.map(m => m._adId).filter(id => id));
    
    newAdData.forEach(({ ad, lat, lon }, adId) => {
        if (!existingAdIds.has(adId)) {
            // This is a new marker, add it
            const marker = L.marker([lat, lon])
                .bindPopup(`
                    <div class="popup-content">
                        <h3><a href="/ad/${ad.id}">${ad.title}</a></h3>
                        <p>€${ad.price}</p>
                        <p>${ad.location || ''}</p>
                    </div>
                `)
                .on('click', () => viewAd(ad.id));
            
            // Store the ad ID on the marker for tracking
            marker._adId = adId;
            
            marker.addTo(map);
            markers.push(marker);
            console.log(`[map.js] Added new marker for ad ${adId} at ${lat}, ${lon}`);
        }
    });
    
    console.log('[map.js] Updated markers smoothly, total:', markers.length);
    // Never call fitBounds here - user controls the view
}

function viewAd(adId) {
    // Navigate to ad detail page
    window.location.href = `/ad/${adId}`;
}

// fitMapToMarkers function removed - user controls map view

// Helper function to check if we should initialize map
function shouldInitMap() {
    const mapContainer = document.getElementById('map-container');
    const mapView = document.getElementById('map-view');
    const shouldInit = mapContainer && mapView && !mapInitialized && !isInitializing;
    
    // Only log if we're actually going to initialize or if this is a retry attempt
    if (shouldInit) {
        console.log('[map.js] Conditions met, ready to initialize map');
    }
    
    return shouldInit;
}

// Helper function to attempt map initialization with retries
function attemptMapInit() {
    if (shouldInitMap()) {
        console.log('[map.js] Initializing map...');
        setTimeout(initMap, 50); // Small delay to ensure DOM is ready
    }
    // Removed the else clause that was causing console spam
}

// Initialize map when DOM is ready
document.addEventListener('htmx:load', function() {
    // Only log if we're in map view
    if (document.getElementById('map-container')) {
        console.log('[map.js] htmx:load event fired, map container found');
        attemptMapInit();
    }
});

// No HTMX interference - map is managed purely by JavaScript

// Also try to initialize on regular DOMContentLoaded
document.addEventListener('DOMContentLoaded', function() {
    // Only log if we're in map view
    if (document.getElementById('map-container')) {
        console.log('[map.js] DOMContentLoaded event fired, map container found');
        attemptMapInit();
    }
});

// Additional initialization for map view specifically
document.addEventListener('htmx:afterRequest', function(event) {
    // Only attempt initialization if we're likely in map view
    if (document.getElementById('map-container')) {
        console.log('[map.js] htmx:afterRequest event fired, map container found');
        setTimeout(attemptMapInit, 200); // Longer delay for after request
    }
});

// Listen for map view button clicks specifically
document.addEventListener('click', function(event) {
    // Check if the clicked element is the map view button
    if (event.target && event.target.closest && event.target.closest('[hx-vals*="map"]')) {
        console.log('[map.js] Map view button clicked, will attempt initialization after swap');
        // The actual initialization will happen in htmx:afterSwap
    }
});

// Removed the interval-based initialization that was causing console spam
// The map will now only initialize when the user actually switches to map view
// or when the page loads with map view already active 