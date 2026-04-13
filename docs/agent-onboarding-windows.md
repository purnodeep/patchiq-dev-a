# Onboarding a Windows Endpoint

This runbook covers installing the PatchIQ agent on a Windows machine
(Windows 10, Windows 11, or Windows Server 2019+).

## Requirements

- Administrator account on the target machine
- Network access from the target machine to your PatchIQ server
  (the public address is baked into the binary you download)
- ~50 MB free disk space

## Steps

1. **Generate a registration token.**
   In the PatchIQ web UI, open **Agent Downloads**, click
   **Generate registration token**, and copy the token shown
   (it looks like `K7M-3PQ-9XR`).

2. **Download `patchiq-agent.exe`.**
   On the same Agent Downloads page, click **Download Windows agent**.

3. **Copy the file to the target machine.**
   Any method works — USB, network share, RDP file transfer.

4. **Right-click `patchiq-agent.exe` → Run as administrator.**
   Windows will prompt for elevation (UAC). Click **Yes**.

   On unsigned builds, Windows SmartScreen may show "Windows protected
   your PC". Click **More info** then **Run anyway**.

5. **Paste the token in the wizard.**
   A small terminal window appears with the PatchIQ Agent Setup wizard.
   Paste your token and press Enter.

6. **Wait for the wizard to finish.**
   The wizard will:
   - Connect to the PatchIQ server
   - Register this endpoint
   - Install the `PatchIQAgent` Windows service
   - Start the service

   When you see "Setup complete!", close the window. The service is now
   running in the background and will start automatically on every boot.

7. **Verify in PatchIQ.**
   In the web UI, open **Endpoints**. Your new machine should appear
   within ~30 seconds, marked online.

## Uninstalling

There is no Add/Remove Programs entry in the current version. To remove
the agent:

1. Open an Administrator PowerShell.
2. Run: `& "C:\Path\To\patchiq-agent.exe" service uninstall`
3. Delete `C:\Program Files\PatchIQ\` (or wherever you placed the binary).
4. Delete `C:\ProgramData\PatchIQ\` (config + local database).

## Troubleshooting

- **"Must be run as Administrator"** — close the wizard, right-click the
  binary again and choose "Run as administrator".
- **"connection refused" / "no route to host"** — your machine cannot
  reach the PatchIQ server. Verify with `Test-NetConnection
  <server-address> -Port <port>`. Check your firewall and proxy.
- **Wizard appears but token is rejected** — the token may have expired
  (24h lifetime) or already been used. Generate a fresh one in the UI.
- **Endpoint doesn't appear in UI after 1 minute** — open
  `C:\ProgramData\PatchIQ\agent.log` and look for errors. The most
  common cause is a network issue between the machine and the server.
