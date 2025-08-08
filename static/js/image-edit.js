// image-edit.js: Handles drag/drop reordering and deletion of ad images on the edit page

document.addEventListener('DOMContentLoaded', function () {
  const gallery = document.getElementById('image-gallery');
  const orderInput = document.getElementById('image_order');
  const deletedInput = document.getElementById('deleted_images');

  if (!gallery) return;

  // Drag and drop reordering
  let dragged = null;

  gallery.addEventListener('dragstart', function (e) {
    if (e.target.closest('[data-image-idx]')) {
      dragged = e.target.closest('[data-image-idx]');
      e.dataTransfer.effectAllowed = 'move';
    }
  });

  gallery.addEventListener('dragover', function (e) {
    e.preventDefault();
    const over = e.target.closest('[data-image-idx]');
    if (dragged && over && dragged !== over) {
      over.style.border = '2px dashed #888';
    }
  });

  gallery.addEventListener('dragleave', function (e) {
    const over = e.target.closest('[data-image-idx]');
    if (over) {
      over.style.border = '';
    }
  });

  gallery.addEventListener('drop', function (e) {
    e.preventDefault();
    const over = e.target.closest('[data-image-idx]');
    if (dragged && over && dragged !== over) {
      over.style.border = '';
      // Move dragged before over
      gallery.insertBefore(dragged, over);
      updateOrderInput();
    }
  });

  gallery.addEventListener('dragend', function () {
    Array.from(gallery.children).forEach(el => {
      el.style.border = '';
    });
  });

  // Delete image
  gallery.addEventListener('click', function (e) {
    if (e.target.closest('.delete-image-btn')) {
      const thumb = e.target.closest('[data-image-idx]');
      if (thumb) {
        const idx = thumb.getAttribute('data-image-idx');
        // Add to deleted list
        let deleted = deletedInput.value ? deletedInput.value.split(',') : [];
        if (!deleted.includes(idx)) {
          deleted.push(idx);
          deletedInput.value = deleted.filter(Boolean).join(',');
        }
        // Remove from DOM
        thumb.remove();
        updateOrderInput();
      }
    }
  });

  function updateOrderInput() {
    // Update the order input to reflect current DOM order
    const order = Array.from(gallery.querySelectorAll('[data-image-idx]'))
      .map(el => el.getAttribute('data-image-idx'))
      .filter(Boolean)
      .join(',');
    orderInput.value = order;
  }
}); 