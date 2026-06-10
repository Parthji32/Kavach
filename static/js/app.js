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

    // Notification dropdown toggle
    window.toggleNotifications = function() {
        var menu = document.getElementById('notification-menu');
        if (menu) {
            menu.classList.toggle('hidden');
        }
    };

    // Close notification dropdown when clicking outside
    document.addEventListener('click', function(e) {
        var wrapper = document.getElementById('notification-wrapper');
        var menu = document.getElementById('notification-menu');
        if (wrapper && menu && !wrapper.contains(e.target)) {
            menu.classList.add('hidden');
        }
    });

    // Close notification dropdown on Escape key
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            var menu = document.getElementById('notification-menu');
            if (menu) menu.classList.add('hidden');
        }
    });

    // QR Code display handler
    // When a QR token is created, render the QR image from the payload
    window.renderQRCodePayload = function(containerEl, payload) {
        if (!containerEl || !payload) return;

        if (payload.startsWith('data:image')) {
            var img = document.createElement('img');
            img.src = payload;
            img.alt = 'QR Code Token';
            img.className = 'w-48 h-48 rounded-lg border border-kavach-border bg-white p-2 mx-auto';
            containerEl.innerHTML = '';
            containerEl.appendChild(img);
        }
    };

    // Generic base64 image renderer
    // Checks if a token payload is a base64 image and renders it inline
    window.renderBase64Image = function(containerEl, payload, options) {
        if (!containerEl || !payload) return false;

        if (!payload.startsWith('data:image')) return false;

        var opts = options || {};
        var img = document.createElement('img');
        img.src = payload;
        img.alt = opts.alt || 'Token Image';
        img.className = opts.className || 'max-w-full rounded-lg border border-kavach-border bg-white p-2';

        if (opts.replace) {
            containerEl.innerHTML = '';
        }
        containerEl.appendChild(img);
        return true;
    };

    // After HTMX swap, check if any token payload is a base64 image and render it
    document.addEventListener('htmx:afterSwap', function(event) {
        var payloadEls = event.detail.target.querySelectorAll('[data-token-payload]');
        payloadEls.forEach(function(el) {
            var payload = el.getAttribute('data-token-payload');
            window.renderBase64Image(el, payload, { replace: true, alt: 'Token QR Code' });
        });
    });

    console.log('[Kavach] Client initialized');
})();
