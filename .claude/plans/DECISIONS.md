# Architecture Decisions

## 2026-02-22 - Skill Directory Format for Claude Code Discovery

**Context**: `EnsureSudoSkill()` wrote the operating-in-vagrant skill as a standalone `operating-in-vagrant.md` file. Claude Code v2.1.50 only discovers skills as directories with `SKILL.md` as entrypoint (`<skill-name>/SKILL.md`). The standalone file was invisible to Claude Code.
**Decision**: Changed `EnsureSudoSkill()` to write `operating-in-vagrant/SKILL.md` (directory format). Added legacy cleanup that removes `operating-in-vagrant.md` and `vagrant-sudo.md` on each invocation.
**Consequences**: Claude Code properly discovers the skill. Legacy files from previous sessions cleaned up automatically. Tests updated to expect new path structure.

## 2026-02-22 - Pre-load VM Context via ~/.claude/CLAUDE.md

**Context**: Claude Code lazy-loads skills (descriptions in context, full content only when invoked). The operating-in-vagrant skill was discovered but not pre-loaded — users had to ask Claude to invoke it.
**Decision**: Added step 6 to `SyncClaudeConfig()` that writes `~/.claude/CLAUDE.md` inside the VM with the skill body (YAML frontmatter stripped). CLAUDE.md is always loaded in full by Claude Code at session start. The skill directory remains for `/operating-in-vagrant` slash command access.
**Consequences**: VM context is immediately available to Claude without user interaction. Dual presence (CLAUDE.md + skill) provides both automatic context and on-demand invocation.

## 2026-02-22 - User Profile Sync via Go archive/tar with Security Hardening

**Context**: VM sessions only sync 6 items (configs, settings, statusline, OAuth, VM CLAUDE.md). User's skills (33), commands (33), agents (19), and rules (9) are missing. Syncing 94 files individually via `writeFileToVM()` would take ~19 seconds. 4-perspective brainstorm (Architect, Implementer, Devil's Advocate, Security) evaluated 3 approaches.
**Decision**: Go `archive/tar` in-memory with stdin pipe to `vagrant ssh -c "tar xf - -C ~/.claude/"`. Symlink targets validated (must resolve within ~/.claude/ or ~/.agents/). Broken/cyclic symlinks skipped. .DS_Store, CLAUDE.md, operating-in-vagrant/ excluded from tar. 100MB size limit. Post-extract chmod normalization. CLAUDE.md merge: VM context first, user content appended. New file `sync_profile.go` + tests. Package-level injectable `createProfileTarFunc`.
**Consequences**: Full user profile in VM via single SSH call (~2s). Symlinks safely dereferenced. Design doc at `.claude/plans/2026-02-22-user-profile-sync-design.md`.

## 2026-02-22 - OAuth Credential Forwarding for Vagrant VM

**Context**: Max subscription users authenticate via OAuth, not API keys. Vagrant VM sessions stripped `oauthAccount` from config files and only forwarded `ANTHROPIC_API_KEY` via SSH, so Max users hit a login prompt every time Claude Code started in the VM.
**Decision**: Three-pronged approach: (1) File-based: extract OAuth credentials from host (macOS Keychain via `security find-generic-password` or `~/.claude/.credentials.json` on Linux), write as `~/.claude/.credentials.json` inside VM with `chmod 600`. (2) Env var: forward `CLAUDE_CODE_OAUTH_TOKEN` via SSH `SendEnv` alongside `ANTHROPIC_API_KEY`. (3) Onboarding bypass: inject `hasCompletedOnboarding: true` into both global and user configs synced to VM via new `injectVMFields()` function.
**Consequences**: Max users authenticate seamlessly in VM. Access + refresh tokens synced so auto-refresh works natively. Env var override available for CI/manual flows. Injectable `extractOAuthCredentialsFunc` follows existing `getAvailableMCPsFunc` pattern for clean testing. Credentials never logged (only success/failure).

## 2026-02-21 - Config Sync: Strip host-only fields with generic stripJSONKeys()

