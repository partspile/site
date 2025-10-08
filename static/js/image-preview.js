// image-preview.js: Clean image upload system using DOM as single source of truth
let draggedElement = null;
let dragAndDropSetup = false;
let touchStartX = 0;
let touchStartY = 0;
let isDragging = false;
let pendingAdditions = 0; // Track pending async additions to prevent race conditions

function canAddMoreImages() {
	const preview = document.getElementById('image-preview');
	return (preview.children.length + pendingAdditions) < MAX_IMAGES_PER_AD;
}

function handleUploadClick() {
	if (canAddMoreImages()) {
		document.getElementById('images').click();
	}
}

function previewImages(input) {
	const preview = document.getElementById('image-preview');
	
	// Add new files to the DOM
	if (input.files && input.files.length > 0) {
		let addedCount = 0;
		let skippedCount = 0;
		Array.from(input.files).forEach(file => {
			// Check if we've reached the max limit (including pending additions)
			const currentCount = preview.children.length + pendingAdditions;
			if (currentCount >= MAX_IMAGES_PER_AD) {
				skippedCount++;
				return; // Skip adding more files
			}
			
			// Check if file is already in our collection (by name and size)
			const isDuplicate = Array.from(preview.children).some(thumbnail => 
				thumbnail.fileReference.name === file.name && thumbnail.fileReference.size === file.size
			);
			if (!isDuplicate) {
				pendingAdditions++; // Reserve a slot
				addThumbnailToDOM(preview, file);
				addedCount++;
			}
		});
		
		// Show warning if we hit the limit
		if (skippedCount > 0) {
			alert(`Maximum of ${MAX_IMAGES_PER_AD} images allowed per ad. ${skippedCount} image(s) were not added.`);
		}
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
	// Outer container for highlight background
	const container = document.createElement('div');
	container.className = 'inline-block cursor-move';
	container.style.margin = '2px';
	container.style.padding = '3px';
	container.style.borderRadius = '8px';
	container.setAttribute('draggable', 'true');
	container.fileReference = file; // Store file reference directly
	
	// Inner thumbnail container
	const thumbnail = document.createElement('div');
	thumbnail.className = 'relative';
	thumbnail.style.width = '90px';
	thumbnail.style.height = '90px';
	
	// Prevent click events from bubbling up to the upload area
	container.onclick = function(event) {
		event.stopPropagation();
	};
	
	const reader = new FileReader();
	reader.onload = function(e) {
		const img = document.createElement('img');
		img.className = 'object-cover rounded';
		img.style.display = 'block';
		img.style.width = '90px';
		img.style.height = '90px';
		img.style.borderRadius = '6px';
		img.src = e.target.result;
		img.alt = file.name;
		
		const deleteBtn = document.createElement('button');
		deleteBtn.className = 'absolute top-0 right-0 w-6 h-6 bg-red-500 text-white rounded-full text-xs hover:bg-red-600';
		deleteBtn.innerHTML = '×';
		deleteBtn.onclick = function(event) {
			event.stopPropagation();
			container.remove();
			updateUploadAreaText();
			updateFileInputFromDOM();
			toggleUploadContent();
		};
		
		thumbnail.appendChild(img);
		thumbnail.appendChild(deleteBtn);
		container.appendChild(thumbnail);
		preview.appendChild(container);
		
		pendingAdditions--; // Release the reserved slot now that it's added
		
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
			over.style.backgroundColor = '#22c55e';
		}
	});
	
	preview.addEventListener('dragleave', function(e) {
		const over = e.target.closest('[draggable="true"]');
		if (over) {
			over.style.backgroundColor = '';
		}
	});
	
	preview.addEventListener('drop', function(e) {
		e.preventDefault();
		const over = e.target.closest('[draggable="true"]');
		if (draggedElement && over && draggedElement !== over) {
			over.style.backgroundColor = '';
			
			// Determine if we're dragging forward or backward
			const allThumbnails = Array.from(preview.children);
			const draggedIndex = allThumbnails.indexOf(draggedElement);
			const overIndex = allThumbnails.indexOf(over);
			
			if (draggedIndex < overIndex) {
				// Dragging forward: insert after the drop target
				preview.insertBefore(draggedElement, over.nextSibling);
			} else {
				// Dragging backward: insert before the drop target
				preview.insertBefore(draggedElement, over);
			}
			
			// Update the file input to reflect the new order
			updateFileInputFromDOM();
		}
	});
	
	preview.addEventListener('dragend', function() {
		// Clean up any remaining drag styles
		Array.from(preview.children).forEach(el => {
			el.style.backgroundColor = '';
			el.style.opacity = '';
		});
		draggedElement = null;
	});
	
	// Touch support for mobile devices
	preview.addEventListener('touchstart', function(e) {
		const thumbnail = e.target.closest('[draggable="true"]');
		if (thumbnail) {
			draggedElement = thumbnail;
			const touch = e.touches[0];
			touchStartX = touch.clientX;
			touchStartY = touch.clientY;
			isDragging = false;
			
			// Add a slight delay before showing visual feedback
			setTimeout(() => {
				if (draggedElement) {
					draggedElement.style.opacity = '0.5';
				}
			}, 100);
		}
	}, { passive: false });
	
	preview.addEventListener('touchmove', function(e) {
		if (!draggedElement) return;
		
		const touch = e.touches[0];
		const deltaX = Math.abs(touch.clientX - touchStartX);
		const deltaY = Math.abs(touch.clientY - touchStartY);
		
		// Consider it a drag if moved more than 10px
		if (deltaX > 10 || deltaY > 10) {
			isDragging = true;
			e.preventDefault(); // Prevent scrolling while dragging
		}
		
		if (isDragging) {
			// Find element under touch point
			const elementAtPoint = document.elementFromPoint(touch.clientX, touch.clientY);
			const over = elementAtPoint?.closest('[draggable="true"]');
			
			// Clear all hover styles first
			Array.from(preview.children).forEach(el => {
				if (el !== draggedElement) {
					el.style.backgroundColor = '';
				}
			});
			
			// Add hover style to drop target
			if (over && over !== draggedElement) {
				over.style.backgroundColor = '#22c55e';
			}
		}
	}, { passive: false });
	
	preview.addEventListener('touchend', function(e) {
		if (!draggedElement) return;
		
		if (isDragging) {
			const touch = e.changedTouches[0];
			const elementAtPoint = document.elementFromPoint(touch.clientX, touch.clientY);
			const over = elementAtPoint?.closest('[draggable="true"]');
			
			if (over && over !== draggedElement) {
				// Determine if we're dragging forward or backward
				const allThumbnails = Array.from(preview.children);
				const draggedIndex = allThumbnails.indexOf(draggedElement);
				const overIndex = allThumbnails.indexOf(over);
				
				if (draggedIndex < overIndex) {
					// Dragging forward: insert after the drop target
					preview.insertBefore(draggedElement, over.nextSibling);
				} else {
					// Dragging backward: insert before the drop target
					preview.insertBefore(draggedElement, over);
				}
				
				// Update the file input to reflect the new order
				updateFileInputFromDOM();
			}
		}
		
		// Clean up
		Array.from(preview.children).forEach(el => {
			el.style.backgroundColor = '';
			el.style.opacity = '';
		});
		draggedElement = null;
		isDragging = false;
	}, { passive: false });
}

