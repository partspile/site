// map.js

let mapInstance = null;
let isFirstDataLoad = true;

function initMap(savedBounds = null) {
    // Initialize the map once with reasonable bounds
    mapInstance = L.map('map-container');
    
    if (savedBounds && savedBounds.minLat && savedBounds.maxLat && savedBounds.minLon && savedBounds.maxLon) {
      // Use saved bounds
      const bounds = L.latLngBounds(
        L.latLng(savedBounds.minLat, savedBounds.minLon), // Southwest corner
        L.latLng(savedBounds.maxLat, savedBounds.maxLon)   // Northeast corner
      );
      mapInstance.fitBounds(bounds);
    } else {
      // Default to Kansas bounds (central US)
      const kansasBounds = L.latLngBounds(
        L.latLng(36.9931, -102.0517), // Southwest corner
        L.latLng(40.0016, -94.5882)   // Northeast corner
      );
      mapInstance.fitBounds(kansasBounds);
    }
  
    // Add OpenStreetMap tile layer
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
    }).addTo(mapInstance);
  
    // Debounce timer for map updates
    let timeout;
    let isUserInteraction = false;
  
    // Track user interactions
    mapInstance.on('dragstart zoomstart', function() {
      isUserInteraction = true;
    });
    
    // Listen for user interactions (panning and zooming)
    mapInstance.on('moveend', function() {
      if (isUserInteraction) {
        clearTimeout(timeout);
        timeout = setTimeout(updateMapData, 1000);
        isUserInteraction = false; // Reset flag
      }
    });
  
    function updateMapData() {
      const bounds = mapInstance.getBounds();
      const minLat = bounds.getSouth();
      const maxLat = bounds.getNorth();
      const minLon = bounds.getWest();
      const maxLon = bounds.getEast();
  
      // Trigger htmx ajax to update #map-data using search-page with map view
      const url = `/search-page?view=map&minLat=${minLat}&maxLat=${maxLat}&minLon=${minLon}&maxLon=${maxLon}`;
      htmx.ajax('GET', url, {target: '#map-data', swap: 'innerHTML'});
    }
    
    // Load initial data
    updateMapData();
  }
  
  function updateMapMarkers() {
    if (!mapInstance) return;
    
    // Get current markers and their ad IDs
    const existingMarkers = new Map();
    mapInstance.eachLayer(function(layer) {
      if (layer instanceof L.Marker && layer.adId) {
        existingMarkers.set(layer.adId, layer);
      }
    });
    
    // Get new ad data
    const adElements = document.querySelectorAll('#map-data [data-ad-id]');
    const newAdIds = new Set();
    const points = [];
    
    adElements.forEach(el => {
      const adId = el.getAttribute('data-ad-id');
      const lat = parseFloat(el.getAttribute('data-lat'));
      const lon = parseFloat(el.getAttribute('data-lon'));
      
      if (!isNaN(lat) && !isNaN(lon)) {
        newAdIds.add(adId);
        points.push([lat, lon]);
        
        // Only create marker if it doesn't already exist
        if (!existingMarkers.has(adId)) {
          const marker = L.marker([lat, lon]).addTo(mapInstance);
          marker.adId = adId; // Store ad ID for future reference
          
          // Get image URL from data attribute
          const imageURL = el.getAttribute('data-image');
          
          // Create popup content with image
          let popupContent = `<b>${el.getAttribute('data-title')}</b><br>Price: $${el.getAttribute('data-price')}`;
          
          // Add image if available
          if (imageURL && imageURL.trim() !== '') {
            popupContent = `
              <div style="text-align: center;">
                <img src="${imageURL}" alt="${el.getAttribute('data-title')}" 
                     style="width: 120px; height: 120px; object-fit: cover; border-radius: 4px; margin-bottom: 8px;">
                <br>
                <b>${el.getAttribute('data-title')}</b><br>
                Price: $${el.getAttribute('data-price')}
              </div>
            `;
          }
          
          marker.bindPopup(popupContent);
        }
      }
    });
    
    // Remove markers for ads that are no longer in the current view
    existingMarkers.forEach((marker, adId) => {
      if (!newAdIds.has(adId)) {
        mapInstance.removeLayer(marker);
      }
    });
    
    // Fit bounds only on first data load to show all markers
    if (points.length > 0 && isFirstDataLoad) {
      const bounds = L.latLngBounds(points);
      mapInstance.fitBounds(bounds);
      isFirstDataLoad = false; // Mark that we've done the initial fit
    }
  }

  // Listen for map data updates
  htmx.on('htmx:afterSwap', function(evt) {
    // Handle map-data updates
    if (evt.detail.target && evt.detail.target.id === 'map-data') {
      updateMapMarkers();
    }
  });