**Context**: Screenshots from VM testing showed 5 errors: `installMethod is native` (directory/binary not found), `22 plugins failed to install`, `Not logged in`, `~/.local/bin not in PATH`. Root cause: host config files synced verbatim to VM contain fields that reference host-specific state.
**Decision**: Refactored `stripMCPServers()` into generic `stripJSONKeys(data, keys)`. Added `stripHostOnlyFields()` (strips `mcpServers`, `installMethod`, `oauthAccount` from ~/.claude.json) and `stripSettingsForVM()` (strips `enabledPlugins`, `hooks` from settings.json).
**Consequences**: VM gets clean config. `installMethod` auto-detected by Claude Code. No plugin install failures. Host hooks don't pollute VM. OAuth state removed (API key auth via ANTHROPIC_API_KEY env var forwarded through SSH SendEnv). Belt-and-suspenders: Vagrantfile provisioning also creates `~/.local/bin` and symlinks claude binary.

## 2026-02-21 - Skill Rewrite: "operating-in-vagrant" (Supercharged Claude)

**Context**: Existing `vagrant-sudo` skill was functional but didn't capture the "Supercharged Claude" mindset from emilburzo's blog post. Skill best practices (platform.claude.com) recommend: concise, gerund naming, capabilities-first, proper discovery metadata.
**Decision**: Rewrote skill as `operating-in-vagrant`. Leads with unrestricted capabilities (sudo, Docker, system config, global packages). Includes host networking (10.0.2.2), common Docker patterns, and a "Mindset" section encouraging bold experimentation. Renamed file from `vagrant-sudo.md` to `operating-in-vagrant.md`.
**Consequences**: Claude inside VM should be more proactive about using sudo, Docker, and system tools. Better skill discovery via descriptive frontmatter. Credential guard hook still enforced separately via settings.local.json.

## 2026-02-21 - MCP in Vagrant VM: Use @modelcontextprotocol packages via STDIO

**Context**: MCP tools were not loading inside the Vagrant VM. Host-side configs synced into VM referenced `agent-deck mcp-proxy` and Unix pool sockets that don't exist in the VM. 14 MCP servers from `~/.claude.json` all pointed to host-only binaries.
**Decision**: Clean all `mcpServers` from global/user Claude configs inside VM. Install official `@modelcontextprotocol/server-*` npm packages globally. Configure project-level `.mcp.json` with STDIO transport using `npx -y` invocations.
**Consequences**: MCP servers run natively inside VM. No dependency on host-side agent-deck or pool sockets. Packages cached globally via `npm install -g` to avoid repeated npx downloads. Node.js 18.x works despite engine warnings for 20+.

## 2026-02-21 - Vagrant Command Wrapping: Simplified tmux-free command

**Context**: `buildClaudeCommand()` generates commands containing `tmux set-environment CLAUDE_SESSION_ID "$session_id"` which fails inside the VM (no tmux server). Error: `error connecting to /tmp/tmux-1000/default`.
**Decision**: Add `buildVagrantClaudeCommand()` that produces a clean command using `export` instead of `tmux set-environment`. Called from `applyVagrantWrapper` instead of wrapping the tmux-dependent command.
**Consequences**: No tmux errors inside VM. Session ID available via env var. Host-side tmux session ID capture still works (via `tmux.SetEnvironment` after `Start()`).

## 2026-02-14 - Vagrant Mode: Wrapper Command Approach (Approach 1)

**Context**: User wants a "Just do it" checkbox that spawns Claude Code in an isolated Vagrant VM with `--dangerously-skip-permissions` and sudo access. Three approaches evaluated by multi-perspective brainstorm.
**Decision**: Wrapper Command approach -- add checkbox, auto-manage VM lifecycle, wrap commands via `vagrant ssh -c`. No provider abstraction, no security hardening beyond VM isolation.
**Consequences**: Minimal complexity (4 modified + 3 new files). VirtualBox dependency. First boot latency (5-10 min). Bidirectional sync risk accepted as intentional.
