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
        info: '#004aad',
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

    // Auto-hide alerts after 5 seconds (except persistent ones)
    const alerts = document.querySelectorAll('.alert:not(.alert-persistent)');
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

// Weight unit conversion functions
function gramsToOunces(grams) {
    return grams * 0.035274;
}

function ouncesToGrams(ounces) {
    return Math.round(ounces / 0.035274);
}

// Enhanced weight formatting with auto-conversion to lbs when oz is too large
function formatWeightWithUnit(grams, unit) {
    if (unit === 'oz') {
        const oz = gramsToOunces(grams);
        
        // Auto-convert to lbs when oz >= 16 (1 pound = 16 ounces)
        if (oz >= 16) {
            const lbs = oz / 16;
            if (lbs >= 10) {
                // Show whole lbs for large weights
                return Math.round(lbs) + ' lbs';
            } else {
                // Show lbs with 1 decimal for smaller weights
                return lbs.toFixed(1) + ' lbs';
            }
        }
        
        // Show oz with appropriate precision
        if (oz < 1) {
            return oz.toFixed(3) + ' oz';
        } else if (oz < 10) {
            return oz.toFixed(2) + ' oz';
        } else {
            return oz.toFixed(1) + ' oz';
        }
    }
    return grams + ' g';
}

// Cookie management
function setCookie(name, value, days = 365) {
    const expires = new Date();
    expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
    document.cookie = name + '=' + value + ';expires=' + expires.toUTCString() + ';path=/';
}

function getCookie(name) {
    const nameEQ = name + "=";
    const ca = document.cookie.split(';');
    for (let i = 0; i < ca.length; i++) {
        let c = ca[i];
        while (c.charAt(0) === ' ') c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) === 0) return c.substring(nameEQ.length, c.length);
    }
    return null;
}

// Weight unit validation and conversion
function isValidWeightUnit(unit) {
    return unit === 'g' || unit === 'oz';
}

function convertWeightDisplays(unit) {
    // Convert all weight displays on the page
    const weightElements = document.querySelectorAll('[data-weight]');
    weightElements.forEach(element => {
        const grams = parseInt(element.dataset.weight);
        element.textContent = formatWeightWithUnit(grams, unit);
    });
    
    // Convert statistics (using data-weight attributes for accurate conversion)
    const statElements = document.querySelectorAll('.stat-value[data-weight]');
    statElements.forEach(element => {
        const grams = parseInt(element.dataset.weight);
        if (!isNaN(grams)) {
            element.textContent = formatWeightWithUnit(grams, unit);
        }
    });
    
    // Convert category headers
    const categoryHeaders = document.querySelectorAll('.category-section h3');
    categoryHeaders.forEach(header => {
        const text = header.textContent;
        const gramsMatch = text.match(/(\d+)g/g);
        if (gramsMatch) {
            let newText = text;
            gramsMatch.forEach(match => {
                const grams = parseInt(match.replace('g', ''));
                const converted = formatWeightWithUnit(grams, unit);
                newText = newText.replace(match, converted);
            });
            header.textContent = newText;
        }
    });
}

function changeWeightUnit(unit) {
    // Validate unit before setting
    if (!isValidWeightUnit(unit)) {
        console.warn('Invalid weight unit:', unit, 'Defaulting to grams');
        unit = 'g';
    }
    
    // Save preference to cookie
    setCookie('weight_unit', unit);
    
    // Convert all weight displays on the page
    convertWeightDisplays(unit);
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
    formatDate,
    gramsToOunces,
    ouncesToGrams,
    formatWeightWithUnit,
    setCookie,
    getCookie,
    isValidWeightUnit,
    convertWeightDisplays,
    changeWeightUnit
};