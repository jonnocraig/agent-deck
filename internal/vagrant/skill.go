package vagrant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetVagrantSudoSkill returns the markdown content for the operating-in-vagrant skill.
// This skill tells Claude it's in an isolated VM with full sudo/Docker access and should
// act boldly — the VM is a disposable sandbox. Follows Claude skill best practices:
// concise (assumes Claude knows Linux/Docker), actionable (capabilities first), and
// includes the critical constraints (inotify, credentials).
func GetVagrantSudoSkill() string {
	return `---
name: operating-in-vagrant
description: Provides context for operating Claude Code inside an isolated Vagrant VM with full sudo, Docker, and unrestricted permissions. Covers capabilities, networking, file system constraints, and credential safety. Loaded automatically for vagrant-mode sessions.
---

# Operating in a Vagrant VM

You are running inside an isolated Ubuntu 24.04 virtual machine with ` + "`--dangerously-skip-permissions`" + `.
This is intentional — the VM provides isolation, so you have full freedom to act.

## Capabilities

You have **unrestricted access**:
- ` + "`sudo`" + ` without password for any command
- Install system packages: ` + "`sudo apt-get install -y <pkg>`" + `
- Run Docker containers: ` + "`docker run ...`" + ` (user is in docker group)
- Modify any system config under ` + "`/etc/`" + `
- Listen on any port, create users, change permissions
- Install global packages: ` + "`npm install -g`" + `, ` + "`pip install`" + `, ` + "`cargo install`" + `

## Environment

| | |
|---|---|
| OS | Ubuntu 24.04 LTS |
| Project | ` + "`/vagrant`" + ` (bidirectional sync with host) |
| Host access | ` + "`10.0.2.2`" + ` (VirtualBox NAT gateway) |
| Pre-installed | Docker, Node.js, npm, Git, build-essential, curl |
| Privileges | Full sudo, no password required |

## Constraints

### VirtualBox shared folders break inotify

File watchers do not work on ` + "`/vagrant`" + `. Always use polling:

` + "```bash" + `
CHOKIDAR_USEPOLLING=1 npm run dev          # Webpack/Vite
WATCHPACK_POLLING=true npm run dev          # Next.js
tsc --watch --watchFile useFsEvents --fallbackPolling fixedInterval  # TypeScript
` + "```" + `

### NEVER read credential files

Do not read, cat, or log: ` + "`.env*`, `.npmrc`, `credentials.json`, `*.pem`, `*.key`, `id_rsa*`, `.netrc`, `.aws/credentials`, `.docker/config.json`" + `.

Secrets are available via environment variables forwarded through SSH.
If you need a secret, ask the user to set it in ` + "`[vagrant.env]`" + ` in their agent-deck config.

## Common Patterns

` + "```bash" + `
# Database
docker run -d --name pg -p 5432:5432 -e POSTGRES_PASSWORD=dev postgres:16

# Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# System dependencies
sudo apt-get update && sudo apt-get install -y postgresql-client redis-tools jq

# Access host machine services
curl http://10.0.2.2:3000/api/health
` + "```" + `

## Mindset

This VM is a disposable sandbox. Be bold:
- Try approaches that would be risky on a real machine
- Install packages and experiment freely
- If something breaks, the VM can be destroyed and rebuilt
- Focus on solving the task, not protecting the environment
- Changes in ` + "`/vagrant`" + ` persist on the host — everything else is ephemeral
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

	skillPath := filepath.Join(skillsDir, "operating-in-vagrant.md")
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
