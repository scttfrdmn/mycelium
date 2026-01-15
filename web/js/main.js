// Mycelium Landing Page - Interactive Features

// Tab Switching for Install Instructions
function showTab(tabName) {
    const contents = document.querySelectorAll('.tab-content');
    contents.forEach(content => {
        content.classList.remove('active');
    });

    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(button => {
        button.classList.remove('active');
    });

    const selectedTab = document.getElementById(tabName);
    if (selectedTab) {
        selectedTab.classList.add('active');
    }

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
    let defaultTab = 'homebrew';

    if (userAgent.includes('win')) {
        defaultTab = 'scoop';
    }

    showTab(defaultTab);

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

    document.querySelectorAll('.feature-card, .example, .preview-card').forEach(el => {
        observer.observe(el);
    });
});

// Copy to Clipboard for Code Blocks
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

if (navigator.clipboard) {
    document.addEventListener('DOMContentLoaded', addCopyButtons);
}

// ═══════════════════════════════════════════════════════════════
// Dashboard - Client-Side EC2 Queries
// ═══════════════════════════════════════════════════════════════

// AWS regions to query
const AWS_REGIONS = [
    'us-east-1', 'us-east-2', 'us-west-1', 'us-west-2',
    'eu-west-1', 'eu-west-2', 'eu-central-1',
    'ap-southeast-1', 'ap-southeast-2', 'ap-northeast-1'
];

// Dashboard API - Client-Side EC2 queries using user's AWS credentials
const DashboardAPI = {
    // Cross-account role ARN (development account where instances live)
    crossAccountRoleArn: 'arn:aws:iam::435415984226:role/SpawnDashboardCrossAccountReadRole',
    crossAccountCredentials: null,

    // Assume cross-account role to access EC2 instances in development account
    async assumeCrossAccountRole() {
        if (this.crossAccountCredentials && this.crossAccountCredentials.expiration > Date.now()) {
            return this.crossAccountCredentials;
        }

        const sts = new AWS.STS();
        const data = await sts.assumeRole({
            RoleArn: this.crossAccountRoleArn,
            RoleSessionName: 'spawn-dashboard-session',
            DurationSeconds: 3600
        }).promise();

        this.crossAccountCredentials = {
            accessKeyId: data.Credentials.AccessKeyId,
            secretAccessKey: data.Credentials.SecretAccessKey,
            sessionToken: data.Credentials.SessionToken,
            expiration: data.Credentials.Expiration.getTime()
        };

        return this.crossAccountCredentials;
    },

    // List instances across all regions (parallel queries)
    async listInstances() {
        if (!AWS.config.credentials) {
            throw new Error('AWS credentials not configured');
        }

        // Assume cross-account role first
        await this.assumeCrossAccountRole();

        const results = await Promise.allSettled(
            AWS_REGIONS.map(region => this.listInstancesInRegion(region))
        );

        // Combine all successful results
        const allInstances = [];
        results.forEach(result => {
            if (result.status === 'fulfilled' && result.value) {
                allInstances.push(...result.value);
            }
        });

        // Sort by launch time (newest first)
        allInstances.sort((a, b) => new Date(b.launch_time) - new Date(a.launch_time));

        return {
            success: true,
            regions_queried: AWS_REGIONS,
            total_instances: allInstances.length,
            instances: allInstances
        };
    },

    // List instances in a specific region
    async listInstancesInRegion(region) {
        // Use cross-account credentials for EC2 API calls
        const ec2 = new AWS.EC2({
            region: region,
            credentials: new AWS.Credentials({
                accessKeyId: this.crossAccountCredentials.accessKeyId,
                secretAccessKey: this.crossAccountCredentials.secretAccessKey,
                sessionToken: this.crossAccountCredentials.sessionToken
            })
        });

        // Query EC2 with filters (filter by spawn:managed tag only)
        const params = {
            Filters: [
                { Name: 'tag:spawn:managed', Values: ['true'] }
            ]
        };

        const data = await ec2.describeInstances(params).promise();

        // Convert to instance list
        const instances = [];
        data.Reservations.forEach(reservation => {
            reservation.Instances.forEach(instance => {
                instances.push(this.convertInstance(instance, region));
            });
        });

        return instances;
    },

    // Convert EC2 instance to dashboard format
    convertInstance(instance, region) {
        const tags = {};
        (instance.Tags || []).forEach(tag => {
            tags[tag.Key] = tag.Value;
        });

        return {
            instance_id: instance.InstanceId,
            name: tags['Name'] || instance.InstanceId,
            instance_type: instance.InstanceType,
            state: instance.State.Name,
            region: region,
            availability_zone: instance.Placement.AvailabilityZone,
            public_ip: instance.PublicIpAddress || null,
            private_ip: instance.PrivateIpAddress || null,
            launch_time: instance.LaunchTime,
            ttl: tags['spawn:ttl'] || null,
            dns_name: tags['spawn:dns-name'] || null,
            spot_instance: instance.InstanceLifecycle === 'spot',
            key_name: instance.KeyName || null,
            tags: tags
        };
    },

    // Get user account info
    async getUserProfile() {
        if (!AWS.config.credentials) {
            throw new Error('AWS credentials not configured');
        }

        const sts = new AWS.STS();
        const identity = await sts.getCallerIdentity().promise();

        // Also get cross-account identity
        let devAccountIdentity = null;
        try {
            await this.assumeCrossAccountRole();
            const devSts = new AWS.STS({
                credentials: new AWS.Credentials({
                    accessKeyId: this.crossAccountCredentials.accessKeyId,
                    secretAccessKey: this.crossAccountCredentials.secretAccessKey,
                    sessionToken: this.crossAccountCredentials.sessionToken
                })
            });
            devAccountIdentity = await devSts.getCallerIdentity().promise();
        } catch (error) {
            console.warn('Could not get dev account identity:', error);
        }

        return {
            success: true,
            user: {
                user_id: identity.Arn,
                aws_account_id: identity.Account,
                user_arn: identity.Arn,
                dev_account_id: devAccountIdentity?.Account || null
            }
        };
    }
};

