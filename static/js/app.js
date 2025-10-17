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
    // Initialize dropdown navigation
    initializeDropdown();

    // Initialize weight unit selector
    initializeWeightUnitSelector();

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

// Dropdown navigation functionality
function initializeDropdown() {
    const dropdown = document.querySelector('.nav-links .dropdown');
    const dropdownToggle = document.querySelector('.nav-links .dropdown-toggle');
    const dropdownMenu = document.querySelector('.nav-links .dropdown-menu');

    if (!dropdown || !dropdownToggle || !dropdownMenu) {
        return; // No dropdown on this page
    }

    // Toggle dropdown on button click
    dropdownToggle.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        dropdown.classList.toggle('active');
        dropdownToggle.setAttribute('aria-expanded', dropdown.classList.contains('active'));
    });

    // Close dropdown when clicking outside
    document.addEventListener('click', function(e) {
        if (!dropdown.contains(e.target)) {
            dropdown.classList.remove('active');
            dropdownToggle.setAttribute('aria-expanded', 'false');
        }
    });

    // Close dropdown on escape key
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && dropdown.classList.contains('active')) {
            dropdown.classList.remove('active');
            dropdownToggle.setAttribute('aria-expanded', 'false');
            dropdownToggle.focus();
        }
    });

    // Handle arrow keys for dropdown navigation
    dropdownToggle.addEventListener('keydown', function(e) {
        if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            dropdown.classList.add('active');
            dropdownToggle.setAttribute('aria-expanded', 'true');
            const firstItem = dropdownMenu.querySelector('.dropdown-item');
            if (firstItem) {
                firstItem.focus();
            }
        }
    });

    // Handle navigation within dropdown
    const dropdownItems = dropdownMenu.querySelectorAll('.dropdown-item');
    dropdownItems.forEach((item, index) => {
        item.addEventListener('keydown', function(e) {
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                const nextIndex = (index + 1) % dropdownItems.length;
                dropdownItems[nextIndex].focus();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                const prevIndex = (index - 1 + dropdownItems.length) % dropdownItems.length;
                dropdownItems[prevIndex].focus();
            } else if (e.key === 'Escape') {
                e.preventDefault();
                dropdown.classList.remove('active');
                dropdownToggle.setAttribute('aria-expanded', 'false');
                dropdownToggle.focus();
            }
        });
    });
}

// Initialize weight unit selector
function initializeWeightUnitSelector() {
    const selector = document.getElementById('weightUnitSelector');
    if (!selector) {
        return; // No selector on this page
    }

    // Get saved unit preference
    let savedUnit = getCookie('weightUnit') || 'g';

    // Validate and set
    if (!isValidWeightUnit(savedUnit)) {
        savedUnit = 'g';
        setCookie('weightUnit', 'g');
    }

    selector.value = savedUnit;

    // Convert displays on page load if not grams
    if (savedUnit !== 'g') {
        convertWeightDisplays(savedUnit);
    }
}

/* ============================================
   Mobile Bottom Sheet Modals
   ============================================ */

// Check if viewport is mobile
function isMobileViewport() {
    return window.innerWidth <= 768;
}

// Show bottom sheet action modal
function showActionSheet(title, actions) {
    if (!isMobileViewport()) {
        return false; // Don't show on desktop
    }

    // Remove existing action sheet if any
    const existing = document.querySelector('.mobile-action-sheet');
    if (existing) {
        existing.remove();
    }

    // Create action sheet
    const sheet = document.createElement('div');
    sheet.className = 'mobile-action-sheet';

    // Build actions HTML
    const actionsHTML = actions.map(action => {
        const dangerClass = action.danger ? ' danger' : '';
        return `<button class="action-sheet-item${dangerClass}" onclick="${action.onclick}; closeActionSheet();">
            ${action.label}
        </button>`;
    }).join('');

    sheet.innerHTML = `
        <div class="action-sheet-backdrop" onclick="closeActionSheet()"></div>
        <div class="action-sheet-content">
            <div class="action-sheet-header">
                <h3>${title}</h3>
                <button onclick="closeActionSheet()" class="btn-close">×</button>
            </div>
            <div class="action-sheet-body">
                ${actionsHTML}
            </div>
        </div>
    `;

    document.body.appendChild(sheet);

    // Trigger animation
    requestAnimationFrame(() => {
        sheet.classList.add('active');
    });

    return true;
}

// Close action sheet
function closeActionSheet() {
    const sheet = document.querySelector('.mobile-action-sheet');
    if (sheet) {
        sheet.classList.remove('active');
        setTimeout(() => sheet.remove(), 300);
    }
}

// Admin user actions (for admin.html)
function showAdminUserActions(userId, username, isAdmin, isActivated) {
    if (!isMobileViewport()) {
        return false; // Desktop uses inline dropdowns
    }

    const actions = [
        {
            label: isAdmin ? 'Remove Admin' : 'Make Admin',
            onclick: `toggleAdmin(${userId})`,
            danger: false
        },
        {
            label: isActivated ? 'Deactivate User' : 'Activate User',
            onclick: `toggleActivation(${userId})`,
            danger: false
        }
    ];

    // Add resend activation email option if not activated
    if (!isActivated) {
        actions.push({
            label: 'Resend Activation Email',
            onclick: `resendActivation(${userId}, '${username}')`,
            danger: false
        });
    }

    // Add ban user option
    actions.push({
        label: 'Ban User',
        onclick: `banUser(${userId}, '${username}')`,
        danger: true
    });

    showActionSheet(`${username} Actions`, actions);
    return true;
}

// Pack actions (for packs.html)
function showPackActions(packId, packName) {
    if (!isMobileViewport()) {
        return false;
    }

    const actions = [
        {
            label: 'View Pack',
            onclick: `window.location.href='/packs/${packId}'`,
            danger: false
        },
        {
            label: 'Edit Pack',
            onclick: `window.location.href='/packs/${packId}/edit'`,
            danger: false
        },
        {
            label: 'Duplicate Pack',
            onclick: `submitPackAction(${packId}, 'duplicate')`,
            danger: false
        },
        {
            label: 'Delete Pack',
            onclick: `confirmDeletePack(${packId}, '${packName}')`,
            danger: true
        }
    ];

    showActionSheet(`${packName}`, actions);
    return true;
}

// Helper function to submit pack actions
function submitPackAction(packId, action) {
    const form = document.createElement('form');
    form.method = 'POST';
    form.action = `/packs/${packId}/${action}`;

    // Add CSRF token
    if (csrfToken) {
        const csrfInput = document.createElement('input');
        csrfInput.type = 'hidden';
        csrfInput.name = 'csrf_token';
        csrfInput.value = csrfToken;
        form.appendChild(csrfInput);
    }

    document.body.appendChild(form);
    form.submit();
}

// Confirm pack deletion
function confirmDeletePack(packId, packName) {
    if (confirm(`Are you sure you want to delete "${packName}"? This action cannot be undone.`)) {
        submitPackAction(packId, 'delete');
    }
}

// Close action sheet when pressing Escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeActionSheet();
    }
});

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
    changeWeightUnit,
    initializeDropdown,
    initializeWeightUnitSelector,
    // Mobile action sheets
    isMobileViewport,
    showActionSheet,
    closeActionSheet,
    showAdminUserActions,
    showPackActions,
    submitPackAction,
    confirmDeletePack
};

// Make action sheet functions globally available
window.showActionSheet = showActionSheet;
window.closeActionSheet = closeActionSheet;
window.showAdminUserActions = showAdminUserActions;
window.showPackActions = showPackActions;
window.submitPackAction = submitPackAction;
window.confirmDeletePack = confirmDeletePack;