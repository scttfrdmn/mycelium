// Multi-Provider OIDC Authentication for Spawn Dashboard
// Client-Side Architecture: Users query their own AWS accounts directly

// Configuration (credentials from config/google-oauth-credentials.json)
const AUTH_CONFIG = {
    region: 'us-east-1',
    identityPoolId: 'us-east-1:f51165b6-05f7-46c7-a8d3-947080c17876',

    providers: {
        globus: {
            name: 'Globus Auth',
            clientId: '8b578341-b7b5-4e0b-b29f-14a8b3d9011a',
            authEndpoint: 'https://auth.globus.org/v2/oauth2/authorize',
            scope: 'openid profile email',
            cognitoKey: 'auth.globus.org'
        },
        google: {
            name: 'Google',
            clientId: '721954523328-d2ark4gifse2j3g763opso2isgqmp4em.apps.googleusercontent.com',
            authEndpoint: 'https://accounts.google.com/o/oauth2/v2/auth',
            scope: 'openid profile email',
            cognitoKey: 'accounts.google.com'
        },
        github: {
            name: 'GitHub',
            clientId: 'Ov23liOPNcrWFpDvtWrX',
            authEndpoint: 'https://github.com/login/oauth/authorize',
            scope: 'read:user user:email',
            cognitoKey: null, // GitHub OAuth 2.0 (not OIDC) - requires backend Lambda
            disabled: true // Coming soon - requires additional backend infrastructure
        }
    }
};

// Authentication Manager
class AuthManager {
    constructor() {
        this.currentUser = null;
        this.credentials = null;
        this.loadFromStorage();
    }

    // Initialize authentication (check for OAuth callback)
    async init() {
        // Check if we're returning from OAuth callback
        const hash = window.location.hash.substring(1);
        const params = new URLSearchParams(hash);

        if (params.has('id_token')) {
            await this.handleOAuthCallback(params);
            window.history.replaceState({}, document.title, window.location.pathname);
            return true;
        }

        // Check if we have stored credentials
        if (this.credentials && this.credentials.expiration > Date.now()) {
            await this.configureAWS();
            return true;
        }

        return false;
    }

    // Start OAuth login flow
    login(provider) {
        const config = AUTH_CONFIG.providers[provider];
        if (!config) {
            throw new Error(`Unknown provider: ${provider}`);
        }

        if (config.disabled) {
            alert(`${config.name} is coming soon! It requires additional backend infrastructure (OAuth 2.0 → OIDC token exchange). For now, please use Google or Globus Auth.`);
            return;
        }

        if (!config.clientId) {
            alert(`${config.name} is not configured yet. Please set up OAuth apps first.`);
            return;
        }

        if (!AUTH_CONFIG.identityPoolId) {
            alert('Cognito Identity Pool not configured. Run setup-dashboard-cognito.sh first.');
            return;
        }

        // Build OAuth authorization URL
        const redirectUri = window.location.origin + '/callback';
        const state = this.generateState(provider);

        const params = new URLSearchParams({
            client_id: config.clientId,
            redirect_uri: redirectUri,
            response_type: 'id_token token',
            scope: config.scope,
            state: state,
            nonce: this.generateNonce()
        });

        // Store state for verification
        sessionStorage.setItem('oauth_state', state);
        sessionStorage.setItem('oauth_provider', provider);

        // Redirect to OAuth provider
        window.location.href = `${config.authEndpoint}?${params.toString()}`;
    }

    // Handle OAuth callback
    async handleOAuthCallback(params) {
        const storedState = sessionStorage.getItem('oauth_state');
        const returnedState = params.get('state');

        if (storedState !== returnedState) {
            throw new Error('OAuth state mismatch - possible CSRF attack');
        }

        const provider = sessionStorage.getItem('oauth_provider');
        const config = AUTH_CONFIG.providers[provider];

        // Get ID token
        const idToken = params.get('id_token');
        if (!idToken) {
            throw new Error('No ID token received from OAuth provider');
        }

        // Parse ID token to get user info
        const userInfo = this.parseJWT(idToken);
        this.currentUser = {
            provider: provider,
            id: userInfo.sub,
            email: userInfo.email || userInfo.preferred_username || 'User',
            name: userInfo.name || userInfo.email || 'User'
        };

        // Exchange OIDC token for AWS credentials via Cognito Identity Pool
        await this.exchangeTokenForCredentials(provider, idToken);

        // Save to storage
        this.saveToStorage();

        // Clean up
        sessionStorage.removeItem('oauth_state');
        sessionStorage.removeItem('oauth_provider');

        console.log('Logged in as:', this.currentUser.email);
    }

