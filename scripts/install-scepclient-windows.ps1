# SCEP Client Installation Script for Windows
# This script installs micromdm/scepclient and configures it for Step CA

[CmdletBinding()]
param(
    [Parameter(Mandatory=$false)]
    [string]$SCEPServerUrl = "https://your-webhook-server:4443/scep/challenge/{serial}",
    [Parameter(Mandatory=$false)]
    [string]$StepCaUrl = "https://your-step-ca-server:4443"
)

# Colors for output
$Green = "`e[32m"
$Red = "`e[31m"
$Yellow = "`e[33m"
$Reset = "`e[0m"

Write-Host "${Green}=== SCEP Client Installation for Windows ===${Reset}`n"

# Check if PowerShell is running as administrator
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "${Yellow}Warning: Not running as Administrator${Reset}"
    Write-Host "Some operations may require elevated privileges."
    Write-Host "Press any key to continue anyway...`n"
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

# Check if winget is available
$wingetAvailable = Get-Command winget -ErrorAction SilentlyContinue

if (-not $wingetAvailable) {
    # Check if chocolatey is available
    $chocoAvailable = Get-Command choco -ErrorAction SilentlyContinue

    if (-not $chocoAvailable) {
        Write-Host "${Red}Error: Neither winget nor chocolatey is installed${Reset}"
        Write-Host "Please install one of them:"
        Write-Host "  - winget: https://aka.ms/winget"
        Write-Host "  - chocolatey: https://chocolatey.org/install"
        exit 1
    } else {
        Write-Host "${Green}✓ Chocolatey is installed${Reset}`n"
    }
} else {
    Write-Host "${Green}✓ winget is installed${Reset}`n"
}

# Install SCEP client
Write-Host "Installing micromdm/scepclient..."

if ($wingetAvailable) {
    winget install micromdm.scepclient --accept-package-agreements --accept-source-agreements
} else {
    choco install scepclient -y
}

Write-Host "${Green}✓ SCEP client installed${Reset}`n"

# Set up Step CA root certificate
Write-Host "${Yellow}Setting up Step CA root certificate...${Reset}"
$env:APPDATA = $env:APPDATA + "\step-ca"
$rootCertPath = Join-Path $env:APPDATA "root_ca.crt"

# Create Step CA directory if it doesn't exist
if (-not (Test-Path $env:APPDATA)) {
    New-Item -ItemType Directory -Path $env:APPDATA -Force | Out-Null
}

# Check if root CA certificate already exists
if (Test-Path $rootCertPath) {
    Write-Host "${Green}✓ Root CA certificate already exists${Reset}"
} else {
    Write-Host "${Yellow}Please download your Step CA root certificate to:${Reset}"
    Write-Host "${Yellow}${rootCertPath}${Reset}"
    Write-Host "${Yellow}You can generate it from your Step CA server:${Reset}"
    Write-Host "step ca root $rootCertPath"
}

# Create SCEP profile configuration
Write-Host "${Yellow}Creating SCEP profile configuration...${Reset}"

# Get device hostname
$deviceHostname = $env:COMPUTERNAME

# Generate a random serial number (16 characters)
$deviceSerial = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 16 | % {[char]$_})

# Create SCEP profile configuration
$scepConfigPath = Join-Path $env:APPDATA "scepclient.yaml"
$scepConfigContent = @"
# SCEP Client Configuration
server: ${SCEPServerUrl}

# CA URL (for certificate issuance)
ca_url: ${StepCaUrl}

# Device information
device:
  serial: ${deviceSerial}
  hostname: ${deviceHostname}

# Certificate profile
profile:
  name: "step-ca"
  challenge:
    # Challenge will be automatically handled
    enabled: true

# Certificate request settings
request:
  key_type: RSA2048
  key_size: 2048
  validity: 365

# Logging
logging:
  level: info
  file: ${env:TEMP}\scepclient.log
"@

$scepConfigContent | Out-File -FilePath $scepConfigPath -Encoding UTF8

Write-Host "${Green}✓ SCEP client configuration created${Reset}"
Write-Host "${Green}✓ Device serial: ${deviceSerial}${Reset}`n"

# Test connection
Write-Host "${Yellow}Testing SCEP connection...${Reset}"
try {
    scepclient ping
    Write-Host "${Green}✓ SCEP client is working correctly${Reset}`n"
} catch {
    Write-Host "${Red}✗ SCEP client ping failed${Reset}"
    Write-Host "Please check your configuration and try again"
    exit 1
}

# Instructions
Write-Host "${Green}=== Installation Complete ===${Reset}`n"
Write-Host "SCEP client has been installed successfully."
Write-Host ""
Write-Host "Configuration file location: ${scepConfigPath}"
Write-Host ""
Write-Host "To request a certificate, run:"
Write-Host "  scepclient request --profile step-ca"
Write-Host ""
Write-Host "To update the device serial in the dashboard:"
Write-Host "  1. Navigate to the dashboard at http://localhost:8080"
Write-Host "  2. Register the device with serial: ${deviceSerial}"
Write-Host "  3. Provide the passphrase used in the configuration"
Write-Host ""
Write-Host "To view logs:"
Write-Host "  Get-Content ${env:TEMP}\scepclient.log -Tail 20"
Write-Host ""
Write-Host "Note: You may need to restart PowerShell for PATH changes to take effect."