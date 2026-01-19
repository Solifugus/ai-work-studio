# AI Work Studio Installation Script for Windows
# PowerShell script for automated installation
# Simplicity through experience, not complexity through programming

param(
    [switch]$NoInteractive,
    [string]$InstallDir,
    [string]$DataDir,
    [switch]$Help
)

# Configuration
$ProjectName = "AI Work Studio"
$RepoUrl = "https://github.com/yourusername/ai-work-studio"
$DefaultInstallDir = "$env:LOCALAPPDATA\Programs\AIWorkStudio"
$DefaultDataDir = "$env:APPDATA\AIWorkStudio"
$DefaultLogDir = "$env:LOCALAPPDATA\AIWorkStudio\logs"
$ServiceName = "AIWorkStudioAgent"

# Colors for output
$Colors = @{
    Red     = "Red"
    Green   = "Green"
    Yellow  = "Yellow"
    Blue    = "Blue"
    Cyan    = "Cyan"
    Default = "White"
}

function Write-Header {
    Write-Host "================================" -ForegroundColor $Colors.Blue
    Write-Host "  AI Work Studio Installer" -ForegroundColor $Colors.Blue
    Write-Host "================================" -ForegroundColor $Colors.Blue
    Write-Host ""
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor $Colors.Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ Error: $Message" -ForegroundColor $Colors.Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ Warning: $Message" -ForegroundColor $Colors.Yellow
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor $Colors.Blue
}

function Show-Help {
    Write-Host "AI Work Studio Installer for Windows" -ForegroundColor $Colors.Blue
    Write-Host ""
    Write-Host "Usage: .\install.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -InstallDir <path>   Custom installation directory"
    Write-Host "  -DataDir <path>      Custom data directory"
    Write-Host "  -NoInteractive       Run without user prompts"
    Write-Host "  -Help                Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\install.ps1                               # Interactive installation"
    Write-Host "  .\install.ps1 -NoInteractive                # Silent installation"
    Write-Host "  .\install.ps1 -InstallDir 'C:\Tools\AWS'    # Custom install location"
}

function Test-Requirements {
    Write-Info "Checking system requirements..."

    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Error "PowerShell 5.0+ required. Current version: $($PSVersionTable.PSVersion)"
        exit 1
    }
    Write-Success "PowerShell $($PSVersionTable.PSVersion) found"

    # Check for Go
    try {
        $goVersion = & go version 2>$null
        if ($LASTEXITCODE -eq 0) {
            $version = ($goVersion -split ' ')[2] -replace 'go', ''
            Write-Success "Go $version found"

            # Check version requirement
            $requiredVersion = [Version]"1.21.0"
            $currentVersion = [Version]$version
            if ($currentVersion -lt $requiredVersion) {
                Write-Error "Go version $version found, but 1.21.0+ required"
                exit 1
            }
        } else {
            throw "Go not found"
        }
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        Write-Host "Please install Go 1.21+ from https://golang.org/dl/"
        exit 1
    }

    # Check for Git
    try {
        $null = & git --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Git found"
        } else {
            throw "Git not found"
        }
    }
    catch {
        Write-Error "Git is not installed or not in PATH"
        Write-Host "Please install Git from https://git-scm.com/download/win"
        exit 1
    }

    # Check if running as administrator (for system-wide install)
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    $script:IsAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

    if ($script:IsAdmin) {
        Write-Info "Running as Administrator"
    } else {
        Write-Info "Running as regular user (user-local installation)"
    }
}