// Dashboard UI Functions
async function loadDashboard() {
    const dashboardSection = document.getElementById('dashboard');
    if (!dashboardSection) return;

    try {
        const tbody = document.getElementById('instances-tbody');
        const errorDiv = document.getElementById('dashboard-error');
        const loadingDiv = document.getElementById('dashboard-loading');

        if (loadingDiv) loadingDiv.style.display = 'block';
        if (errorDiv) errorDiv.style.display = 'none';
        if (tbody) tbody.innerHTML = '';

        // Check AWS SDK
        if (typeof AWS === 'undefined') {
            throw new Error('AWS SDK not loaded. Please refresh the page.');
        }

        // Load instances
        const response = await DashboardAPI.listInstances();

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (response.success && response.instances && response.instances.length > 0) {
            displayInstances(response.instances);
        } else {
            if (tbody) {
                tbody.innerHTML = `
                    <tr>
                        <td colspan="7" style="text-align: center; padding: 2rem; color: var(--text-muted);">
                            No instances found. Launch your first instance with <code>spawn</code>!
                        </td>
                    </tr>
                `;
            }
        }
    } catch (error) {
        console.error('Failed to load dashboard:', error);

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (errorDiv) {
            errorDiv.style.display = 'block';
            const errorMessage = error.message || 'Unknown error';

            if (errorMessage.includes('credentials') || errorMessage.includes('not authorized')) {
                errorDiv.innerHTML = `
                    <strong>⚠️ Authentication Required</strong><br>
                    Please configure your AWS credentials to view your instances.<br>
                    <small>Make sure your IAM user has EC2 read permissions.</small>
                `;
            } else {
                errorDiv.innerHTML = `<strong>Error:</strong> ${errorMessage}`;
            }
        }
    }
}

function displayInstances(instances) {
    const tbody = document.getElementById('instances-tbody');
    if (!tbody) return;

    tbody.innerHTML = instances.map(instance => {
        const launchTime = new Date(instance.launch_time);
        const age = formatAge(launchTime);
        const stateClass = getStateClass(instance.state);

        return `
            <tr>
                <td><strong>${escapeHtml(instance.name)}</strong></td>
                <td><code>${escapeHtml(instance.instance_type)}</code></td>
                <td><span class="badge badge-${stateClass}">${escapeHtml(instance.state)}</span></td>
                <td><code>${escapeHtml(instance.region)}</code></td>
                <td>${instance.public_ip ? `<code>${escapeHtml(instance.public_ip)}</code>` : '<span style="color: var(--text-muted);">—</span>'}</td>
                <td>${instance.dns_name ? `<code>${escapeHtml(instance.dns_name)}</code>` : '<span style="color: var(--text-muted);">—</span>'}</td>
                <td>${age}</td>
            </tr>
        `;
    }).join('');
}

function formatAge(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m`;
    return `${seconds}s`;
}

function getStateClass(state) {
    const stateMap = {
        'running': 'success',
        'stopped': 'warning',
        'terminated': 'danger',
        'pending': 'info',
        'stopping': 'warning',
        'shutting-down': 'danger'
    };
    return stateMap[state] || 'default';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { DashboardAPI, showTab };
}
