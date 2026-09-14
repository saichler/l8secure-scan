/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Main application initialization for Layer 8 Secure Scan

// Get authentication headers with bearer token
function getAuthHeaders() {
    const bearerToken = sessionStorage.getItem('bearerToken');
    return {
        'Authorization': bearerToken ? `Bearer ${bearerToken}` : '',
        'Content-Type': 'application/json'
    };
}

// Utility function for making authenticated API calls
async function makeAuthenticatedRequest(url, options = {}) {
    const bearerToken = sessionStorage.getItem('bearerToken');

    if (!bearerToken) {
        console.error('No bearer token found');
        window.location.href = 'l8ui/login/index.html';
        return;
    }

    const headers = {
        'Authorization': `Bearer ${bearerToken}`,
        'Content-Type': 'application/json',
        ...options.headers
    };

    try {
        const response = await fetch(url, {
            ...options,
            headers: headers
        });

        if (response.status === 401) {
            sessionStorage.removeItem('bearerToken');
            window.location.href = 'l8ui/login/index.html';
            return;
        }

        return response;
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// Logout function
function logout() {
    sessionStorage.removeItem('bearerToken');
    localStorage.removeItem('bearerToken');
    localStorage.removeItem('rememberedUser');
    window.location.href = 'l8ui/login/index.html';
}

// Initialize the application
document.addEventListener('DOMContentLoaded', async function() {
    // Load app configuration first
    if (typeof Layer8DConfig !== 'undefined') {
        await Layer8DConfig.load();
    }

    // Check if bearer token exists (user is logged in)
    const bearerToken = sessionStorage.getItem('bearerToken');
    if (!bearerToken) {
        window.location.href = 'l8ui/login/index.html';
        return;
    }

    // Sync bearer token to localStorage so iframes can access it
    localStorage.setItem('bearerToken', bearerToken);
    window.bearerToken = bearerToken;

    // Set username in header from current session
    const username = sessionStorage.getItem('currentUser') || 'User';
    document.querySelector('.username').textContent = username;

    // Module filter skipped -- secscan has no server-side ModConfig service
    // (ModconfigFailureNoLogout). If Layer8DModuleFilter is present, mark it
    // as loaded so downstream code doesn't block.
    if (typeof Layer8DModuleFilter !== 'undefined') {
        Layer8DModuleFilter._loaded = true;
    }

    // §11.6 asks for the admin nav entry to be hidden from non-opsadmin
    // users. NOT implemented here: verified (see plans/PROGRESS.md) that
    // neither the /auth response nor the bearer JWT (l8secure's Claim type
    // is just jwt.StandardClaims -- no roles) carries the caller's role to
    // the client, and no verified endpoint exposes "my own roles" either.
    // The admin nav link stays visible to everyone; this is not a security
    // gap -- the customer role has zero permissions on Customer server-side
    // (§14's deny-before-allow, the real boundary per §11.6's own text), so
    // a customer-role user who clicks in just sees an empty/erroring table.

    // Add event listeners to navigation links
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            navLinks.forEach(l => l.classList.remove('active'));
            this.classList.add('active');
            const section = this.getAttribute('data-section');
            loadSection(section);
        });
    });

    // Load default section (Vulnerability Management -- the Image Groups
    // dashboard, PRD §11.1) only once a customer is confirmed -- an
    // unscoped user (opsadmin, PRD §4) must not have any section fetched/
    // rendered, and no query fired (loadCategoryCache, KPI counts, table
    // loads all happen inside sectionInitializers.vulnmgmt), until they've
    // picked one via SecScanCustomerPicker. Verified as a real bug: this
    // call used to run unconditionally here, so the whole dashboard loaded
    // in the background behind the picker popup regardless of selection.
    if (typeof SecScanCustomerPicker !== 'undefined') {
        SecScanCustomerPicker.checkAndPrompt(function() {
            loadSection('vulnmgmt');
        });
    } else {
        loadSection('vulnmgmt');
    }
});
