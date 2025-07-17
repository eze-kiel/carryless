// Global app functionality

// CSRF token management
let csrfToken = '';

async function fetchCSRFToken() {
    try {
        const response = await fetch('/api/csrf-token');
        if (response.ok) {
            const data = await response.json();
            csrfToken = data.token;
            return csrfToken;
        }
    } catch (error) {
        console.error('Failed to fetch CSRF token:', error);
    }
    return null;
}

// Generic modal management
function showModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'flex';
    }
}

function hideModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'none';
        // Reset form if it exists
        const form = modal.querySelector('form');
        if (form) {
            form.reset();
        }
    }
}

// Generic API request helper
async function apiRequest(url, options = {}) {
    if (!csrfToken && ['POST', 'PUT', 'DELETE'].includes(options.method)) {
        await fetchCSRFToken();
    }

    const defaultOptions = {
        headers: {
            'X-CSRF-Token': csrfToken,
            ...options.headers
        }
    };

    const response = await fetch(url, { ...defaultOptions, ...options });
    return response;
}

// Form submission helper
async function submitForm(form, url, method = 'POST') {
    const formData = new FormData(form);
    
    if (!csrfToken) {
        await fetchCSRFToken();
    }
    
    formData.append('csrf_token', csrfToken);

    try {
        const response = await fetch(url, {
            method: method,
            body: formData,
            headers: {
                'X-CSRF-Token': csrfToken
            }
        });

        return response;
    } catch (error) {
        throw new Error('Request failed: ' + error.message);
    }
}

// Toast notifications
function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    
    // Add styles
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 1rem;
        border-radius: 4px;
        color: white;
        font-weight: 500;
        z-index: 10000;
        opacity: 0;
        transform: translateX(100%);
        transition: all 0.3s ease;
    `;
    
    // Type-specific styling
    const colors = {
        info: '#007bff',
        success: '#28a745',
        warning: '#ffc107',
        error: '#dc3545'
    };
    
    toast.style.backgroundColor = colors[type] || colors.info;
    
    document.body.appendChild(toast);
    
    // Animate in
    requestAnimationFrame(() => {
        toast.style.opacity = '1';
        toast.style.transform = 'translateX(0)';
    });
    
    // Auto remove
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 300);
    }, 3000);
}

// Confirmation dialog
function confirmAction(message, callback) {
    if (confirm(message)) {
        callback();
    }
}

// Initialize app
document.addEventListener('DOMContentLoaded', function() {
    // Close modals when clicking outside
    document.addEventListener('click', function(event) {
        if (event.target.classList.contains('modal')) {
            event.target.style.display = 'none';
        }
    });

    // Close modals with escape key
    document.addEventListener('keydown', function(event) {
        if (event.key === 'Escape') {
            const modals = document.querySelectorAll('.modal');
            modals.forEach(modal => {
                if (modal.style.display === 'flex') {
                    modal.style.display = 'none';
                }
            });
        }
    });

    // Auto-hide alerts after 5 seconds
    const alerts = document.querySelectorAll('.alert');
    alerts.forEach(alert => {
        setTimeout(() => {
            alert.style.opacity = '0';
            setTimeout(() => {
                if (alert.parentNode) {
                    alert.parentNode.removeChild(alert);
                }
            }, 300);
        }, 5000);
    });
});

// Weight formatting helper
function formatWeight(grams) {
    if (grams >= 1000) {
        return (grams / 1000).toFixed(1) + ' kg';
    }
    return grams + ' g';
}

// Date formatting helper
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString();
}

// Export for use in other scripts
window.Carryless = {
    fetchCSRFToken,
    showModal,
    hideModal,
    apiRequest,
    submitForm,
    showToast,
    confirmAction,
    formatWeight,
    formatDate
};