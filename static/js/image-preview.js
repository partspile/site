// Handles image previews, deletion, and drag-and-drop reordering for ad create/edit forms
(function() {
  // Utility to create a thumbnail with a trashcan overlay and drag handle
  function createThumbnail(file, idx, onDelete, onDragStart, onDragOver, onDrop, isDragging) {
    const wrapper = document.createElement('div');
    wrapper.className = 'relative inline-block m-1 group';
    wrapper.style.width = '96px';
    wrapper.style.height = '96px';
    wrapper.setAttribute('draggable', 'true');
    wrapper.dataset.idx = idx;
    if (isDragging) wrapper.style.opacity = '0.5';

    wrapper.addEventListener('dragstart', onDragStart);
    wrapper.addEventListener('dragover', onDragOver);
    wrapper.addEventListener('drop', onDrop);
    wrapper.addEventListener('dragend', function() {
      wrapper.style.opacity = '';
    });

    const img = document.createElement('img');
    img.className = 'object-cover w-24 h-24 rounded border';
    img.style.display = 'block';
    img.style.width = '96px';
    img.style.height = '96px';
    img.alt = file.name;
    img.src = URL.createObjectURL(file);
    wrapper.appendChild(img);

    const trash = document.createElement('img');
    trash.src = '/images/trashcan.svg';
    trash.alt = 'Delete';
    trash.className = 'absolute top-0 right-0 w-6 h-6 p-1 bg-white rounded-full shadow cursor-pointer opacity-0 group-hover:opacity-100 transition-opacity';
    trash.style.zIndex = 10;
    trash.title = 'Remove image';
    trash.addEventListener('click', function(e) {
      e.stopPropagation();
      e.preventDefault();
      onDelete(idx);
    });
    wrapper.appendChild(trash);

    return wrapper;
  }

  // Main logic for handling file input, preview, and order
  function setupImagePreview(inputId, previewId, orderInputId) {
    const input = document.getElementById(inputId);
    if (!input) return;
    let files = [];
    let preview = document.getElementById(previewId);
    if (!preview) {
      preview = document.createElement('div');
      preview.id = previewId;
      preview.className = 'flex flex-row flex-wrap gap-2 mt-2';
      input.parentNode.appendChild(preview);
    }
    let orderInput = document.getElementById(orderInputId);
    if (!orderInput) {
      orderInput = document.createElement('input');
      orderInput.type = 'hidden';
      orderInput.name = 'image_order';
      orderInput.id = orderInputId;
      input.form.appendChild(orderInput);
    }
    let draggingIdx = null;

    function updateOrderInput() {
      // Store the order as contiguous 1-based indices
      const order = files.map((_, i) => i + 1);
      orderInput.value = order.join(',');
    }

    function renderPreviews() {
      preview.innerHTML = '';
      files.forEach((file, idx) => {
        const thumb = createThumbnail(
          file,
          idx,
          (removeIdx) => {
            files.splice(removeIdx, 1);
            updateInputFiles();
            renderPreviews();
          },
          function(e) { // dragstart
            draggingIdx = idx;
            e.dataTransfer.effectAllowed = 'move';
          },
          function(e) { // dragover
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';
          },
          function(e) { // drop
            e.preventDefault();
            if (draggingIdx === null || draggingIdx === idx) return;
            const dragged = files[draggingIdx];
            files.splice(draggingIdx, 1);
            files.splice(idx, 0, dragged);
            updateInputFiles();
            renderPreviews();
            draggingIdx = null;
          },
          draggingIdx === idx
        );
        preview.appendChild(thumb);
      });
      updateOrderInput();
    }

    function updateInputFiles() {
      // Create a new DataTransfer to update the input's files
      const dt = new DataTransfer();
      files.forEach(f => dt.items.add(f));
      input.files = dt.files;
    }

    input.addEventListener('change', function(e) {
      // Instead of replacing, append new files (if any)
      const newFiles = Array.from(input.files);
      // Only add files that are not already in the list (by name+size)
      newFiles.forEach((f, i) => {
        if (!files.some(existing => existing.name === f.name && existing.size === f.size)) {
          // Store original index for order
          f._origIdx = files.length + i;
          files.push(f);
        }
      });
      // Assign _origIdx for all files if not present (for initial load)
      files.forEach((f, i) => { if (f._origIdx === undefined) f._origIdx = i; });
      updateInputFiles();
      renderPreviews();
    });
  }

  // Setup for both create and edit ad forms
  document.addEventListener('DOMContentLoaded', function() {
    setupImagePreview('images', 'image-preview', 'image_order');
  });
})(); 