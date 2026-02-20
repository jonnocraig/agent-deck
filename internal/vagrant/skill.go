package vagrant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetVagrantSudoSkill returns the markdown content for the vagrant-sudo skill.
// This skill informs Claude that it's running in an isolated Ubuntu VM with sudo access.
func GetVagrantSudoSkill() string {
	return `---
name: vagrant-sudo
description: Guidelines for running Claude Code inside an isolated Vagrant VM with sudo access. Loaded automatically when a vagrant-mode session starts.
---

# Vagrant Sudo Skill

You are running in an isolated Ubuntu 24.04 LTS virtual machine with sudo access.

## VM Environment

- **Operating System**: Ubuntu 24.04 LTS (Jammy Jellyfish)
- **Privileges**: Full sudo access available
- **Pre-installed Tools**: Docker, Node.js, Git, build-essential, curl, wget
- **Project Location**: /vagrant (synced to host machine)
- **Isolation**: This VM can be destroyed and recreated at any time

## Important Constraints

### File Watchers (inotify) DO NOT WORK

File watchers (inotify) **do NOT work** on /vagrant with VirtualBox shared folders. Always use polling mode:

- **Webpack/Vite**: Set ` + "`CHOKIDAR_USEPOLLING=1`" + ` environment variable
- **Next.js**: Set ` + "`WATCHPACK_POLLING=true`" + ` environment variable
- **TypeScript**: Use ` + "`tsc --watch --poll`" + ` flag for watch mode
- **Alternative**: Switch to NFS shared folders (` + "`synced_folder_type = \"nfs\"`" + ` in Vagrantfile) for native inotify support

### Credential Files - NEVER READ OR TRANSMIT

**NEVER** read, cat, print, log, or transmit credential files:
- .env, .env.local, .env.production
- .npmrc, .yarnrc
- credentials.json, service-account.json
- *.pem, *.key, *.crt (private keys/certs)
- id_rsa, id_ed25519 (SSH keys)
- .netrc, .docker/config.json
- .aws/credentials, .gcloud/credentials

**Use environment variables for secrets** - they are forwarded via SSH.
If you need a secret, ask the user to add it to ` + "`[vagrant.env]`" + ` section in config.toml.

## Best Practices

1. **Use Docker for services** when possible (databases, Redis, etc.)
2. **Install system packages** with ` + "`sudo apt-get install`" + ` when needed
3. **Changes in /vagrant are reflected on the host** - no need to sync
4. **VM is ephemeral** - don't store important data outside /vagrant
5. **Use polling mode** for any file watchers (webpack, vite, tsc --watch)

## Example Commands

` + "```bash" + `
# Install a system package
sudo apt-get update && sudo apt-get install -y postgresql-client

# Run a Docker container
sudo docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=dev postgres:16

# Start a dev server with polling (Next.js example)
WATCHPACK_POLLING=true npm run dev

# TypeScript watch mode with polling
tsc --watch --poll
` + "```" + `
`
}

// EnsureSudoSkill writes the vagrant-sudo skill to the project's .claude/skills directory
// and injects the credential guard hook into .claude/settings.local.json.
// This method is idempotent and will overwrite files if they already exist.
func (m *Manager) EnsureSudoSkill() error {
	skillsDir := filepath.Join(m.projectPath, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	skillPath := filepath.Join(skillsDir, "vagrant-sudo.md")
	content := GetVagrantSudoSkill()

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write vagrant-sudo skill: %w", err)
	}

	// Inject the credential guard PreToolUse hook into settings.local.json
	if err := InjectCredentialGuardHook(m.projectPath); err != nil {
		return fmt.Errorf("failed to inject credential guard hook: %w", err)
	}

	return nil
}

// GetCredentialGuardHook returns the credential guard hook definition as a Go map.
// This hook blocks Claude from reading credential files in vagrant mode.
func GetCredentialGuardHook() map[string]interface{} {
	return map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Read|View|Cat",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": `bash -c 'FILE="$CLAUDE_TOOL_ARG_file_path"; PATTERNS=".env|.npmrc|credentials.json|.pem|.key|id_rsa|id_ed25519|.netrc|.docker/config.json|.aws/credentials|.gcloud/credentials"; echo "$FILE" | grep -qiE "($PATTERNS)" && echo "BLOCKED: Reading credential files is not allowed in vagrant mode. Use [vagrant.env] for secrets." && exit 1 || exit 0'`,
						},
					},
				},
			},
		},
	}
}

// InjectCredentialGuardHook writes or merges the credential guard hook into
// the project's .claude/settings.local.json file.
func InjectCredentialGuardHook(projectPath string) error {
	claudeDir := filepath.Join(projectPath, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	// Read existing settings if file exists
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, start with empty settings
			settings = make(map[string]interface{})
		} else {
			return fmt.Errorf("failed to read settings.local.json: %w", err)
		}
	} else {
		// File exists, parse it
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse settings.local.json: %w", err)
		}
	}

	// Get the credential guard hook
	guardHook := GetCredentialGuardHook()
	guardHooks := guardHook["hooks"].(map[string]interface{})
	guardPreToolUse := guardHooks["PreToolUse"].([]interface{})

	// Merge hooks section
	existingHooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		// No existing hooks, create new
		settings["hooks"] = guardHooks
	} else {
		// Merge PreToolUse hooks
		existingHooks["PreToolUse"] = guardPreToolUse
	}

	// Write back to file with pretty formatting
	data, err = json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.local.json: %w", err)
	}

	return nil
}
