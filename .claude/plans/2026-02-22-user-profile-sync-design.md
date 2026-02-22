# User Profile Sync to Vagrant VM — Design Document

> Generated: 2026-02-22
> Brainstorm perspectives: Architect, Implementer, Devil's Advocate, Security Analyst
> Chosen approach: Go archive/tar with security hardening

## Summary

Sync the user's full Claude Code profile (skills, commands, agents, rules) from the host into the Vagrant VM via a single in-memory tar archive piped over SSH. Merge the user's host CLAUDE.md with VM-specific instructions (VM context first, user content appended).

## Acceptance Criteria

- All 4 profile directories synced to VM: skills/, commands/, agents/, rules/
- Symlinks dereferenced (resolved to real file content in tar)
- Symlink targets validated (must resolve within ~/.claude/ or ~/.agents/)
- Broken/cyclic symlinks skipped with log warning
- .DS_Store, .git excluded from tar
- CLAUDE.md excluded from tar (handled separately in step 6)
- operating-in-vagrant/ excluded from tar (VM-managed by EnsureSudoSkill)
- 100MB size limit on dereferenced content
- Post-extract permission normalization (644 files, 755 dirs)
- CLAUDE.md merge: VM context first, separator, user content appended
- Missing directories handled gracefully (skip, no error)
- All errors non-fatal (matching steps 1-5 pattern)
- Tests: 80%+ coverage on new code

## Non-Goals

