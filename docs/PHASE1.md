# Phase 1: Core Webhook and Database Implementation

## Overview

This phase implements the core webhook functionality with SCEP challenge validation, database storage, and a management dashboard.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Step CA Webhook                          │
├─────────────────────────────────────────────────────────────┤
│  HTTP Handlers                                               │
│  ├── /scep/challenge/{serial} - Challenge validation       │
│  ├── /scep/register - Device registration                  │
│  ├── /scep/challenge/get/{serial} - Get challenge          │
│  ├── /scep/devices - List devices                           │
│  ├── /scep/device/{serial} - Delete device                  │
│  ├── /scep/device/rotate/{serial} - Rotate passphrase       │
│  ├── /dashboard/ - Management dashboard                     │
├─────────────────────────────────────────────────────────────┤
│  Database Layer                                              │
│  ├── SQLite database (scep.db)                              │
│  ├── Device CRUD operations                                  │
│  ├── Passphrase storage (encrypted)                         │
│  └── Challenge validation logic                             │
├─────────────────────────────────────────────────────────────┤
│  SCEP Challenge Module                                        │
│  ├── Challenge encryption/decryption                        │
│  ├── Challenge validation                                   │
│  └── Diceware passphrase generation                          │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Database Layer (`pkg/db/`)

**File: `database.go`**

Features:
- SQLite-based persistent storage
- Device CRUD operations
- Secure passphrase storage
- Challenge validation
- Database migrations

**API:**
- `RegisterDevice(serial, hostname, passphrase)` - Register a new device
- `GetDevice(serial)` - Retrieve device by serial number
- `GetDeviceByHostname(hostname)` - Retrieve device by hostname
- `ListDevices()` - List all devices
- `DeleteDevice(serial)` - Remove a device
- `RotatePassphrase(serial)` - Generate new passphrase

### 2. SCEP Challenge Module (`pkg/scep/`)

**File: `scep.go`**

Features:
- Challenge encryption using device passphrase
- Challenge validation
- Diceware passphrase generation
- Secure challenge storage

**API:**
- `EncryptSCEPChallenge(passphrase, serial)` - Generate encrypted challenge
- `ValidateSCEPChallenge(passphrase, serial, challenge)` - Validate challenge
- `GenerateDicewarePassphrase()` - Generate random passphrase

### 3. Server Layer (`pkg/server/`)

**File: `scep.go`**

Features:
- HTTP handlers for SCEP operations
- Challenge validation webhook
- Device registration
- Dashboard API endpoints
- Security authentication

**API Endpoints:**
- `GET /scep/challenge/{serial}` - Validate SCEP challenge
- `POST /scep/register` - Register new device
- `GET /scep/challenge/get/{serial}` - Get encrypted challenge
- `GET /scep/devices` - List all devices
- `DELETE /scep/device/{serial}` - Delete device
- `POST /scep/device/rotate/{serial}` - Rotate passphrase

### 4. Dashboard (`dashboard/`)

**Files:**
- `main.go` - Dashboard API server
- `templates/index.html` - User interface
- `static/style.css` - Styling
- `static/script.js` - JavaScript logic

Features:
- Device management UI
- Device registration form
- Device list with actions
- Passphrase rotation
- Toast notifications
- Responsive design

### 5. Main Application (`main.go`)

**Features:**
- TLS configuration
- Webhook authentication
- Database initialization
- Route registration
- Server startup

## Usage

### Starting the Webhook Server

```bash
# Run the main webhook server
go run main.go
```

The server will start on `https://localhost:4443`

### Dashboard

1. Open the dashboard at `http://localhost:8080` (from a different port)
2. Register a new device with:
   - Serial number (16 characters)
   - Hostname
   - Diceware passphrase
3. The dashboard will generate an encrypted challenge for the device
4. The device can request a certificate using the challenge

### Device Registration API

```bash
# Register a device
curl -X POST https://localhost:4443/scep/register \
  -H "Content-Type: application/json" \
  -d '{
    "serial_number": "1234567890123456",
    "hostname": "device.local",
    "passphrase": "Queen-Ate-Lamp-Wise-Smell"
  }'
```

### List Devices

```bash
curl -X GET https://localhost:4443/scep/devices
```

### Delete Device

```bash
curl -X DELETE https://localhost:4443/scep/device/1234567890123456
```

### Rotate Passphrase

```bash
curl -X POST https://localhost:4443/scep/device/rotate/1234567890123456
```

## Security Considerations

1. **TLS**: All communication is encrypted using TLS with client certificate validation
2. **Passphrase Storage**: Passphrases are stored securely in the database
3. **Challenge Encryption**: Challenges are encrypted using device passphrases
4. **Authentication**: Webhook requests are authenticated using secret keys
5. **XSS Protection**: Dashboard uses Content Security Policy
6. **CSRF Protection**: (To be implemented in Phase 2)

## Testing

Run the database tests:

```bash
go test ./pkg/db/...
go test ./pkg/scep/...
go test ./pkg/server/...
```

## Deployment

### Environment Variables

```bash
# TLS Certificates
export CERT_FILE="webhook.crt"
export KEY_FILE="webhook.key"

# Client CA certificates
export CLIENT_CAS="root_ca.crt"

# Server Address
export ADDRESS=":4443"

# Database File
export DB_FILE="scep.db"
```

### Docker (Optional)

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o webhook

EXPOSE 4443

CMD ["./webhook"]
```

## Files Created

### Core Components
- `pkg/scep/scep.go` - SCEP challenge validation and encryption
- `pkg/db/database.go` - Database layer with device management
- `pkg/db/database_test.go` - Database tests
- `pkg/server/scep.go` - HTTP handlers for SCEP operations
- `main.go` - Main application entry point

### Dashboard
- `dashboard/main.go` - Dashboard API server
- `dashboard/templates/index.html` - Dashboard UI
- `dashboard/static/style.css` - Dashboard styling
- `dashboard/static/script.js` - Dashboard JavaScript

### Scripts
- `scripts/install-scepclient-macos.sh` - macOS installation script
- `scripts/install-scepclient-windows.ps1` - Windows installation script

### Documentation
- `docs/PHASE1.md` - This document

## Next Steps (Phase 2)

- [ ] Implement WebSocket for real-time updates
- [ ] Add user authentication
- [ ] Implement event logging and auditing
- [ ] Add rate limiting
- [ ] Implement CSRF protection
- [ ] Add metrics and monitoring
- [ ] Implement advanced challenge strategies
- [ ] Add certificate lifecycle management
- [ ] Implement certificate revocation checking
- [ ] Add mobile device support (iOS)
- [ ] Add Windows mobile device support

## References

- SCEP RFC: [RFC 8899](https://datatracker.ietf.org/doc/html/rfc8899)
- Step CA Documentation: [https://smallstep.com/docs/certificates/step-ca](https://smallstep.com/docs/certificates/step-ca)
- Micromdm SCEP Client: [https://github.com/micromdm/scepclient](https://github.com/micromdm/scepclient)
- Diceware Passphrases: [https://world.std.com/~cme/html/rick.html](https://world.std.com/~cme/html/rick.html)