# Configuration Guide

*Tuning AI Work Studio for your specific needs and preferences*

AI Work Studio is designed to learn your preferences through experience, but some configuration helps it work optimally in your environment. This guide covers both initial setup and ongoing tuning options.

## Table of Contents

- [Configuration Philosophy](#configuration-philosophy)
- [Configuration Files](#configuration-files)
- [LLM Provider Setup](#llm-provider-setup)
- [Budget Management](#budget-management)
- [Storage Configuration](#storage-configuration)
- [Interface Preferences](#interface-preferences)
- [Performance Tuning](#performance-tuning)
- [Security and Privacy](#security-and-privacy)
- [Advanced Configuration](#advanced-configuration)

## Configuration Philosophy

AI Work Studio follows the principle **"Learn preferences, don't configure them."** Most customization happens naturally as the system learns your patterns. Configuration focuses on:

- **Environmental setup**: Connecting to services and tools you use
- **Resource constraints**: Budget, storage, processing limits
- **Security boundaries**: Privacy and access controls
- **Performance optimization**: Speed and efficiency tuning

**What you DON'T need to configure**:
- Workflow preferences (learned through use)
- Communication style (adapts to your feedback)
- Quality thresholds (develops through experience)
- Method selection (evolves based on success patterns)

## Configuration Files

### Main Configuration File

**Location**:
- **Linux/macOS**: `~/.config/ai-work-studio/config.json`
- **Windows**: `%APPDATA%\AIWorkStudio\config.json`

**Structure**:
```json
{
  "version": "1.0",
  "data_directory": "~/.config/ai-work-studio",
  "log_directory": "~/.local/share/ai-work-studio/logs",
  "log_level": "info",

  "storage": {
    "type": "file",
    "backup_enabled": true,
    "backup_interval": "24h",
    "backup_retention": "90d",
    "compression": true
  },

  "llm": {
    "default_provider": "local",
    "fallback_provider": "anthropic",
    "budget": {
      "daily_limit": 25.0,
      "monthly_limit": 500.0,
      "warn_threshold": 80.0,
      "currency": "USD"
    },
    "routing": {
      "simple_tasks": "local",
      "complex_analysis": "anthropic",
      "creative_tasks": "openai"
    }
  },

  "agent": {
    "auto_start": false,
    "check_interval": "5m",
    "max_concurrent_objectives": 2,
    "learning_rate": "balanced"
  },

  "ui": {
    "theme": "auto",
    "startup_view": "goals",
    "auto_save": true,
    "notifications": true
  },

  "privacy": {
    "anonymize_exports": true,
    "local_processing_preferred": true,
    "data_sharing": false
  }
}
```

### Environment Variables

Sensitive configuration can be set via environment variables:

```bash
# API Keys
export ANTHROPIC_API_KEY="your-key-here"
export OPENAI_API_KEY="your-key-here"
export HUGGINGFACE_API_KEY="your-key-here"

# Storage
export AI_WORK_STUDIO_DATA_DIR="/custom/data/path"
export AI_WORK_STUDIO_LOG_LEVEL="debug"

# Security
export AI_WORK_STUDIO_ENCRYPTION_KEY="your-encryption-key"
```

### Per-Goal Configuration

Individual goals can have specific settings:

```json
{
  "goal_id": "goal-123",
  "name": "Organize digital workspace",
  "config": {
    "privacy_level": "high",
    "preferred_provider": "local",
    "max_budget_per_objective": 5.0,
    "auto_execute": false,
    "notification_level": "summary"
  }
}
```

## LLM Provider Setup

### Local Models

**Benefits**: Privacy, no cost per use, works offline
**Drawbacks**: Requires local resources, may be slower for complex tasks

**Setup**:
```json
{
  "llm": {
    "local": {
      "enabled": true,
      "model_path": "~/.local/share/ai-work-studio/models",
      "default_model": "llama-2-7b-chat",
      "context_length": 4096,
      "gpu_acceleration": true,
      "memory_limit": "8GB"
    }
  }
}
```

**Model Management**:
```bash
# Download recommended models
ai-work-studio --download-models

# List available models
ai-work-studio --list-models

# Set default model
ai-work-studio --set-default-model llama-2-13b-chat
```

**Performance Tuning**:
```json
{
  "local": {
    "performance": {
      "batch_size": 8,
      "threads": 4,
      "gpu_layers": 35,
      "context_length": 4096,
      "temperature": 0.7
    }
  }
}
```

### Anthropic Claude

**Benefits**: High quality, excellent reasoning, good for complex tasks
**Drawbacks**: Costs per use, requires internet connection

**Setup**:
1. **Get API Key**: Sign up at [console.anthropic.com](https://console.anthropic.com/)
2. **Store Key Securely**:
   ```bash
   # Option 1: Environment variable
   export ANTHROPIC_API_KEY="your-key-here"

   # Option 2: Key file (more secure)
   echo "your-key-here" > ~/.config/ai-work-studio/keys/anthropic.key
   chmod 600 ~/.config/ai-work-studio/keys/anthropic.key
   ```

3. **Configure Provider**:
   ```json
   {
     "llm": {
       "anthropic": {
         "enabled": true,
         "model": "claude-3-sonnet-20240229",
         "max_tokens": 4096,
         "temperature": 0.3,
         "timeout": 30000
       }
     }
   }
   ```

### OpenAI

**Benefits**: Wide model selection, good for creative tasks
**Drawbacks**: Costs per use, rate limits

**Setup**:
1. **Get API Key**: Sign up at [platform.openai.com](https://platform.openai.com/api-keys)
2. **Configure**:
   ```json
   {
     "llm": {
       "openai": {
         "enabled": true,
         "model": "gpt-4",
         "max_tokens": 4096,
         "temperature": 0.4,
         "organization": "optional-org-id"
       }
     }
   }
   ```

### Provider Routing

Configure which provider to use for different types of tasks:

```json
{
  "llm": {
    "routing": {
      "rules": [
        {
          "condition": "token_count < 1000 AND privacy_level != 'high'",
          "provider": "local"
        },
        {
          "condition": "task_type == 'analysis' AND budget_available > 2.0",
          "provider": "anthropic"
        },
        {
          "condition": "task_type == 'creative'",
          "provider": "openai"
        },
        {
          "condition": "privacy_level == 'high'",
          "provider": "local"
        }
      ],
      "default": "local",
      "fallback": "anthropic"
    }
  }
}
```

## Budget Management

### Setting Limits

```json
{
  "budget": {
    "daily_limit": 25.0,
    "weekly_limit": 150.0,
    "monthly_limit": 500.0,
    "yearly_limit": 5000.0,
    "currency": "USD",

    "thresholds": {
      "warn_at": 80.0,
      "pause_at": 95.0,
      "hard_stop": true
    },

    "by_provider": {
      "anthropic": {
        "daily_limit": 15.0,
        "monthly_limit": 300.0
      },
      "openai": {
        "daily_limit": 10.0,
        "monthly_limit": 200.0
      }
    }
  }
}
```

### Cost Optimization

**Automatic Optimization**:
```json
{
  "budget": {
    "optimization": {
      "prefer_local": true,
      "cache_responses": true,
      "batch_similar_requests": true,
      "use_smaller_models_when_possible": true,
      "compress_context": true
    }
  }
}
```

**Manual Controls**:
```bash
# Check current budget usage
ai-work-studio --budget-status

# Pause spending for today
ai-work-studio --pause-spending

# Reset budget counters (new billing period)
ai-work-studio --reset-budget monthly
```

## Storage Configuration

### File-Based Storage (Default)

```json
{
  "storage": {
    "type": "file",
    "data_directory": "~/.config/ai-work-studio",
    "structure": {
      "nodes": "nodes",
      "edges": "edges",
      "methods": "methods",
      "backups": "backups",
      "cache": "cache"
    },

    "performance": {
      "cache_size": "100MB",
      "index_enabled": true,
      "compression": "gzip",
      "async_writes": true
    },

    "maintenance": {
      "auto_cleanup": true,
      "retention_policy": "90d",
      "vacuum_interval": "7d"
    }
  }
}
```

### Backup Configuration

```json
{
  "backup": {
    "enabled": true,
    "interval": "24h",
    "retention": {
      "daily": 7,
      "weekly": 4,
      "monthly": 12
    },

    "locations": [
      "local",
      "cloud"
    ],

    "cloud": {
      "provider": "none",
      "encryption": true,
      "compression": true
    },

    "verification": {
      "enabled": true,
      "test_restore": "weekly"
    }
  }
}
```

### Data Migration

**To new location**:
```bash
# Export current data
ai-work-studio --export-data /path/to/backup.tar.gz

# Change data directory in config
# Import to new location
ai-work-studio --import-data /path/to/backup.tar.gz
```

**To different format** (future AmorphDB):
```bash
# Export in migration format
ai-work-studio --export-for-migration /path/to/migration.json

# When AmorphDB backend is available:
ai-work-studio --migrate-to-amorphdb /path/to/migration.json
```

## Interface Preferences

### Theme and Appearance

```json
{
  "ui": {
    "theme": "auto",  // "light", "dark", "auto"
    "color_scheme": "default",  // "default", "high_contrast", "colorblind"
    "font_size": "medium",  // "small", "medium", "large", "extra_large"
    "font_family": "system",  // "system", "monospace", "sans_serif"

    "layout": {
      "panel_sizes": "balanced",  // "compact", "balanced", "spacious"
      "toolbar_position": "top",  // "top", "bottom", "side"
      "sidebar_width": 300
    }
  }
}
```

### Startup Behavior

```json
{
  "ui": {
    "startup": {
      "view": "goals",  // "goals", "objectives", "status", "last_used"
      "restore_session": true,
      "auto_open_agent": false,
      "check_for_updates": true
    }
  }
}
```

### Notifications

```json
{
  "notifications": {
    "enabled": true,
    "types": {
      "objective_completed": true,
      "goal_milestone": true,
      "budget_warnings": true,
      "system_errors": true,
      "method_suggestions": false
    },

    "delivery": {
      "desktop": true,
      "in_app": true,
      "sound": false
    },

    "quiet_hours": {
      "enabled": true,
      "start": "22:00",
      "end": "08:00"
    }
  }
}
```

## Performance Tuning

### System Resources

```json
{
  "performance": {
    "memory": {
      "max_usage": "2GB",
      "cache_size": "512MB",
      "swap_threshold": 80
    },

    "cpu": {
      "max_threads": 4,
      "background_priority": "low",
      "throttle_when_idle": true
    },

    "disk": {
      "cache_writes": true,
      "compression": true,
      "async_io": true
    }
  }
}
```

### Network Configuration

```json
{
  "network": {
    "timeout": 30000,
    "retry_attempts": 3,
    "proxy": {
      "enabled": false,
      "http_proxy": "",
      "https_proxy": "",
      "no_proxy": "localhost,127.0.0.1"
    },

    "rate_limiting": {
      "enabled": true,
      "requests_per_minute": 60
    }
  }
}
```

### Concurrent Processing

```json
{
  "concurrency": {
    "max_concurrent_objectives": 2,
    "max_background_tasks": 5,
    "thread_pool_size": 8,
    "batch_processing": true
  }
}
```

## Security and Privacy

### Data Protection

```json
{
  "security": {
    "encryption": {
      "at_rest": true,
      "algorithm": "AES-256-GCM",
      "key_derivation": "PBKDF2"
    },

    "access_control": {
      "require_auth": false,
      "session_timeout": "24h",
      "auto_lock": true
    },

    "audit": {
      "log_access": true,
      "log_modifications": true,
      "retention": "1y"
    }
  }
}
```

### Privacy Settings

```json
{
  "privacy": {
    "data_sharing": {
      "telemetry": false,
      "crash_reports": true,
      "usage_analytics": false,
      "method_sharing": false
    },

    "processing": {
      "prefer_local": true,
      "anonymize_cloud_requests": true,
      "cache_cloud_responses": false
    },

    "exports": {
      "anonymize_personal_data": true,
      "exclude_sensitive_goals": true,
      "strip_metadata": true
    }
  }
}
```

### API Key Management

```bash
# Secure key storage
mkdir -p ~/.config/ai-work-studio/keys
chmod 700 ~/.config/ai-work-studio/keys

# Store keys securely
echo "your-anthropic-key" | gpg --encrypt > ~/.config/ai-work-studio/keys/anthropic.key.gpg
echo "your-openai-key" | gpg --encrypt > ~/.config/ai-work-studio/keys/openai.key.gpg

# Configure to use encrypted keys
```

```json
{
  "security": {
    "api_keys": {
      "encryption": "gpg",
      "key_rotation": "90d",
      "auto_validate": true
    }
  }
}
```

## Advanced Configuration

### Plugin System (Future)

```json
{
  "plugins": {
    "enabled": true,
    "directory": "~/.config/ai-work-studio/plugins",
    "auto_update": false,
    "security_mode": "strict",

    "installed": [
      "ai-work-studio-calendar-integration",
      "ai-work-studio-git-methods",
      "ai-work-studio-email-tools"
    ]
  }
}
```

### Custom Method Templates

```json
{
  "methods": {
    "templates_directory": "~/.config/ai-work-studio/method-templates",
    "auto_import": true,
    "sharing": {
      "enabled": true,
      "anonymize": true,
      "community_repository": "official"
    }
  }
}
```

### Integration Webhooks

```json
{
  "integrations": {
    "webhooks": {
      "enabled": false,
      "endpoints": [
        {
          "event": "goal_completed",
          "url": "https://api.yourservice.com/goal-complete",
          "method": "POST",
          "auth": "bearer_token"
        }
      ]
    }
  }
}
```

### Custom LLM Providers

```json
{
  "llm": {
    "custom_providers": [
      {
        "name": "company_internal",
        "type": "openai_compatible",
        "base_url": "https://ai.company.com/v1",
        "api_key_env": "COMPANY_AI_API_KEY",
        "models": ["company-gpt-4", "company-claude"]
      }
    ]
  }
}
```

## Configuration Validation

### Check Configuration

```bash
# Validate current configuration
ai-work-studio --config-check

# Test LLM connections
ai-work-studio --test-llm-providers

# Verify backup configuration
ai-work-studio --test-backup

# Check security settings
ai-work-studio --security-audit
```

### Configuration Templates

**Minimal Setup**:
```json
{
  "version": "1.0",
  "llm": {
    "default_provider": "local"
  }
}
```

**Power User Setup**:
```json
{
  "version": "1.0",
  "llm": {
    "default_provider": "anthropic",
    "routing": { /* complex routing rules */ }
  },
  "budget": {
    "daily_limit": 50.0,
    "optimization": { /* all optimizations enabled */ }
  },
  "storage": {
    "performance": { /* high-performance settings */ }
  }
}
```

**Privacy-Focused Setup**:
```json
{
  "version": "1.0",
  "llm": {
    "default_provider": "local"
  },
  "privacy": {
    "prefer_local": true,
    "data_sharing": false,
    "anonymize_exports": true
  },
  "security": {
    "encryption": { /* full encryption */ }
  }
}
```

## Getting Help

- **Configuration Errors**: See [troubleshooting guide](troubleshooting.md)
- **Performance Issues**: Check logs and system resource usage
- **Security Questions**: Review privacy and security sections above
- **Custom Setups**: Join community discussions for advanced configurations

---

**Next**: Troubleshoot common issues with our [troubleshooting guide](troubleshooting.md).