    // Exchange OIDC token for AWS credentials
    async exchangeTokenForCredentials(provider, idToken) {
        if (!AUTH_CONFIG.identityPoolId) {
            throw new Error('Cognito Identity Pool ID not configured');
        }

        const config = AUTH_CONFIG.providers[provider];

        // Configure AWS SDK region
        AWS.config.region = AUTH_CONFIG.region;

        const cognitoIdentity = new AWS.CognitoIdentity();

        // Get Identity ID
        const logins = {};
        logins[config.cognitoKey] = idToken;

        try {
            const identityData = await cognitoIdentity.getId({
                IdentityPoolId: AUTH_CONFIG.identityPoolId,
                Logins: logins
            }).promise();

            const identityId = identityData.IdentityId;

            // Get temporary credentials
            const credentialsData = await cognitoIdentity.getCredentialsForIdentity({
                IdentityId: identityId,
                Logins: logins
            }).promise();

            this.credentials = {
                accessKeyId: credentialsData.Credentials.AccessKeyId,
                secretAccessKey: credentialsData.Credentials.SecretKey,
                sessionToken: credentialsData.Credentials.SessionToken,
                expiration: credentialsData.Credentials.Expiration.getTime(),
                identityId: identityId
            };

            // Configure AWS SDK with new credentials
            await this.configureAWS();

        } catch (error) {
            console.error('Failed to exchange token for credentials:', error);
            throw new Error(`Authentication failed: ${error.message}`);
        }
    }

    // Configure AWS SDK with credentials
    async configureAWS() {
        if (!this.credentials) {
            throw new Error('No credentials available');
        }

        AWS.config.update({
            region: AUTH_CONFIG.region,
            credentials: new AWS.Credentials({
                accessKeyId: this.credentials.accessKeyId,
                secretAccessKey: this.credentials.secretAccessKey,
                sessionToken: this.credentials.sessionToken
            })
        });

        // Verify credentials work
        try {
            const sts = new AWS.STS();
            const identity = await sts.getCallerIdentity().promise();
            console.log('✓ AWS credentials configured');
            console.log('  User ARN:', identity.Arn);
            console.log('  Account:', identity.Account);
        } catch (error) {
            console.error('Failed to verify AWS credentials:', error);
            throw error;
        }
    }

    // Logout
    logout() {
        this.currentUser = null;
        this.credentials = null;
        localStorage.removeItem('spawn_auth');
        sessionStorage.clear();
        AWS.config.credentials = null;
        window.location.reload();
    }

    // Check if user is logged in
    isAuthenticated() {
        return this.currentUser !== null &&
               this.credentials !== null &&
               this.credentials.expiration > Date.now();
    }

    // Get current user
    getUser() {
        return this.currentUser;
    }

    // Save to localStorage
    saveToStorage() {
        const data = {
            user: this.currentUser,
            credentials: this.credentials
        };
        localStorage.setItem('spawn_auth', JSON.stringify(data));
    }

    // Load from localStorage
    loadFromStorage() {
        try {
            const data = localStorage.getItem('spawn_auth');
            if (data) {
                const parsed = JSON.parse(data);
                this.currentUser = parsed.user;
                this.credentials = parsed.credentials;

                if (this.credentials && this.credentials.expiration <= Date.now()) {
                    console.log('Stored credentials expired');
                    this.logout();
                }
            }
        } catch (error) {
            console.error('Failed to load stored auth:', error);
            localStorage.removeItem('spawn_auth');
        }
    }

    // Helper: Generate random state
    generateState(provider) {
        return provider + '_' + Math.random().toString(36).substring(2, 15);
    }

    // Helper: Generate nonce
    generateNonce() {
        return Math.random().toString(36).substring(2, 15) +
               Math.random().toString(36).substring(2, 15);
    }

    // Helper: Parse JWT token
    parseJWT(token) {
        try {
            const base64Url = token.split('.')[1];
            const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
            const jsonPayload = decodeURIComponent(atob(base64).split('').map(c => {
                return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
            }).join(''));

            return JSON.parse(jsonPayload);
        } catch (error) {
            console.error('Failed to parse JWT:', error);
            throw new Error('Invalid token format');
        }
    }
}

// Global auth manager instance
const authManager = new AuthManager();

// Initialize on page load
document.addEventListener('DOMContentLoaded', async function() {
    try {
        const authenticated = await authManager.init();

        if (authenticated) {
            // User is logged in
            showDashboard();
            loadDashboard(); // From main.js
        } else {
            // Show login page
            showLoginPage();
        }
    } catch (error) {
        console.error('Auth initialization failed:', error);
        showError('Authentication error: ' + error.message);
        showLoginPage();
    }
});

// UI Functions
function showLoginPage() {
    const dashboardSection = document.getElementById('dashboard');
    if (dashboardSection) {
        dashboardSection.style.display = 'none';
    }

    const loginSection = document.getElementById('login-section');
    if (loginSection) {
        loginSection.style.display = 'block';
    }
}

function showDashboard() {
    const loginSection = document.getElementById('login-section');
    if (loginSection) {
        loginSection.style.display = 'none';
    }

    const dashboardSection = document.getElementById('dashboard');
    if (dashboardSection) {
        dashboardSection.style.display = 'block';
    }

    // Update user info display
    const user = authManager.getUser();
    if (user) {
        const userDisplay = document.getElementById('user-display');
        if (userDisplay) {
            userDisplay.innerHTML = `
                <span>Logged in as <strong>${escapeHtml(user.email)}</strong> via ${user.provider}</span>
                <button onclick="authManager.logout()" class="btn-logout">Logout</button>
            `;
        }
    }
}

function showError(message) {
    const errorDiv = document.getElementById('auth-error');
    if (errorDiv) {
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { authManager, AuthManager };
}
