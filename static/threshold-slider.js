// Threshold slider functionality
function updateThreshold(value) {
    // Update the displayed value
    const valueDisplay = document.getElementById('threshold-value');
    if (valueDisplay) {
        valueDisplay.textContent = value;
    }
    
    // Get current search parameters
    const searchBox = document.getElementById('searchBox');
    const viewInput = document.getElementById('view-type-input');
    const query = searchBox ? searchBox.value : '';
    const view = viewInput ? viewInput.value : 'list';
    
    // Only trigger search if there's a query
    if (query.trim() !== '') {
        // Build the search URL with the new threshold
        const searchUrl = `/search?q=${encodeURIComponent(query)}&view=${encodeURIComponent(view)}&threshold=${value}`;
        
        // Trigger the search with HTMX
        htmx.ajax('GET', searchUrl, {
            target: '#searchResults',
            swap: 'outerHTML'
        });
    }
}

// Initialize threshold slider when page loads
document.addEventListener('DOMContentLoaded', function() {
    const slider = document.getElementById('threshold-slider');
    const valueDisplay = document.getElementById('threshold-value');
    
    if (slider && valueDisplay) {
        // Set initial value from URL parameter or default
        const urlParams = new URLSearchParams(window.location.search);
        const threshold = urlParams.get('threshold') || '0.7';
        slider.value = threshold;
        valueDisplay.textContent = threshold;
    }
}); 