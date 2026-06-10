// Kavach - Client-side JavaScript
// Minimal JS for non-HTMX interactions

(function() {
    'use strict';

    // Copy to clipboard
    window.copyToClipboard = function(btn, text) {
        navigator.clipboard.writeText(text).then(function() {
            var originalHTML = btn.innerHTML;
            btn.innerHTML = '<svg class="w-4 h-4 text-kavach-accent" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>';
            setTimeout(function() {
                btn.innerHTML = originalHTML;
            }, 2000);
        }).catch(function(err) {
            console.error('Copy failed:', err);
        });
    };

    // Theme toggle (future use)
    window.toggleTheme = function() {
        document.body.classList.toggle('light-mode');
        localStorage.setItem('theme', document.body.classList.contains('light-mode') ? 'light' : 'dark');
    };

    // Initialize theme from localStorage
    if (localStorage.getItem('theme') === 'light') {
        document.body.classList.add('light-mode');
    }

    // HTMX event listeners
    document.addEventListener('htmx:responseError', function(event) {
        var target = event.detail.target;
        if (target) {
            target.innerHTML = '<div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-sm text-red-400">Request failed. Please try again.</div>';
        }
    });

    // Auto-dismiss notifications after 5s
    document.addEventListener('htmx:afterSwap', function(event) {
        var notifications = event.detail.target.querySelectorAll('[data-autodismiss]');
        notifications.forEach(function(el) {
            setTimeout(function() {
                el.style.opacity = '0';
                el.style.transition = 'opacity 0.3s';
                setTimeout(function() { el.remove(); }, 300);
            }, 5000);
        });
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Ctrl/Cmd + K = focus search (future)
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            var search = document.getElementById('global-search');
            if (search) search.focus();
        }
    });

    console.log('[Kavach] Client initialized');
})();