- Secret scanning of profile files (user's own content, not our concern)
- Incremental/delta sync (always full sync on each SyncClaudeConfig call)
- Compressing the tar (gzip unnecessary for <1MB payload)
- Syncing ~/.claude/settings.local.json (VM-managed)
- Syncing ~/.claude/.credentials.json (handled separately in step 5)

## Context & Constraints

- Go 1.24, using stdlib archive/tar (no external dependencies)
- Manager struct has writeFileToVMFunc injectable + vagrantCmd() helper
- Package-level injectables for testing (extractOAuthCredentialsFunc pattern)
- Current SyncClaudeConfig() has 6 steps, all non-fatal
- User has ~94 files across 4 dirs, some skills are symlinks to ~/.agents/skills/
- VM is Ubuntu 24.04 with vagrant user, ~/.claude/ may not pre-exist

## Exploration Findings

### Perspective: Architect
- Single tar for all 4 dirs (not separate tars) — simpler, atomic
- Direct stdin pipe via cmd.StdinPipe() — no base64, no temp files
- Three-phase SyncClaudeConfig: critical configs (1-5), CLAUDE.md merge (6), batch tar (7)
- Package-level injectable createProfileTarFunc for testing

### Perspective: Implementer
- Go archive/tar with bytes.Buffer for in-memory creation
- os.Stat() (not os.Lstat()) for symlink dereferencing
- filepath.Rel() from ~/.claude/ for correct tar paths
- filepath.WalkDir for directory traversal
- cmd.Start() before stdin.Write() to avoid deadlock
- New file sync_profile.go (~150 lines) + sync_profile_test.go (~300 lines)

### Perspective: Devil's Advocate
- CRITICAL: symlinks could point to huge dirs or sensitive files
- CRITICAL: tar extract could clobber VM's operating-in-vagrant/ skill
- Tar must exclude CLAUDE.md to avoid overwriting step 6's merged version
- 94 individual SSH calls (~19s) is acceptable alternative but tar is cleaner
- Missing dirs and empty tar must be handled gracefully

### Perspective: Security Analyst
- Symlink path validation is the main security surface
- Restrict targets to ~/.claude/ and ~/.agents/ prefixes
- Tar extraction is safe since we create the tar ourselves
- Post-extract chmod normalization prevents restrictive host perms leaking
- No secret scanning needed (user's own markdown content)
- vagrant ssh is encrypted, stdin pipe doesn't leak content to ps

## Approaches Considered

### Approach 1: Go archive/tar with Security Hardening (Selected)
In-memory tar via Go stdlib, stdin pipe to vagrant ssh, symlink validation.
- Pros: Fast (~2s), single SSH call, Go-native, full control over content
- Cons: ~150 lines new code, symlink validation complexity

### Approach 2: Simple Per-File Sync
Reuse writeFileToVM() for each file, skip skills with symlinks.
- Pros: Zero new infrastructure, proven pattern
- Cons: ~61 SSH calls (~30s), doesn't sync symlinked skills

### Approach 3: External tar Binary Pipe
Shell out to host tar -chf piped to vagrant ssh.
- Pros: Proven Unix approach, native -h flag
- Cons: Platform differences (GNU vs BSD), harder to test

## Design

### Architecture

```
SyncClaudeConfig() — 7 Steps
├── Steps 1-5: Individual file syncs (unchanged)
│   ├── 1. Global config (~/.claude/.claude.json)
│   ├── 2. User config (~/.claude.json)
│   ├── 3. Settings (~/.claude/settings.json)
│   ├── 4. Statusline (~/.claude/statusline.sh)
│   └── 5. OAuth credentials (~/.claude/.credentials.json)
├── Step 6: CLAUDE.md merge (MODIFIED)
│   └── getMergedVMClaudeMD() → VM context + separator + user CLAUDE.md
└── Step 7: Batch tar sync (NEW)
    └── syncProfileToVM() → createProfileTar() | vagrant ssh tar xf
```

```
Host                                VM
┌─────────────────────┐           ┌─────────────────────┐
│ ~/.claude/           │           │ ~/.claude/           │
│ ├── skills/ (33)    │  step 7   │ ├── skills/ (33)    │
│ ├── commands/ (33)  │──────────▶│ ├── commands/ (33)  │
│ ├── agents/ (19)    │ tar pipe  │ ├── agents/ (19)    │
│ └── rules/ (9)      │ 1 SSH     │ └── rules/ (9)      │
│                      │           │                      │
│ ├── CLAUDE.md       │  step 6   │ ├── CLAUDE.md       │
│ │ (user content)    │──────────▶│ │ (VM + user merged)│
└─────────────────────┘           └─────────────────────┘
```

### Components

New file: `internal/vagrant/sync_profile.go`

| Function | Purpose |
|----------|---------|
| `createProfileTar(homeDir string) ([]byte, error)` | Build in-memory tar of 4 profile dirs |
| `shouldSkipEntry(name string) bool` | Filter .DS_Store, .git, etc. |
| `isExcludedProfilePath(relPath string) bool` | Exclude CLAUDE.md, operating-in-vagrant/ |
| `validateSymlinkTarget(path, homeDir string) error` | Ensure symlink within allowed prefixes |
| `(m *Manager) syncProfileToVM() error` | Pipe tar to VM, normalize permissions |

Modified: `internal/vagrant/sync.go`

| Function | Change |
|----------|--------|
| `getVMClaudeMD()` → `getMergedVMClaudeMD()` | Append host CLAUDE.md after VM context |
| `SyncClaudeConfig()` | Call syncProfileToVM() as step 7 |

Injectable: `var createProfileTarFunc = createProfileTar` (package-level)

### Error Handling

All step 6-7 errors are non-fatal (log and continue).

| Scenario | Behavior |
|----------|----------|
| ~/.claude/ missing | Skip step 7, return nil |
| All 4 dirs missing | Empty tar, skip SSH, return nil |
| Broken symlink | Log warning, skip entry |
| Symlink outside allowed | Log warning, skip entry |
| Content exceeds 100MB | Return error, logged non-fatal |
| SSH pipe breaks | Return error, logged non-fatal |
| Host CLAUDE.md missing | VM-only content in step 6 |

### Testing Strategy

New file: `internal/vagrant/sync_profile_test.go`

| Test | Verifies |
|------|----------|
| TestCreateProfileTar_BasicStructure | All 4 dirs with correct relative paths |
| TestCreateProfileTar_DereferencesSymlinks | Symlinked dir becomes real files |
| TestCreateProfileTar_SkipsDSStore | .DS_Store excluded |
| TestCreateProfileTar_ExcludesCLAUDEMD | CLAUDE.md not in tar |
| TestCreateProfileTar_ExcludesOperatingInVagrant | skills/operating-in-vagrant/ excluded |
| TestCreateProfileTar_MissingDirsGraceful | Only existing dirs, no error |
| TestCreateProfileTar_AllDirsMissing | Empty bytes, no error |
| TestCreateProfileTar_BrokenSymlinkSkipped | Skipped with warning |
| TestCreateProfileTar_SymlinkOutsideAllowed | Skipped with warning |
| TestCreateProfileTar_SizeLimitEnforced | Error on >100MB |
| TestShouldSkipEntry | Table test for skip patterns |
| TestIsExcludedProfilePath | Table test for exclusions |
| TestValidateSymlinkTarget | Table test for allowed/disallowed |

Modified: `internal/vagrant/sync_test.go`

| Test | Verifies |
|------|----------|
| TestGetMergedVMClaudeMD_WithHostFile | VM first, separator, user appended |
| TestGetMergedVMClaudeMD_NoHostFile | VM-only content |
| TestSyncClaudeConfig_CallsProfileSync | Step 7 invoked |

### Security Model

- Symlink targets restricted to ~/.claude/ and ~/.agents/
- Broken/cyclic symlinks skipped (filepath.EvalSymlinks error)
- .DS_Store, .git excluded
- CLAUDE.md and operating-in-vagrant/ excluded (VM-managed)
- 100MB size limit
- Post-extract chmod 644 (files) / 755 (dirs)
- No secret scanning (user's own content)
- SSH transport encrypted

## Next Steps

- [ ] Create implementation plan (use `agentic-ai-plan`)
- [ ] Implement sync_profile.go + sync_profile_test.go
- [ ] Modify sync.go step 6 (CLAUDE.md merge) + add step 7
- [ ] Update sync_test.go for new behavior
- [ ] Rebuild binary and run full test suite
- [ ] Fresh VM test (vagrant destroy + vagrant up)
- [ ] Verify skills/commands/agents/rules visible in VM

## Follow-Up Tasks (Outside This Feature)

- [ ] Increase default VM RAM from 4GB to 16GB (settings.MemoryMB in vagrantfile.go)