function handleDrop(event) {
	const files = Array.from(event.dataTransfer.files).filter(file => 
		file.type.startsWith('image/')
	);
	
	if (files.length > 0) {
		const preview = document.getElementById('image-preview');
		let addedCount = 0;
		let skippedCount = 0;
		files.forEach(file => {
			// Check if we've reached the max limit (including pending additions)
			const currentCount = preview.children.length + pendingAdditions;
			if (currentCount >= MAX_IMAGES_PER_AD) {
				skippedCount++;
				return; // Skip adding more files
			}
			
			// Check if file is already in our collection (by name and size)
			const isDuplicate = Array.from(preview.children).some(thumbnail => 
				thumbnail.fileReference.name === file.name && thumbnail.fileReference.size === file.size
			);
			if (!isDuplicate) {
				pendingAdditions++; // Reserve a slot
				addThumbnailToDOM(preview, file);
				addedCount++;
			}
		});
		
		// Show warning if we hit the limit
		if (skippedCount > 0) {
			alert(`Maximum of ${MAX_IMAGES_PER_AD} images allowed per ad. ${skippedCount} image(s) were not added.`);
		}
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
	}
}

function updateUploadAreaText() {
	const uploadContent = document.getElementById('upload-content');
	const uploadArea = document.getElementById('upload-area');
	if (uploadContent) {
		const titleElement = uploadContent.querySelector('.text-lg');
		const subtitleElement = uploadContent.querySelector('.text-sm');
		
		const preview = document.getElementById('image-preview');
		const fileCount = preview.children.length;
		const remaining = MAX_IMAGES_PER_AD - fileCount;
		
		if (fileCount === 0) {
			titleElement.textContent = 'Upload Images';
			subtitleElement.textContent = `Click to browse or drag and drop (up to ${MAX_IMAGES_PER_AD} images)`;
			uploadContent.classList.remove('opacity-50', 'cursor-not-allowed');
			uploadArea.style.cursor = 'pointer';
		} else if (fileCount >= MAX_IMAGES_PER_AD) {
			titleElement.textContent = `Maximum ${MAX_IMAGES_PER_AD} Images Reached`;
			subtitleElement.textContent = 'Delete an image to add more';
			uploadContent.classList.add('opacity-50', 'cursor-not-allowed');
			uploadArea.style.cursor = 'not-allowed';
		} else {
			titleElement.textContent = `Add More Images`;
			subtitleElement.textContent = `${fileCount} of ${MAX_IMAGES_PER_AD} (${remaining} remaining)`;
			uploadContent.classList.remove('opacity-50', 'cursor-not-allowed');
			uploadArea.style.cursor = 'pointer';
		}
	}
}