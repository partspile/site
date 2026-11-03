// account-picture.js: JPEG encode + MinIO presigned PUT for profile picture

const ACCOUNT_PICTURE_MAX_WIDTH = 1200;
const ACCOUNT_PICTURE_QUALITY = 0.8;

function accountPictureCsrfToken() {
	const el = document.querySelector('[hx-headers]');
	if (!el) {
		return '';
	}
	try {
		const headers = JSON.parse(el.getAttribute('hx-headers'));
		return headers['X-Csrf-Token'] || '';
	} catch (e) {
		return '';
	}
}

function setAccountPictureStatus(msg) {
	const el = document.getElementById('account-picture-status');
	if (!el) {
		return;
	}
	if (!msg) {
		el.textContent = '';
		el.classList.add('hidden');
		return;
	}
	el.textContent = msg;
	el.classList.remove('hidden');
}

function showAccountPictureError(msg) {
	const el = document.getElementById('account-picture-error');
	if (!el) {
		alert(msg);
		return;
	}
	el.textContent = msg;
	el.classList.remove('hidden');
}

function clearAccountPictureError() {
	const el = document.getElementById('account-picture-error');
	if (!el) {
		return;
	}
	el.textContent = '';
	el.classList.add('hidden');
}

async function loadAccountPictureBitmap(file) {
	if (typeof createImageBitmap === 'function') {
		return createImageBitmap(file);
	}
	return new Promise((resolve, reject) => {
		const url = URL.createObjectURL(file);
		const img = new Image();
		img.onload = () => {
			URL.revokeObjectURL(url);
			resolve(img);
		};
		img.onerror = () => {
			URL.revokeObjectURL(url);
			reject(new Error('Failed to decode image'));
		};
		img.src = url;
	});
}

function encodeAccountPicture(source, maxWidth, quality) {
	const sw = source.width || source.naturalWidth;
	const sh = source.height || source.naturalHeight;
	const scale = sw > maxWidth ? maxWidth / sw : 1;
	const w = Math.max(1, Math.round(sw * scale));
	const h = Math.max(1, Math.round(sh * scale));
	const canvas = document.createElement('canvas');
	canvas.width = w;
	canvas.height = h;
	const ctx = canvas.getContext('2d', { alpha: false });
	ctx.drawImage(source, 0, 0, w, h);
	return new Promise((resolve, reject) => {
		canvas.toBlob((blob) => {
			if (!blob || blob.type !== 'image/jpeg') {
				reject(new Error('JPEG encode failed'));
				return;
			}
			resolve(blob);
		}, 'image/jpeg', quality);
	});
}

function putAccountPicture(url, blob) {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('PUT', url);
		xhr.setRequestHeader('Content-Type', 'image/jpeg');
		xhr.onload = () => {
			if (xhr.status >= 200 && xhr.status < 300) {
				resolve();
			} else {
				reject(new Error('Upload failed: ' + xhr.status));
			}
		};
		xhr.onerror = () => reject(new Error('Network error during upload'));
		xhr.send(blob);
	});
}

async function uploadAccountPicture(file) {
	clearAccountPictureError();
	setAccountPictureStatus('Processing image...');
	const bitmap = await loadAccountPictureBitmap(file);
	let blob;
	try {
		blob = await encodeAccountPicture(
			bitmap, ACCOUNT_PICTURE_MAX_WIDTH, ACCOUNT_PICTURE_QUALITY);
	} finally {
		if (bitmap.close) {
			bitmap.close();
		}
	}
	setAccountPictureStatus('Preparing upload...');
	const presignRes = await fetch(
		'/auth/user/profile/picture/presign', {
			method: 'POST',
			headers: {
				'Accept': 'application/json',
				'X-Csrf-Token': accountPictureCsrfToken(),
			},
			credentials: 'same-origin',
		});
	if (!presignRes.ok) {
		const data = await presignRes.json().catch(() => ({}));
		throw new Error(data.error || 'Failed to prepare upload');
	}
	const { putUrl } = await presignRes.json();
	if (!putUrl) {
		throw new Error('Missing upload URL');
	}

	setAccountPictureStatus('Uploading...');
	await putAccountPicture(putUrl, blob);

	setAccountPictureStatus('Finishing...');
	const confirmRes = await fetch(
		'/auth/user/profile/picture/confirm', {
			method: 'POST',
			headers: {
				'Accept': 'application/json',
				'X-Csrf-Token': accountPictureCsrfToken(),
			},
			credentials: 'same-origin',
		});
	if (!confirmRes.ok) {
		const data = await confirmRes.json().catch(() => ({}));
		throw new Error(data.error || 'Failed to confirm upload');
	}
	setAccountPictureStatus('Done');
	window.location.reload();
}

function bindAccountPictureUpload() {
	const btn = document.getElementById('account-picture-upload-btn');
	const input = document.getElementById('account-picture-file');
	if (!btn || !input || btn.dataset.bound === '1') {
		return;
	}
	btn.dataset.bound = '1';
	btn.addEventListener('click', () => {
		clearAccountPictureError();
		input.value = '';
		input.click();
	});
	input.addEventListener('change', async () => {
		if (!input.files || input.files.length === 0) {
			return;
		}
		btn.disabled = true;
		try {
			await uploadAccountPicture(input.files[0]);
		} catch (err) {
			console.error(err);
			setAccountPictureStatus('');
			showAccountPictureError(err.message || 'Upload failed');
			btn.disabled = false;
		}
	});
}

document.addEventListener('DOMContentLoaded', bindAccountPictureUpload);
document.body.addEventListener('htmx:afterSwap', (e) => {
	if (e.detail && e.detail.target &&
		e.detail.target.id === 'account-picture-section') {
		bindAccountPictureUpload();
	}
});
