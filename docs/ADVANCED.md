# Container-Make Documentation

## Lifecycle Hooks

Container-Make supports the following lifecycle hooks in order of execution:

### Execution Order

```
1. onCreateCommand     ← Container created for the first time
2. postCreateCommand   ← After onCreate, before start
3. postStartCommand    ← After container starts
4. postAttachCommand   ← After attaching to container
```

### Error Handling

- Each command's exit code is checked
- Non-zero exit codes will stop execution and report the error
- Timing information is displayed for each command
- Use `set -e` in your commands for strict error handling

### Example

```json
{
  "postCreateCommand": [
    "npm install",
    "npm run build"
  ],
  "postStartCommand": "echo 'Container started!'",
  "postAttachCommand": "zsh"
}
```

---

## Port Forwarding

### Supported Formats

| Format | Description |
|--------|-------------|
| `8080` | Forward port 8080 TCP |
| `8080/tcp` | Explicit TCP protocol |
| `53/udp` | UDP protocol |
| `8080:80` | Map host 8080 to container 80 |
| `8080:80/tcp` | Full specification |

### Port Conflict Detection

- Container-Make automatically checks if ports are in use
- Conflicting ports are skipped with a warning
- Use different host ports to avoid conflicts

---

## SSH & Git Credentials

### SSH Agent Forwarding (Automatic)

Container-Make automatically forwards SSH agent:

**Linux/Mac:**
```bash
# Ensure SSH agent is running
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa

# SSH agent is automatically mounted
cm run -- ssh-add -l
```

**Windows (1809+):**
```powershell
# Enable OpenSSH Agent
Get-Service ssh-agent | Set-Service -StartupType Automatic
Start-Service ssh-agent
ssh-add $env:USERPROFILE\.ssh\id_rsa
```

### Git Credential Helper

For HTTPS repositories, mount your Git credentials:

```json
{
  "mounts": [
    "source=${localEnv:HOME}/.gitconfig,target=/home/vscode/.gitconfig,type=bind"
  ],
  "containerEnv": {
    "GIT_AUTHOR_NAME": "Your Name",
    "GIT_AUTHOR_EMAIL": "you@example.com"
  }
}
```

---

## Build Cache (CI/CD)

Use environment variables for build caching:

### Local Cache
```bash
# Cache to local directory
CM_CACHE_FROM=type=local,src=/tmp/cm-cache \
CM_CACHE_TO=type=local,dest=/tmp/cm-cache \
cm prepare
```

### Registry Cache (GitHub Actions)
```yaml
- name: Build Dev Container
  env:
    CM_CACHE_FROM: type=registry,ref=ghcr.io/${{ github.repository }}/cache
    CM_CACHE_TO: type=registry,ref=ghcr.io/${{ github.repository }}/cache,mode=max
  run: cm prepare
```

### S3 Cache
```bash
CM_CACHE_FROM=type=s3,region=us-east-1,bucket=my-cache \
CM_CACHE_TO=type=s3,region=us-east-1,bucket=my-cache \
cm prepare
```

---

## Docker Compose

### Configuration

```json
{
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "runServices": ["app", "db", "redis"],
  "shutdownAction": "stopCompose"
}
```

### Service Mapping

Container-Make provides these compose helpers:

- `cm run` - Executes command in the main service
- `cm prepare` - Builds all services and pulls images

### Multiple Compose Files

```json
{
  "dockerComposeFile": [
    "docker-compose.yml",
    "docker-compose.dev.yml"
  ]
}
```

---

## DevContainer Features

### Using Features

```json
{
  "features": {
    "ghcr.io/devcontainers/features/go:1": {
      "version": "1.21"
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  }
}
```

### Feature Options

Features accept configuration options specific to each feature. See the feature's documentation for available options.

> **Note**: Feature installation script execution is currently in development. Features are parsed and downloaded but may require manual installation.
