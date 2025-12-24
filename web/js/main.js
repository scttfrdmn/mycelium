// Mycelium Landing Page - Interactive Features

// Tab Switching for Install Instructions
function showTab(tabName) {
    // Hide all tab contents
    const contents = document.querySelectorAll('.tab-content');
    contents.forEach(content => {
        content.classList.remove('active');
    });

    // Remove active class from all buttons
    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(button => {
        button.classList.remove('active');
    });

    // Show selected tab
    const selectedTab = document.getElementById(tabName);
    if (selectedTab) {
        selectedTab.classList.add('active');
    }

    // Add active class to clicked button
    const clickedButton = event?.target;
    if (clickedButton) {
        clickedButton.classList.add('active');
    }
}

// Smooth Scrolling for Anchor Links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// Add Loading Animation to External Links
document.querySelectorAll('a[target="_blank"]').forEach(link => {
    link.addEventListener('click', function() {
        this.style.opacity = '0.7';
        setTimeout(() => {
            this.style.opacity = '1';
        }, 300);
    });
});

// Detect OS and Set Default Tab
function setDefaultInstallTab() {
    const userAgent = navigator.userAgent.toLowerCase();
    let defaultTab = 'homebrew'; // Default to macOS/Linux

    if (userAgent.includes('win')) {
        defaultTab = 'scoop';
    }

    // Show the appropriate tab on page load
    showTab(defaultTab);

    // Highlight the correct button
    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(button => {
        button.classList.remove('active');
    });

    const activeButton = Array.from(buttons).find(btn =>
        (defaultTab === 'scoop' && btn.textContent.includes('Windows')) ||
        (defaultTab === 'homebrew' && btn.textContent.includes('macOS'))
    );

    if (activeButton) {
        activeButton.classList.add('active');
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    setDefaultInstallTab();

    // Add entrance animation to feature cards
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '0';
                entry.target.style.transform = 'translateY(20px)';
                entry.target.style.transition = 'all 0.6s ease';

                setTimeout(() => {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                }, 100);

                observer.unobserve(entry.target);
            }
        });
    }, observerOptions);

    // Observe all feature cards and sections
    document.querySelectorAll('.feature-card, .example, .preview-card').forEach(el => {
        observer.observe(el);
    });
});

// Copy to Clipboard for Code Blocks (Future Enhancement)
function addCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre code');
    codeBlocks.forEach((block, index) => {
        const button = document.createElement('button');
        button.textContent = 'Copy';
        button.className = 'copy-btn';
        button.style.cssText = `
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            padding: 0.3rem 0.8rem;
            background: var(--accent-blue);
            color: var(--bg-dark);
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.85rem;
            opacity: 0;
            transition: opacity 0.3s ease;
        `;

        const pre = block.parentElement;
        pre.style.position = 'relative';
        pre.appendChild(button);

        pre.addEventListener('mouseenter', () => {
            button.style.opacity = '1';
        });

        pre.addEventListener('mouseleave', () => {
            button.style.opacity = '0';
        });

        button.addEventListener('click', () => {
            navigator.clipboard.writeText(block.textContent).then(() => {
                button.textContent = 'Copied!';
                setTimeout(() => {
                    button.textContent = 'Copy';
                }, 2000);
            });
        });
    });
}

// Initialize copy buttons
if (navigator.clipboard) {
    document.addEventListener('DOMContentLoaded', addCopyButtons);
}

// Future: Web Dashboard Integration
// This will be expanded when building the instance management UI
const DashboardAPI = {
    // Placeholder for future API calls
    async listInstances() {
        // TODO: Fetch from API Gateway or Lambda
        return [];
    },

    async getInstance(instanceId) {
        // TODO: Fetch instance details
        return null;
    },

    async startInstance(instanceId) {
        // TODO: Start instance via API
    },

    async stopInstance(instanceId) {
        // TODO: Stop instance via API
    },

    async terminateInstance(instanceId) {
        // TODO: Terminate instance via API
    },

    async getMetrics(instanceId) {
        // TODO: Fetch CloudWatch metrics
        return null;
    }
};

// Export for future use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { DashboardAPI, showTab };
}
