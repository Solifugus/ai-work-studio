# Troubleshooting Guide

*Solutions to common issues and debugging techniques*

This guide helps you identify, diagnose, and resolve common problems with AI Work Studio. Most issues fall into a few categories and have straightforward solutions.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Installation Issues](#installation-issues)
- [Startup Problems](#startup-problems)
- [LLM Provider Issues](#llm-provider-issues)
- [Performance Problems](#performance-problems)
- [Data and Storage Issues](#data-and-storage-issues)
- [Interface Problems](#interface-problems)
- [Learning and Method Issues](#learning-and-method-issues)
- [Network and Connectivity](#network-and-connectivity)
- [Advanced Debugging](#advanced-debugging)

## Quick Diagnostics

### Health Check

Run these commands first to get an overview of system status:

```bash
# Check overall system health
ai-work-studio --health-check

# Verify configuration
ai-work-studio --config-check

# Test LLM providers
ai-work-studio --test-providers

# Check data integrity
ai-work-studio --verify-data
```

### Common Quick Fixes

**Try these first for any issue:**

1. **Restart the application**:
   ```bash
   # Kill any running processes
   pkill -f ai-work-studio

   # Restart
   ai-work-studio
   ```

2. **Check logs for errors**:
   ```bash
   # View recent logs
   tail -f ~/.local/share/ai-work-studio/logs/app.log

   # Check for errors
   grep -i error ~/.local/share/ai-work-studio/logs/app.log
   ```

3. **Verify configuration**:
   ```bash
   ai-work-studio --config-check
   ```

4. **Check disk space and permissions**:
   ```bash
   # Check available space
   df -h ~/.config/ai-work-studio

   # Check permissions
   ls -la ~/.config/ai-work-studio
   ```

## Installation Issues

### "Go not found" or "Command not found"

**Problem**: Go compiler not installed or not in PATH

**Solutions**:
```bash
# Check if Go is installed
go version

# If not found, install Go 1.21+
# Linux:
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# macOS:
brew install go

# Windows: Download installer from https://golang.org/dl/
```

### "ai-work-studio: command not found"

**Problem**: Binary not in PATH or installation incomplete

**Solutions**:
```bash
# Check if binary exists
which ai-work-studio

# If not found, check installation location
ls -la ~/.local/bin/ai-work-studio
ls -la /usr/local/bin/ai-work-studio

# Add to PATH if needed
echo 'export PATH=$HOME/.local/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# Or reinstall
curl -sSL https://raw.githubusercontent.com/yourusername/ai-work-studio/main/install.sh | bash
```

### Build Failures

**Problem**: Compilation errors during installation

**Common causes and solutions**:

**Missing dependencies**:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install build-essential git

# CentOS/RHEL
sudo yum groupinstall "Development Tools"
sudo yum install git

# macOS
xcode-select --install
```

**Outdated Go version**:
```bash
# Check version
go version

# Must be 1.21+, update if older
```

**Network issues**:
```bash
# Try with proxy settings
export GOPROXY=https://proxy.golang.org
go mod download
```

### Permission Denied Errors

**Problem**: Insufficient permissions for installation or data directories

**Solutions**:
```bash
# Fix binary permissions
chmod +x ~/.local/bin/ai-work-studio*

# Fix data directory permissions
chmod -R 755 ~/.config/ai-work-studio
chmod -R 755 ~/.local/share/ai-work-studio

# For system-wide install issues, use sudo:
sudo chown $USER:$USER /usr/local/bin/ai-work-studio*
```

## Startup Problems

### Application Won't Start

**Symptoms**: No window appears, immediate exit, error messages

**Diagnostic steps**:
```bash
# Run from terminal to see error messages
ai-work-studio --debug

# Check if another instance is running
ps aux | grep ai-work-studio

# Kill existing processes
pkill -f ai-work-studio

# Check for port conflicts
netstat -tulpn | grep :8080  # or your configured port
```

**Common fixes**:

**Configuration file issues**:
```bash
# Validate configuration
ai-work-studio --config-check

# Reset to defaults if corrupted
mv ~/.config/ai-work-studio/config.json ~/.config/ai-work-studio/config.json.backup
ai-work-studio --setup
```

**Missing data directories**:
```bash
# Recreate directory structure
mkdir -p ~/.config/ai-work-studio/{nodes,edges,methods,backups,cache}
mkdir -p ~/.local/share/ai-work-studio/logs
```

### GUI Doesn't Display

**Problem**: Application starts but no window appears

**Linux solutions**:
```bash
# Check display environment
echo $DISPLAY

# Try running with display specified
DISPLAY=:0 ai-work-studio

# For Wayland systems
ai-work-studio --backend wayland

# For headless systems
ai-work-studio --no-gui
```

**macOS solutions**:
```bash
# Check security permissions
# Go to System Preferences > Security & Privacy > Privacy
# Ensure AI Work Studio has accessibility permissions

# Try running as regular user (not root)
```

**Windows solutions**:
```powershell
# Check Windows Defender exclusions
# Add installation directory to exclusions

# Run as administrator if permission issues
# Right-click executable -> "Run as administrator"
```

### Slow Startup

**Problem**: Application takes too long to start

**Diagnostic**:
```bash
# Run with timing information
time ai-work-studio --profile

# Check data directory size
du -sh ~/.config/ai-work-studio

# Look for large log files
ls -lah ~/.local/share/ai-work-studio/logs/
```

**Solutions**:
```bash
# Clean up old data
ai-work-studio --cleanup

# Reduce cache size in config
# Edit ~/.config/ai-work-studio/config.json
# Set smaller cache_size

# Archive old logs
gzip ~/.local/share/ai-work-studio/logs/*.log
```

## LLM Provider Issues

### API Key Problems

**"Invalid API key" or "Authentication failed"**

**Solutions**:
```bash
# Verify key format (no extra spaces or characters)
echo $ANTHROPIC_API_KEY | cat -v

# Test key directly
curl -H "Authorization: Bearer $ANTHROPIC_API_KEY" \
     https://api.anthropic.com/v1/messages

# Regenerate key if needed
# Visit provider console and create new key

# Store in file instead of environment
echo "your-key" > ~/.config/ai-work-studio/keys/anthropic.key
chmod 600 ~/.config/ai-work-studio/keys/anthropic.key
```

### Local Model Issues

**Model download failures**:
```bash
# Check disk space
df -h ~/.local/share/ai-work-studio/models

# Download manually
ai-work-studio --download-model llama-2-7b-chat

# Use different model mirror
ai-work-studio --model-source huggingface
```

**Model loading errors**:
```bash
# Check model file integrity
ai-work-studio --verify-model llama-2-7b-chat

# Reduce context length or batch size
# Edit config.json:
{
  "local": {
    "context_length": 2048,
    "batch_size": 4
  }
}
```

**GPU acceleration not working**:
```bash
# Check GPU availability
nvidia-smi  # for NVIDIA GPUs

# Verify CUDA installation
nvcc --version

# Disable GPU if problematic
ai-work-studio --cpu-only
```

### Rate Limiting

**"Rate limit exceeded" or "Too many requests"**

**Solutions**:
```bash
# Check current usage
ai-work-studio --budget-status

# Reduce concurrent requests in config:
{
  "concurrency": {
    "max_concurrent_objectives": 1
  },
  "network": {
    "rate_limiting": {
      "requests_per_minute": 30
    }
  }
}

# Use local model for simple tasks
{
  "llm": {
    "routing": {
      "simple_tasks": "local"
    }
  }
}
```

### Budget Exceeded

**"Budget limit reached" messages**

**Solutions**:
```bash
# Check current usage
ai-work-studio --budget-status

# Increase limits in config
# Or reset for new billing period
ai-work-studio --reset-budget monthly

# Use local models to reduce costs
ai-work-studio --prefer-local
```

## Performance Problems

### Slow Response Times

**Symptoms**: Long delays for objective completion, UI freezing

**Diagnosis**:
```bash
# Check system resources
top -p $(pgrep ai-work-studio)
htop

# Monitor network usage
iftop

# Check logs for performance warnings
grep -i "slow\|timeout\|performance" ~/.local/share/ai-work-studio/logs/app.log
```

**Solutions**:

**Reduce context size**:
```json
{
  "llm": {
    "max_context_length": 2048,
    "context_compression": true
  }
}
```

**Optimize concurrent processing**:
```json
{
  "performance": {
    "max_concurrent_objectives": 1,
    "thread_pool_size": 2
  }
}
```

**Use local models for simple tasks**:
```json
{
  "llm": {
    "routing": {
      "simple_tasks": "local",
      "complex_analysis": "anthropic"
    }
  }
}
```

### High Memory Usage

**Problem**: Application consuming too much RAM

**Solutions**:
```json
{
  "performance": {
    "memory": {
      "max_usage": "1GB",
      "cache_size": "256MB"
    },
    "local": {
      "memory_limit": "4GB",
      "context_length": 2048
    }
  }
}
```

```bash
# Clear cache
ai-work-studio --clear-cache

# Restart application regularly
# Set up cron job for daily restart
```

### Disk Space Issues

**Problem**: Running out of storage space

**Solutions**:
```bash
# Check space usage
du -sh ~/.config/ai-work-studio/*

# Clean up old data
ai-work-studio --cleanup --older-than 30d

# Compress backups
gzip ~/.config/ai-work-studio/backups/*.json

# Move data to larger disk
ai-work-studio --export-data /new/location/backup.tar.gz
# Update config with new data_directory
ai-work-studio --import-data /new/location/backup.tar.gz
```

## Data and Storage Issues

### Corrupted Data

**Symptoms**: Application crashes on startup, missing goals/objectives, error loading data

**Recovery steps**:
```bash
# Check data integrity
ai-work-studio --verify-data

# Restore from backup
ai-work-studio --list-backups
ai-work-studio --restore-backup 2024-01-15

# Manual recovery
cp ~/.config/ai-work-studio/backups/latest.json ~/.config/ai-work-studio/recovery.json
ai-work-studio --import-data recovery.json
```

### Backup Failures

**Problem**: Automated backups not working

**Diagnosis**:
```bash
# Check backup status
ai-work-studio --backup-status

# Test manual backup
ai-work-studio --backup --verify

# Check backup directory permissions
ls -la ~/.config/ai-work-studio/backups
```

**Solutions**:
```bash
# Fix permissions
chmod 755 ~/.config/ai-work-studio/backups

# Increase backup disk space
# Or change backup location in config

# Disable compression if causing issues
{
  "backup": {
    "compression": false
  }
}
```

### Data Migration Issues

**Problem**: Moving data between systems or upgrading

**Solutions**:
```bash
# Export all data
ai-work-studio --export-complete /backup/complete-export.tar.gz

# On new system after installation:
ai-work-studio --import-complete /backup/complete-export.tar.gz

# Verify import
ai-work-studio --verify-data
```

## Interface Problems

### Window Sizing Issues

**Problem**: Window too small/large, panels misaligned

**Solutions**:
```bash
# Reset window settings
rm ~/.config/ai-work-studio/ui-settings.json

# Force specific window size
ai-work-studio --window-size 1200x800

# Use different scaling
ai-work-studio --scale 1.5
```

### Theme or Display Problems

**Problem**: Colors wrong, text unreadable, visual glitches

**Solutions**:
```json
{
  "ui": {
    "theme": "light",  // force light theme
    "high_contrast": true,
    "font_size": "large"
  }
}
```

```bash
# Reset UI settings
rm ~/.config/ai-work-studio/ui-settings.json

# Use system theme
ai-work-studio --system-theme

# Disable animations if causing issues
ai-work-studio --no-animations
```

### Keyboard Shortcuts Not Working

**Problem**: Hotkeys don't respond

**Solutions**:
```bash
# Check for conflicting applications
# Disable other apps using same shortcuts

# Reset shortcuts to defaults
ai-work-studio --reset-shortcuts

# Use alternative shortcuts
ai-work-studio --dvorak-keys  # for Dvorak keyboards
```

## Learning and Method Issues

### Methods Not Improving

**Problem**: System doesn't seem to learn from feedback

**Diagnosis**:
```bash
# Check method evolution
ai-work-studio --method-history

# View learning statistics
ai-work-studio --learning-stats

# Check feedback recording
grep "feedback" ~/.local/share/ai-work-studio/logs/app.log
```

**Solutions**:
- **Provide more specific feedback**: Instead of "didn't work", explain what specifically failed
- **Be consistent**: Give feedback on every objective completion
- **Allow time for learning**: Methods improve over weeks, not days
- **Check learning rate setting**: May be too conservative

```json
{
  "agent": {
    "learning_rate": "aggressive"  // vs "balanced" or "conservative"
  }
}
```

### Objectives Keep Failing

**Problem**: High failure rate on objectives

**Common causes and solutions**:

**Objectives too large**:
- Break into smaller, more specific tasks
- Provide more context and constraints

**Unclear success criteria**:
- Define specific, measurable outcomes
- Give examples of what success looks like

**System lacks domain knowledge**:
- Provide more background information
- Start with simpler objectives to build understanding

**Resource constraints**:
- Check if budget/time limits are too restrictive
- Verify system has access to needed tools

### Poor Quality Results

**Problem**: System produces low-quality outputs

**Solutions**:
- **Raise quality threshold**: Provide feedback that current quality is insufficient
- **Give better examples**: Show what high-quality looks like in your domain
- **Use better models**: Switch to more capable LLM provider for complex tasks
- **Provide more context**: Help system understand your quality standards

## Network and Connectivity

### Connection Timeouts

**Problem**: Frequent network timeouts to LLM providers

**Solutions**:
```json
{
  "network": {
    "timeout": 60000,  // increase timeout
    "retry_attempts": 5,
    "retry_delay": 2000
  }
}
```

```bash
# Test connectivity
curl -I https://api.anthropic.com
ping api.openai.com

# Check proxy settings
echo $HTTP_PROXY $HTTPS_PROXY

# Configure proxy in app
{
  "network": {
    "proxy": {
      "http_proxy": "http://proxy.company.com:8080",
      "https_proxy": "https://proxy.company.com:8080"
    }
  }
}
```

### Firewall Issues

**Problem**: Corporate firewall blocking API requests

**Solutions**:
```bash
# Test specific endpoints
telnet api.anthropic.com 443
telnet api.openai.com 443

# Use local models if external access blocked
{
  "llm": {
    "default_provider": "local",
    "routing": {
      "all_tasks": "local"
    }
  }
}

# Configure firewall exceptions
# Add *.anthropic.com, *.openai.com to whitelist
```

### SSL/TLS Errors

**Problem**: Certificate verification failures

**Solutions**:
```bash
# Update system certificates
sudo apt update && sudo apt install ca-certificates  # Linux
brew install ca-certificates  # macOS

# Configure custom certificates if needed
{
  "network": {
    "tls": {
      "verify_certificates": true,
      "certificate_path": "/path/to/custom/certs"
    }
  }
}

# Temporary workaround (not recommended for production)
ai-work-studio --insecure-tls
```

## Advanced Debugging

### Collecting Debug Information

**For bug reports**:
```bash
# Collect system info
ai-work-studio --system-info > debug-info.txt

# Collect logs
tar -czf logs.tar.gz ~/.local/share/ai-work-studio/logs/

# Export configuration (remove sensitive data first!)
cp ~/.config/ai-work-studio/config.json config-debug.json
# Edit config-debug.json to remove API keys

# Test specific functionality
ai-work-studio --test-all > test-results.txt
```

### Verbose Logging

**Enable detailed logging**:
```json
{
  "log_level": "debug",
  "debug": {
    "log_llm_requests": true,
    "log_method_execution": true,
    "log_performance": true
  }
}
```

```bash
# Run with maximum verbosity
ai-work-studio --debug --verbose --trace

# Monitor real-time
tail -f ~/.local/share/ai-work-studio/logs/debug.log
```

### Performance Profiling

**For performance issues**:
```bash
# CPU profiling
ai-work-studio --profile-cpu

# Memory profiling
ai-work-studio --profile-memory

# Network profiling
ai-work-studio --profile-network

# View results
ai-work-studio --show-profile
```

### Database Debugging

**For data issues**:
```bash
# Check data consistency
ai-work-studio --verify-data --verbose

# Rebuild indices
ai-work-studio --rebuild-index

# Validate all nodes and edges
ai-work-studio --validate-all

# Show data statistics
ai-work-studio --data-stats
```

## Getting Help

### When to Seek Support

- **Installation fails** after following troubleshooting steps
- **Data corruption** not resolved by backup restore
- **Performance issues** persist after optimization
- **Crashes or errors** not covered in this guide

### Information to Provide

When reporting issues, include:

1. **System information**: OS, version, hardware specs
2. **AI Work Studio version**: `ai-work-studio --version`
3. **Configuration**: (with sensitive data removed)
4. **Error logs**: Recent relevant log entries
5. **Reproduction steps**: How to reproduce the issue
6. **Expected vs actual behavior**: What should happen vs what does happen

### Support Channels

- **GitHub Issues**: [Repository issues page](https://github.com/yourusername/ai-work-studio/issues)
- **Community Discussions**: [GitHub Discussions](https://github.com/yourusername/ai-work-studio/discussions)
- **Documentation**: [Project documentation](https://github.com/yourusername/ai-work-studio/docs)

### Emergency Recovery

**If system is completely broken**:
```bash
# Nuclear option - fresh start (saves backups first)
ai-work-studio --emergency-backup
rm -rf ~/.config/ai-work-studio ~/.local/share/ai-work-studio
# Reinstall and restore from backup
```

Remember: AI Work Studio is designed to be resilient. Most issues have simple solutions, and the system can usually recover gracefully from problems.

---

**Congratulations!** You've completed the AI Work Studio user guide. You now have all the knowledge needed to effectively use, configure, and troubleshoot your personal AI assistant.