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
          
          // Create popup content with image and click handler
          let popupContent = `<b>${el.getAttribute('data-title')}</b><br>Price: $${el.getAttribute('data-price')}`;
          
          // Add image if available
          if (imageURL && imageURL.trim() !== '') {
            popupContent = `
              <div onclick="loadAdDetail(${adId})" style="text-align: center; cursor: pointer; padding: 4px;">
                <img src="${imageURL}" alt="${el.getAttribute('data-title')}" 
                     style="width: 120px; height: 120px; object-fit: cover; border-radius: 4px; margin-bottom: 8px;">
                <br>
                <b>${el.getAttribute('data-title')}</b><br>
                Price: $${el.getAttribute('data-price')}
                <br>
              </div>
            `;
          } else {
            popupContent = `
              <div onclick="loadAdDetail(${adId})" style="text-align: center; cursor: pointer; padding: 8px;">
                <b>${el.getAttribute('data-title')}</b><br>
                Price: $${el.getAttribute('data-price')}
                <br>
                <small style="color: #666; font-style: italic;">Click to view details</small>
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

  // Function to load ad detail using HTMX
  function loadAdDetail(adId) {
    // Check if ad is already displayed using standard ad ID format
    const existingAd = document.getElementById(`ad-${adId}`);
    if (existingAd) {
      // Ad already exists, scroll to it and focus
      existingAd.scrollIntoView({ behavior: 'smooth', block: 'start' });
      existingAd.focus();
      return;
    }
    
    // Load ad detail using HTMX
    htmx.ajax('GET', `/ad/detail/${adId}?view=list`, {
      target: '#map-ad-details',
      swap: 'afterbegin'
    });
    
    // Add separator after the ad loads
    setTimeout(() => {
      const adElement = document.getElementById(`ad-${adId}`);
      if (adElement && !adElement.nextElementSibling?.classList.contains('map-ad-separator')) {
        const separator = document.createElement('div');
        separator.className = 'border-b border-gray-200 map-ad-separator';
        adElement.parentNode.insertBefore(separator, adElement.nextSibling);
      }
    }, 100);
  }

  // Listen for map data updates
  htmx.on('htmx:afterSwap', function(evt) {
    // Handle map-data updates
    if (evt.detail.target && evt.detail.target.id === 'map-data') {
      updateMapMarkers();
    }
  });