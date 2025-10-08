// image-preview.js: Clean image upload system using DOM as single source of truth
let draggedElement = null;
let dragAndDropSetup = false;

function previewImages(input) {
	const preview = document.getElementById('image-preview');
	
	// Add new files to the DOM
	if (input.files && input.files.length > 0) {
		Array.from(input.files).forEach(file => {
			// Check if file is already in our collection (by name and size)
			const isDuplicate = Array.from(preview.children).some(thumbnail => 
				thumbnail.fileReference.name === file.name && thumbnail.fileReference.size === file.size
			);
			if (!isDuplicate) {
				addThumbnailToDOM(preview, file);
			}
		});
	}
	
	// Set up drag and drop only once
	if (!dragAndDropSetup) {
		setupDragAndDrop(preview);
		dragAndDropSetup = true;
	}
	
	// Update upload area text and file input
	updateUploadAreaText();
	updateFileInputFromDOM();
	toggleUploadContent();
}

function addThumbnailToDOM(preview, file) {
	const thumbnail = document.createElement('div');
	thumbnail.className = 'relative inline-block m-1 group cursor-move';
	thumbnail.style.width = '96px';
	thumbnail.style.height = '96px';
	thumbnail.setAttribute('draggable', 'true');
	thumbnail.fileReference = file; // Store file reference directly
	
	// Prevent click events from bubbling up to the upload area
	thumbnail.onclick = function(event) {
		event.stopPropagation();
	};
	
	const reader = new FileReader();
	reader.onload = function(e) {
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
		deleteBtn.onclick = function(event) {
			event.stopPropagation();
			thumbnail.remove();
			updateUploadAreaText();
			updateFileInputFromDOM();
			toggleUploadContent();
		};
		
		// Add drag handle indicator
		const dragHandle = document.createElement('div');
		dragHandle.className = 'absolute bottom-0 left-0 w-6 h-6 bg-gray-600 bg-opacity-75 text-white text-xs flex items-center justify-center rounded-tl';
		dragHandle.innerHTML = '⋮⋮';
		dragHandle.style.fontSize = '8px';
		dragHandle.style.lineHeight = '1';
		dragHandle.onclick = function(event) {
			event.stopPropagation();
		};
		
		thumbnail.appendChild(img);
		thumbnail.appendChild(deleteBtn);
		thumbnail.appendChild(dragHandle);
		preview.appendChild(thumbnail);
		
		// Update after adding
		updateUploadAreaText();
		updateFileInputFromDOM();
		toggleUploadContent();
	};
	reader.readAsDataURL(file);
}

function updateFileInputFromDOM() {
	const preview = document.getElementById('image-preview');
	const input = document.getElementById('images');
	const files = Array.from(preview.children).map(thumbnail => thumbnail.fileReference);
	
	const dt = new DataTransfer();
	files.forEach(file => {
		dt.items.add(file);
	});
	input.files = dt.files;
}

function setupDragAndDrop(preview) {
	// Add event delegation to the preview container
	preview.addEventListener('dragstart', function(e) {
		if (e.target.closest('[draggable="true"]')) {
			draggedElement = e.target.closest('[draggable="true"]');
			draggedElement.style.opacity = '0.5';
			e.dataTransfer.effectAllowed = 'move';
		}
	});
	
	preview.addEventListener('dragover', function(e) {
		e.preventDefault();
		const over = e.target.closest('[draggable="true"]');
		if (draggedElement && over && draggedElement !== over) {
			over.style.border = '2px dashed #3b82f6';
			over.style.backgroundColor = '#eff6ff';
		}
	});
	
	preview.addEventListener('dragleave', function(e) {
		const over = e.target.closest('[draggable="true"]');
		if (over) {
			over.style.border = '';
			over.style.backgroundColor = '';
		}
	});
	
	preview.addEventListener('drop', function(e) {
		e.preventDefault();
		const over = e.target.closest('[draggable="true"]');
		if (draggedElement && over && draggedElement !== over) {
			over.style.border = '';
			over.style.backgroundColor = '';
			
			// Move the dragged element before the drop target in the DOM
			preview.insertBefore(draggedElement, over);
			
			// Update the file input to reflect the new order
			updateFileInputFromDOM();
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

function handleDrop(event) {
	const files = Array.from(event.dataTransfer.files).filter(file => 
		file.type.startsWith('image/')
	);
	
	if (files.length > 0) {
		const preview = document.getElementById('image-preview');
		files.forEach(file => {
			// Check if file is already in our collection (by name and size)
			const isDuplicate = Array.from(preview.children).some(thumbnail => 
				thumbnail.fileReference.name === file.name && thumbnail.fileReference.size === file.size
			);
			if (!isDuplicate) {
				addThumbnailToDOM(preview, file);
			}
		});
	}
}

function toggleUploadContent() {
	const uploadContent = document.getElementById('upload-content');
	const imagePreview = document.getElementById('image-preview');
	const preview = document.getElementById('image-preview');
	const fileCount = preview.children.length;
	
	if (fileCount === 0) {
		// Show upload content, hide preview
		uploadContent.classList.remove('hidden');
		imagePreview.classList.add('hidden');
	} else {
		// Show both upload content and preview when images are present
		uploadContent.classList.remove('hidden');
		imagePreview.classList.remove('hidden');
		
		// Update the upload content text to indicate more images can be added
		const titleElement = uploadContent.querySelector('.text-lg');
		const subtitleElement = uploadContent.querySelector('.text-sm');
		
		titleElement.textContent = 'Add More Images';
		subtitleElement.textContent = 'Click to browse or drag and drop';
	}
}

function updateUploadAreaText() {
	const uploadContent = document.getElementById('upload-content');
	if (uploadContent) {
		const titleElement = uploadContent.querySelector('.text-lg');
		const subtitleElement = uploadContent.querySelector('.text-sm');
		
		const preview = document.getElementById('image-preview');
		const fileCount = preview.children.length;
		
		if (fileCount === 0) {
			titleElement.textContent = 'Upload Images';
			subtitleElement.textContent = 'Click to browse or drag and drop';
		} else {
			titleElement.textContent = 'Add More Images';
			subtitleElement.textContent = 'Click to browse or drag and drop';
		}
	}
}