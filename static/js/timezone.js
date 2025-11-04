(function() {
	var timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	if (!document.cookie.includes('timezone=')) {
		document.cookie = 'timezone=' + encodeURIComponent(timezone) + '; path=/; max-age=' + (365 * 24 * 60 * 60) + '; SameSite=Strict' + (location.protocol === 'https:' ? '; Secure' : '');
	}
})();

