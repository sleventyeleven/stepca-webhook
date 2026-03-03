// DOM Elements
const registerForm = document.getElementById('register-form');
const rotateForm = document.getElementById('rotate-form');
const devicesContainer = document.getElementById('devices-container');
const toast = document.getElementById('toast');

// API Base URL
const API_BASE = '/api';

// Toast notification function
function showToast(message, type = 'success') {
    toast.textContent = message;
    toast.className = `toast show ${type}`;
    setTimeout(() => {
        toast.className = 'toast';
    }, 3000);
}

// Register Device
registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const serialNumber = document.getElementById('serial_number').value;
    const hostname = document.getElementById('hostname').value;
    const passphrase = document.getElementById('passphrase').value;

    const btn = registerForm.querySelector('button');
    btn.disabled = true;
    btn.textContent = 'Registering...';

    try {
        const response = await fetch(`${API_BASE}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ serial_number: serialNumber, hostname, passphrase })
        });

        const data = await response.json();

        if (response.ok) {
            showToast(data.message);
            registerForm.reset();
            loadDevices();
        } else {
            showToast(data.message, 'error');
        }
    } catch (error) {
        showToast('Failed to register device', 'error');
        console.error(error);
    } finally {
        btn.disabled = false;
        btn.textContent = 'Register Device';
    }
});

// Rotate Passphrase
rotateForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const serial = document.getElementById('rotate-serial').value;

    const btn = rotateForm.querySelector('button');
    btn.disabled = true;
    btn.textContent = 'Rotating...';

    try {
        const response = await fetch(`${API_BASE}/device/rotate/${serial}`, {
            method: 'POST'
        });

        const data = await response.json();

        if (response.ok) {
            showToast('Passphrase rotated successfully');
            showToast(`New passphrase: ${data.new_passphrase}`, 'success');
        } else {
            showToast(data.message, 'error');
        }
    } catch (error) {
        showToast('Failed to rotate passphrase', 'error');
        console.error(error);
    } finally {
        btn.disabled = false;
        btn.textContent = 'Rotate Passphrase';
    }
});

// Load Devices
async function loadDevices() {
    try {
        const response = await fetch(`${API_BASE}/devices`);
        const devices = await response.json();

        if (devices.length === 0) {
            devicesContainer.innerHTML = `
                <div class="empty-state">
                    <p>No devices registered yet</p>
                </div>
            `;
            return;
        }

        const table = `
            <table class="devices-table">
                <thead>
                    <tr>
                        <th>Serial Number</th>
                        <th>Hostname</th>
                        <th>Created At</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${devices.map(device => `
                        <tr>
                            <td>${device.serial_number}</td>
                            <td>${device.hostname}</td>
                            <td>${device.created_at}</td>
                            <td>
                                <button class="delete-btn" onclick="deleteDevice('${device.serial_number}')">
                                    Delete
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;

        devicesContainer.innerHTML = table;
    } catch (error) {
        showToast('Failed to load devices', 'error');
        console.error(error);
    }
}

// Delete Device
async function deleteDevice(serialNumber) {
    if (!confirm(`Are you sure you want to delete device ${serialNumber}?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/device/${serialNumber}`, {
            method: 'DELETE'
        });

        const data = await response.json();

        if (response.ok) {
            showToast(data.message);
            loadDevices();
        } else {
            showToast(data.message, 'error');
        }
    } catch (error) {
        showToast('Failed to delete device', 'error');
        console.error(error);
    }
}

// Initialize
loadDevices();

// Add deleteDevice to global scope
window.deleteDevice = deleteDevice;