// image-preview.js: Handles image previews and file management for ad forms
let allSelectedFiles = [];
let draggedElement = null;
let dragAndDropSetup = false;

function previewImages(input) {
	const preview = document.getElementById('image-preview');
	
	// Add new files to our collection
	if (input.files && input.files.length > 0) {
		Array.from(input.files).forEach(file => {
			// Check if file is already in our collection (by name and size)
			const isDuplicate = allSelectedFiles.some(existing => 
				existing.name === file.name && existing.size === file.size
			);
			if (!isDuplicate) {
				allSelectedFiles.push(file);
			}
		});
	}
	
	// Update the file input to include all files
	updateFileInput(input);
	
	// Render all thumbnails
	renderThumbnails(preview);
	
	// Set up drag and drop only once
	if (!dragAndDropSetup) {
		setupDragAndDrop(preview);
		dragAndDropSetup = true;
	}
}

function updateFileInput(input) {
	const dt = new DataTransfer();
	allSelectedFiles.forEach(file => {
		dt.items.add(file);
	});
	input.files = dt.files;
}

function renderThumbnails(preview) {
	preview.innerHTML = '';
	preview.className = 'image-preview flex flex-row flex-wrap gap-2 mt-2';
	
	allSelectedFiles.forEach((file, index) => {
		const reader = new FileReader();
		reader.onload = function(e) {
			const thumbnail = document.createElement('div');
			thumbnail.className = 'relative inline-block m-1 group cursor-move';
			thumbnail.style.width = '96px';
			thumbnail.style.height = '96px';
			thumbnail.setAttribute('data-index', index);
			thumbnail.setAttribute('draggable', 'true');
			thumbnail.fileReference = file; // Store file reference directly
			
			const img = document.createElement('img');
			img.className = 'object-cover w-24 h-24 rounded border';
			img.style.display = 'block';
			img.style.width = '96px';
			img.style.height = '96px';
			img.src = e.target.result;
			img.alt = file.name;
			
			const deleteBtn = document.createElement('button');
			deleteBtn.className = 'absolute top-0 right-0 w-6 h-6 bg-red-500 text-white rounded-full text-xs hover:bg-red-600';
			deleteBtn.innerHTML = '×';
			deleteBtn.onclick = function() {
				removeImageFromCollection(index);
			};
			
			// Add drag handle indicator
			const dragHandle = document.createElement('div');
			dragHandle.className = 'absolute bottom-0 left-0 w-6 h-6 bg-gray-600 bg-opacity-75 text-white text-xs flex items-center justify-center rounded-tl';
			dragHandle.innerHTML = '⋮⋮';
			dragHandle.style.fontSize = '8px';
			dragHandle.style.lineHeight = '1';
			
			thumbnail.appendChild(img);
			thumbnail.appendChild(deleteBtn);
			thumbnail.appendChild(dragHandle);
			preview.appendChild(thumbnail);
		};
		reader.readAsDataURL(file);
	});
}

function setupDragAndDrop(preview) {
	// Add event delegation to the preview container
	preview.addEventListener('dragstart', function(e) {
		if (e.target.closest('[data-index]')) {
			draggedElement = e.target.closest('[data-index]');
			e.target.style.opacity = '0.5';
			e.dataTransfer.effectAllowed = 'move';
		}
	});
	
	preview.addEventListener('dragover', function(e) {
		e.preventDefault();
		const over = e.target.closest('[data-index]');
		if (draggedElement && over && draggedElement !== over) {
			over.style.border = '2px dashed #3b82f6';
			over.style.backgroundColor = '#eff6ff';
		}
	});
	
	preview.addEventListener('dragleave', function(e) {
		const over = e.target.closest('[data-index]');
		if (over) {
			over.style.border = '';
			over.style.backgroundColor = '';
		}
	});
	
	preview.addEventListener('drop', function(e) {
		e.preventDefault();
		const over = e.target.closest('[data-index]');
		if (draggedElement && over && draggedElement !== over) {
			over.style.border = '';
			over.style.backgroundColor = '';
			
			// Move the dragged element before the drop target in the DOM
			preview.insertBefore(draggedElement, over);
			
			// Update the files array based on the new DOM order
			updateFilesFromDOMOrder();
			
			// Update the file input
			const input = document.getElementById('images');
			updateFileInput(input);
		}
	});
	
	preview.addEventListener('dragend', function() {
		// Clean up any remaining drag styles
		Array.from(preview.children).forEach(el => {
			el.style.border = '';
			el.style.backgroundColor = '';
			el.style.opacity = '';
		});
		draggedElement = null;
	});
}

function updateFilesFromDOMOrder() {
	const preview = document.getElementById('image-preview');
	const thumbnails = Array.from(preview.children);
	allSelectedFiles = thumbnails.map(thumbnail => thumbnail.fileReference);
}

function removeImageFromCollection(index) {
	allSelectedFiles.splice(index, 1);
	const input = document.getElementById('images');
	updateFileInput(input);
	renderThumbnails(document.getElementById('image-preview'));
}