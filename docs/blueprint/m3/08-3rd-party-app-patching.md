# 3rd-Party App Patching

**Status**: Planned
**Wave**: 2 — Automation & Extensibility
**Dependencies**: Hub catalog, binary distribution, agent patcher module

---

## Vision

Detect, manage, and patch the top 25 third-party applications across Windows, macOS, and Linux. Most enterprise security breaches exploit outdated third-party software, not OS vulnerabilities.

## Deliverables

### Application Detection
- [ ] Detection rules per app: registry keys (Win), plist/spotlight (Mac), dpkg/rpm/desktop files (Linux)
- [ ] Version extraction from installed application metadata
- [ ] Application inventory visible in endpoint detail Software tab

### Top 25 Applications
- [ ] Chrome, Firefox, Edge
- [ ] Adobe Reader, Adobe Acrobat
- [ ] 7-Zip, WinRAR
- [ ] Notepad++, Sublime Text, VS Code
- [ ] VLC, Media Player
- [ ] Java JRE/JDK, .NET Runtime
- [ ] Python, Node.js
- [ ] Zoom, Teams (desktop), Slack (desktop)
- [ ] PuTTY, WinSCP, FileZilla
- [ ] Git, Docker Desktop

### Silent Install Definitions
- [ ] Per-app silent install/upgrade command definitions
- [ ] Installer type detection: MSI, EXE (NSIS/Inno/InstallShield), DMG, PKG, DEB, RPM, AppImage
- [ ] Pre/post install scripts (e.g., close app before upgrade)
- [ ] Rollback: uninstall current + reinstall previous version

### Hub Catalog Integration
- [ ] Catalog entries for 3rd-party apps with version tracking
- [ ] Auto-detect new versions from vendor sites (deterministic parsing, no LLM)
- [ ] Binary storage in MinIO with SHA256 verification

## License Gating
- 3rd-party app detection: PROFESSIONAL+
- 3rd-party app patching: ENTERPRISE
- Custom app definitions: ENTERPRISE