function Get-InstallDirectory {
    if (-not [string]::IsNullOrEmpty($InstallDir)) {
        $script:InstallDirectory = $InstallDir
        Write-Info "Using specified install directory: $script:InstallDirectory"
        return
    }

    if ($NoInteractive) {
        $script:InstallDirectory = $DefaultInstallDir
        Write-Info "Using default install directory: $script:InstallDirectory"
        return
    }

    Write-Host ""
    Write-Host "Choose installation directory:"
    Write-Host "1) $DefaultInstallDir (default, user local)"
    if ($script:IsAdmin) {
        Write-Host "2) C:\Program Files\AIWorkStudio (system-wide)"
    }
    Write-Host "3) Custom directory"
    Write-Host ""

    do {
        $choice = Read-Host "Selection [1]"
        if ([string]::IsNullOrEmpty($choice)) { $choice = "1" }

        switch ($choice) {
            "1" {
                $script:InstallDirectory = $DefaultInstallDir
                $script:NeedsAdmin = $false
                break
            }
            "2" {
                if ($script:IsAdmin) {
                    $script:InstallDirectory = "C:\Program Files\AIWorkStudio"
                    $script:NeedsAdmin = $true
                } else {
                    Write-Warning "Administrator rights required for system-wide installation"
                    continue
                }
                break
            }
            "3" {
                $customDir = Read-Host "Enter custom directory"
                if (-not [string]::IsNullOrEmpty($customDir)) {
                    $script:InstallDirectory = $customDir
                }
                break
            }
            default {
                Write-Warning "Invalid selection. Please choose 1, 2, or 3."
                continue
            }
        }
        break
    } while ($true)

    Write-Info "Will install to: $script:InstallDirectory"
}

function Get-DataDirectory {
    if (-not [string]::IsNullOrEmpty($DataDir)) {
        $script:DataDirectory = $DataDir
        Write-Info "Using specified data directory: $script:DataDirectory"
        return
    }

    if ($NoInteractive) {
        $script:DataDirectory = $DefaultDataDir
        Write-Info "Using default data directory: $script:DataDirectory"
        return
    }

    Write-Host ""
    $userInput = Read-Host "Data directory [$DefaultDataDir]"
    if ([string]::IsNullOrEmpty($userInput)) {
        $script:DataDirectory = $DefaultDataDir
    } else {
        $script:DataDirectory = $userInput
    }

    Write-Info "Will use data directory: $script:DataDirectory"
}

function Setup-Directories {
    Write-Info "Setting up directories..."

    # Create install directory
    if (!(Test-Path $script:InstallDirectory)) {
        New-Item -ItemType Directory -Path $script:InstallDirectory -Force | Out-Null
    }

    # Create data directories
    $dataDirs = @("nodes", "edges", "backups", "cache", "methods", "keys")
    foreach ($dir in $dataDirs) {
        $fullPath = Join-Path $script:DataDirectory $dir
        if (!(Test-Path $fullPath)) {
            New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
        }
    }

    # Create log directory
    if (!(Test-Path $DefaultLogDir)) {
        New-Item -ItemType Directory -Path $DefaultLogDir -Force | Out-Null
    }

    Write-Success "Directories created"
}

function Download-Source {
    Write-Info "Downloading source code..."

    $tempDir = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
    New-Item -ItemType Directory -Path $tempDir | Out-Null

    try {
        if (Test-Path "go.mod" -PathType Leaf) {
            Write-Info "Using current directory (development mode)"
            $script:BuildDirectory = Get-Location
        } else {
            Set-Location $tempDir
            & git clone $RepoUrl ai-work-studio
            if ($LASTEXITCODE -ne 0) {
                throw "Git clone failed"
            }
            Set-Location "ai-work-studio"
            $script:BuildDirectory = Get-Location
        }
        Write-Success "Source code ready"
    }
    catch {
        Write-Error "Failed to download source: $_"
        exit 1
    }
}

