// map.js

let mapInstance = null;
let isFirstDataLoad = true;

function initMap() {
    // Initialize the map once with reasonable bounds
    mapInstance = L.map('map-container');
    
    // Set reasonable initial bounds (North America)
    const initialBounds = L.latLngBounds(
      L.latLng(24.0, -125.0), // Southwest corner
      L.latLng(49.0, -66.0)   // Northeast corner
    );
    mapInstance.fitBounds(initialBounds);
  
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
    
    // Clear existing markers
    mapInstance.eachLayer(function(layer) {
      if (layer instanceof L.Marker) {
        mapInstance.removeLayer(layer);
      }
    });
    
    // Add new markers from updated data
    const adElements = document.querySelectorAll('#map-data [data-ad-id]');
    const points = [];
    adElements.forEach(el => {
      const lat = parseFloat(el.getAttribute('data-lat'));
      const lon = parseFloat(el.getAttribute('data-lon'));
      if (!isNaN(lat) && !isNaN(lon)) {
        points.push([lat, lon]);
        L.marker([lat, lon]).addTo(mapInstance).bindPopup(
          `<b>${el.getAttribute('data-title')}</b><br>Price: $${el.getAttribute('data-price')}`
        );
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