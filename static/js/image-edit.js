// image-edit.js: drag/drop reordering of existing ad images on edit page

document.addEventListener('DOMContentLoaded', function () {
	const gallery = document.getElementById('image-gallery');
	const orderInput = document.getElementById('image_order');

	if (!gallery || !orderInput) {
		return;
	}

	let dragged = null;

	gallery.addEventListener('dragstart', function (e) {
		const thumb = e.target.closest('[data-image-idx]');
		if (!thumb) {
			return;
		}
		dragged = thumb;
		e.dataTransfer.effectAllowed = 'move';
		thumb.style.opacity = '0.5';
	});

	gallery.addEventListener('dragover', function (e) {
		e.preventDefault();
		const over = e.target.closest('[data-image-idx]');
		if (dragged && over && dragged !== over) {
			over.style.outline = '2px dashed #888';
		}
	});

	gallery.addEventListener('dragleave', function (e) {
		const over = e.target.closest('[data-image-idx]');
		if (over) {
			over.style.outline = '';
		}
	});

	gallery.addEventListener('drop', function (e) {
		e.preventDefault();
		const over = e.target.closest('[data-image-idx]');
		if (dragged && over && dragged !== over) {
			over.style.outline = '';
			const all = Array.from(gallery.children);
			const draggedIndex = all.indexOf(dragged);
			const overIndex = all.indexOf(over);
			if (draggedIndex < overIndex) {
				gallery.insertBefore(dragged, over.nextSibling);
			} else {
				gallery.insertBefore(dragged, over);
			}
			updateOrderInput();
		}
	});

	gallery.addEventListener('dragend', function () {
		Array.from(gallery.children).forEach(function (el) {
			el.style.outline = '';
			el.style.opacity = '';
		});
		dragged = null;
	});

	function updateOrderInput() {
		const order = Array.from(gallery.querySelectorAll('[data-image-idx]'))
			.map(function (el) {
				return el.getAttribute('data-image-idx');
			})
			.filter(Boolean)
			.join(',');
		orderInput.value = order;
	}
});