function Build-Project {
    Write-Info "Building AI Work Studio..."

    Push-Location $script:BuildDirectory

    try {
        # Download dependencies
        & go mod download
        if ($LASTEXITCODE -ne 0) { throw "go mod download failed" }

        & go mod tidy
        if ($LASTEXITCODE -ne 0) { throw "go mod tidy failed" }

        # Create bin directory
        if (!(Test-Path "bin")) {
            New-Item -ItemType Directory -Path "bin" | Out-Null
        }

        # Build binaries
        Write-Info "Building studio application..."
        & go build -o "bin\ai-work-studio.exe" .\cmd\studio
        if ($LASTEXITCODE -ne 0) { throw "Studio build failed" }

        Write-Info "Building agent daemon..."
        & go build -o "bin\ai-work-studio-agent.exe" .\cmd\agent
        if ($LASTEXITCODE -ne 0) { throw "Agent build failed" }

        $script:StudioBinary = Join-Path (Get-Location) "bin\ai-work-studio.exe"
        $script:AgentBinary = Join-Path (Get-Location) "bin\ai-work-studio-agent.exe"

        Write-Success "Build completed"
    }
    catch {
        Write-Error "Build failed: $_"
        exit 1
    }
    finally {
        Pop-Location
    }
}

function Install-Binaries {
    Write-Info "Installing binaries..."

    try {
        Copy-Item $script:StudioBinary (Join-Path $script:InstallDirectory "ai-work-studio.exe")
        Copy-Item $script:AgentBinary (Join-Path $script:InstallDirectory "ai-work-studio-agent.exe")
        Write-Success "Binaries installed"
    }
    catch {
        Write-Error "Failed to install binaries: $_"
        exit 1
    }
}

function Configure-System {
    Write-Info "Setting up configuration..."

    $configPath = Join-Path $script:DataDirectory "config.json"

    $config = @{
        version = "1.0"
        data_directory = $script:DataDirectory.Replace('\', '/')
        log_directory = $DefaultLogDir.Replace('\', '/')
        log_level = "info"
        storage = @{
            type = "file"
            backup_enabled = $true
            backup_interval = "24h"
        }
        llm = @{
            default_provider = "local"
            budget = @{
                daily_limit = 100.0
                warn_threshold = 80.0
            }
        }
        agent = @{
            auto_start = $false
            check_interval = "5m"
        }
    }

    $configJson = $config | ConvertTo-Json -Depth 10
    Set-Content -Path $configPath -Value $configJson -Encoding UTF8

    Write-Success "Basic configuration created"
}

function Setup-LlmConfig {
    if ($NoInteractive) {
        Write-Info "Skipping LLM configuration (non-interactive mode)"
        return
    }

    Write-Host ""
    Write-Info "LLM Configuration Setup"
    Write-Host "AI Work Studio supports both local and remote LLM providers."
    Write-Host ""

    Write-Host "Available options:"
    Write-Host "1) Local models only (privacy-focused, no API costs)"
    Write-Host "2) Anthropic Claude API (requires API key)"
    Write-Host "3) OpenAI API (requires API key)"
    Write-Host "4) Mixed (local + remote, configure later)"
    Write-Host ""

    do {
        $llmChoice = Read-Host "Select LLM setup [1]"
        if ([string]::IsNullOrEmpty($llmChoice)) { $llmChoice = "1" }

        switch ($llmChoice) {
            "1" {
                Write-Info "Local-only setup selected"
                break
            }
            "2" {
                Write-Host ""
                Write-Info "Anthropic Claude API setup"
                Write-Host "Get your API key from: https://console.anthropic.com/"
                $anthropicKey = Read-Host "Enter Anthropic API key" -AsSecureString
                if ($anthropicKey.Length -gt 0) {
                    $keyPath = Join-Path $script:DataDirectory "keys\anthropic.key"
                    $bstr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($anthropicKey)
                    $plainKey = [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
                    Set-Content -Path $keyPath -Value $plainKey -Encoding UTF8
                    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)

                    # Set restricted permissions
                    $acl = Get-Acl $keyPath
                    $acl.SetAccessRuleProtection($true, $false)
                    $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                        $env:USERNAME, "FullControl", "Allow"
                    )
                    $acl.SetAccessRule($accessRule)
                    Set-Acl -Path $keyPath -AclObject $acl

                    Write-Success "Anthropic API key configured"
                }
                break
            }
            "3" {
                Write-Host ""
                Write-Info "OpenAI API setup"
                Write-Host "Get your API key from: https://platform.openai.com/api-keys"
                $openaiKey = Read-Host "Enter OpenAI API key" -AsSecureString
                if ($openaiKey.Length -gt 0) {
                    $keyPath = Join-Path $script:DataDirectory "keys\openai.key"
                    $bstr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($openaiKey)
                    $plainKey = [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
                    Set-Content -Path $keyPath -Value $plainKey -Encoding UTF8
                    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)

                    # Set restricted permissions
                    $acl = Get-Acl $keyPath
                    $acl.SetAccessRuleProtection($true, $false)
                    $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                        $env:USERNAME, "FullControl", "Allow"
                    )
                    $acl.SetAccessRule($accessRule)
                    Set-Acl -Path $keyPath -AclObject $acl

                    Write-Success "OpenAI API key configured"
                }
                break
            }
            "4" {
                Write-Info "Mixed setup selected - configure providers later in the UI"
                break
            }
            default {
                Write-Warning "Invalid selection. Please choose 1, 2, 3, or 4."
                continue
            }
        }
        break
    } while ($true)
}

