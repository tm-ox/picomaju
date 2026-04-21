// Apply saved theme before first paint (synchronous, no defer)
(function () {
    var t = localStorage.getItem('theme') ||
        (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    document.documentElement.setAttribute('data-theme', t);
})();

// Restore sidebar collapsed state after DOM is ready
document.addEventListener('DOMContentLoaded', function () {
    if (localStorage.getItem('sidebar-collapsed') === '1') {
        var s = document.getElementById('sidebar');
        if (s) s.classList.add('collapsed');
        var btn = s ? s.querySelector('.sidebar-toggle') : null;
        if (btn) {
            btn.setAttribute('aria-expanded', 'false');
            btn.setAttribute('aria-label', 'Expand sidebar');
        }
    }
});