function Setup-WindowsService {
    if ($script:IsAdmin -and -not $NoInteractive) {
        Write-Host ""
        $installService = Read-Host "Install Windows service for background agent? [y/N]"

        if ($installService -match '^[Yy]') {
            Write-Info "Installing Windows service..."

            try {
                $serviceBinary = Join-Path $script:InstallDirectory "ai-work-studio-agent.exe"
                $configPath = Join-Path $script:DataDirectory "config.json"

                & sc.exe create $ServiceName binPath= "`"$serviceBinary`" --config `"$configPath`"" `
                    start= demand DisplayName= "AI Work Studio Agent" `
                    description= "AI Work Studio background agent service"

                if ($LASTEXITCODE -eq 0) {
                    Write-Success "Windows service installed"
                    Write-Info "Start with: sc start $ServiceName"
                    Write-Info "Or use Services.msc to manage"
                } else {
                    Write-Warning "Service installation failed"
                }
            }
            catch {
                Write-Warning "Failed to install service: $_"
            }
        }
    } elseif (-not $script:IsAdmin) {
        Write-Info "Administrator rights required for Windows service installation"
    }
}

function Add-ToPath {
    if ($NoInteractive) {
        Write-Info "Skipping PATH modification (non-interactive mode)"
        return
    }

    # Check if already in PATH
    $currentPath = $env:PATH
    if ($currentPath -notlike "*$($script:InstallDirectory)*") {
        Write-Host ""
        $addToPath = Read-Host "Add installation directory to PATH? [Y/n]"

        if ($addToPath -notmatch '^[Nn]') {
            try {
                # Add to user PATH
                $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
                if ($userPath -notlike "*$($script:InstallDirectory)*") {
                    $newPath = if ($userPath) { "$userPath;$($script:InstallDirectory)" } else { $script:InstallDirectory }
                    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
                    Write-Success "Added to user PATH"
                    Write-Info "Restart your command prompt to use the new PATH"
                }
            }
            catch {
                Write-Warning "Failed to modify PATH: $_"
                Write-Info "Manual step: Add $($script:InstallDirectory) to your PATH"
            }
        }
    }
}

function Test-Installation {
    if ($NoInteractive) {
        Write-Info "Skipping verification tests (non-interactive mode)"
        return
    }

    Write-Host ""
    $runTests = Read-Host "Run quick verification tests? [Y/n]"

    if ($runTests -notmatch '^[Nn]') {
        Write-Info "Running verification tests..."

        # Test binaries
        $studioExe = Join-Path $script:InstallDirectory "ai-work-studio.exe"
        $agentExe = Join-Path $script:InstallDirectory "ai-work-studio-agent.exe"

        if (Test-Path $studioExe) {
            Write-Success "Studio binary verified"
        } else {
            Write-Error "Studio binary not found"
        }

        if (Test-Path $agentExe) {
            Write-Success "Agent binary verified"
        } else {
            Write-Error "Agent binary not found"
        }

        # Test data directories
        $requiredDirs = @("nodes", "edges", "backups", "cache", "methods")
        $allDirsExist = $true
        foreach ($dir in $requiredDirs) {
            $fullPath = Join-Path $script:DataDirectory $dir
            if (!(Test-Path $fullPath)) {
                $allDirsExist = $false
                break
            }
        }

        if ($allDirsExist) {
            Write-Success "Data directory structure verified"
        } else {
            Write-Error "Data directory structure incomplete"
        }

        # Test configuration
        $configPath = Join-Path $script:DataDirectory "config.json"
        if (Test-Path $configPath) {
            Write-Success "Configuration file verified"
        } else {
            Write-Error "Configuration file missing"
        }
    }
}

function Write-Completion {
    Write-Host ""
    Write-Host "================================" -ForegroundColor $Colors.Green
    Write-Host "  Installation Complete!" -ForegroundColor $Colors.Green
    Write-Host "================================" -ForegroundColor $Colors.Green
    Write-Host ""
    Write-Host "Installation Summary:"
    Write-Host "  Binaries:      $script:InstallDirectory"
    Write-Host "  Data:          $script:DataDirectory"
    Write-Host "  Logs:          $DefaultLogDir"
    Write-Host "  Configuration: $(Join-Path $script:DataDirectory 'config.json')"
    Write-Host ""
    Write-Host "Next Steps:"
    Write-Host "  1. Restart your command prompt (if PATH was modified)"
    Write-Host "  2. Run: ai-work-studio --help"
    Write-Host "  3. Start the GUI: ai-work-studio"
    Write-Host "  4. Optional: Start background agent: ai-work-studio-agent"
    Write-Host ""
    Write-Host "Documentation: docs\installation.md"
    Write-Host "Support: $RepoUrl/issues"
    Write-Host ""
}

function Cleanup {
    if ($script:TempDirectory -and (Test-Path $script:TempDirectory)) {
        try {
            Remove-Item -Path $script:TempDirectory -Recurse -Force
        }
        catch {
            Write-Warning "Could not clean up temporary directory: $script:TempDirectory"
        }
    }
}

# Main installation function
function Install-AIWorkStudio {
    try {
        Write-Header

        if ($Help) {
            Show-Help
            return
        }

        # Check if already installed
        $existingInstall = Get-Command "ai-work-studio" -ErrorAction SilentlyContinue
        if ($existingInstall -and -not $NoInteractive) {
            Write-Warning "AI Work Studio appears to already be installed"
            $continue = Read-Host "Continue with reinstallation? [y/N]"
            if ($continue -notmatch '^[Yy]') {
                Write-Host "Installation cancelled"
                return
            }
        }

        Test-Requirements
        Get-InstallDirectory
        Get-DataDirectory
        Setup-Directories

        # Build or download
        if (Test-Path "go.mod") {
            Write-Info "Installing from current directory"
            $script:BuildDirectory = Get-Location
            Build-Project
        } else {
            Download-Source
            Build-Project
        }

        Install-Binaries
        Configure-System
        Setup-LlmConfig
        Setup-WindowsService
        Add-ToPath
        Test-Installation
        Write-Completion
    }
    catch {
        Write-Error "Installation failed: $_"
        exit 1
    }
    finally {
        Cleanup
    }
}

# Script entry point
if ($MyInvocation.InvocationName -eq $MyInvocation.MyCommand.Name) {
    Install-AIWorkStudio
}