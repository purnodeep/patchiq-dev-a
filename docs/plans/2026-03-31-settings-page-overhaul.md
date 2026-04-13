# Settings Page Overhaul — Design Spec

> **Date**: 2026-03-31 | **Status**: Approved | **Mockup**: `docs/blueprint/m3/mockups/26-settings-redesign.html`
> **POC-PLAN ref**: Section 4.2 (F1) — Settings Page Overhaul

---

## Problem

The current settings page is a single page with a 2-column grid of cards. Three of the four cards have fake "Save" buttons that show dismissive toasts ("managed through Zitadel console"). This creates a poor impression during client demos and erodes trust in the product's maturity.

## Solution

Replace the single-page grid with a **multi-page sidebar-takeover pattern**. When a user navigates to `/settings/*`, the app sidebar transforms to show settings-specific navigation with a "← Back" link. Each settings section gets its own route and full-width content area.

**No fake buttons.** Read-only sections get clear info banners explaining where configuration is managed. Every visible control either works or has transparent context about why it's read-only.

---

## Layout Pattern: Sidebar Takeover

When the user enters any `/settings/*` route:
1. The `AppSidebar` renders a **settings-specific navigation** instead of the main nav
2. A "← Back" link at the top returns to the previous page (or `/` as fallback)
3. The sidebar header shows "Settings" with platform context ("Patch Manager")
4. Navigation items are grouped: **Configuration** (General, Identity & Access, Notifications) and **Account** (License, Appearance)
5. Active section gets an accent-colored left bar indicator + highlighted background
6. Identity & Access shows a green status dot when Zitadel connection is healthy

**Implementation**: The `AppSidebar` component checks if the current route starts with `/settings`. If yes, it renders `SettingsSidebar` instead of the regular nav groups. No separate layout needed — same `AppLayout`, conditional sidebar content.

---

## Routes

| Route | Component | Content |
|-------|-----------|---------|
| `/settings` | Redirect → `/settings/general` | — |
| `/settings/general` | `GeneralSettingsPage` | Org name, timezone, date format, scan interval |
| `/settings/identity` | `IdentitySettingsPage` | Zitadel provider card, SSO URL (read-only), Client ID (masked), Test Connection, Role mappings |
| `/settings/notifications` | `NotificationSettingsPage` | Channel status cards (webhook/slack/email), quick preference toggles |
| `/settings/license` | `LicenseSettingsPage` | Tier badge, usage bar, expiry, feature entitlement matrix |
| `/settings/appearance` | `AppearanceSettingsPage` | Theme cards (dark/light), accent swatches, compact mode toggle |

---

## Section Details

### 1. General (editable)
- **Fields**: Organization Name (text), Timezone (select), Date Format (select), Default Scan Interval (select)
- **Layout**: Full-width form, timezone and date format side-by-side in 2-column row
- **Actions**: Save Changes (brand button) + Discard (ghost button)
- **Saved indicator**: Green checkmark "Saved" text with fadeInUp animation, shown left of buttons
- **API**: `useSettings()` + `useUpdateSettings()` — existing hooks, no changes needed
- **Validation**: Existing Zod schema (org_name required, valid timezone, valid format, valid interval)

### 2. Identity & Access (read-only with Test Connection)
- **Provider card**: Zitadel icon (orange gradient), name, "OIDC + PKCE" protocol, connection status pill
- **Info banner** (blue): "Identity settings are managed through the Zitadel admin console. Fields below are read-only."
- **Fields**: SSO URL (read-only input with badge), Client ID (masked with reveal toggle, read-only)
- **Test Connection**: Ghost button → calls `useTestIAMConnection()` → shows success/error result with latency
- **Role Mappings**: Table with columns: Zitadel Role (mono) → PatchIQ Role → Status (● Active badge)
- **API**: `useIAMSettings()` + `useTestIAMConnection()` — existing hooks
- **Key change**: Remove fake "Save" button entirely. No toast misdirection.

### 3. Notifications (read-only channels + editable preferences)
- **Channel grid**: 3-column grid of cards showing webhook, Slack, email status
  - Each card: icon, name, status dot (green=on, gray=off), URL/workspace, last tested, Test button
  - Test buttons call `useTestChannelByType()` — existing hook
- **Info banner**: "Channel configuration and event subscriptions are managed on the Notifications page →" (link)
- **Quick Preferences**: Toggle rows for common notification categories:
  - Deployment alerts, Critical CVE alerts, Compliance threshold breach, Agent disconnect, License expiry
  - Toggles are functional — save via a new lightweight preferences API or localStorage for POC
- **Key change**: Remove fake "Save Integrations" button. Channel cards are status displays, not edit forms.

### 4. License (read-only)
- **License card**: Tier badge (star icon, purple gradient border), usage bar (green/amber/red), count, remaining
- **Expiry row**: Calendar icon, date in mono, "X days remaining" in green
- **Feature matrix**: 2-column grid of features with check/X icons. Enterprise shows most enabled, Multi-Site and HA/DR disabled.
- **API**: `useLicenseStatus()` — existing hook
- **Key change**: Remove fake "Manage License" button entirely. The display is informational and honest.

### 5. Appearance (editable, localStorage)
- **Theme cards**: Dark and Light as selectable cards with miniature UI previews showing sidebar + content
- **Accent color**: Row of 8 color swatches (emerald, violet, blue, amber, red, pink, cyan, purple)
- **Display toggles**: Compact mode (reduce table padding), Show monospace data (mono font for IDs/hashes)
- **Persistence**: All appearance settings save to localStorage immediately (no API call needed)
- **Integration**: Uses existing `ThemeConfigurator` logic from `@patchiq/ui`

---

## Component Architecture

```
web/src/pages/settings/
├── SettingsLayout.tsx          # Wrapper that provides settings context
├── GeneralSettingsPage.tsx     # /settings/general (refactored from GeneralSettingsCard)
├── IdentitySettingsPage.tsx    # /settings/identity (refactored from IAMCard)
├── NotificationSettingsPage.tsx # /settings/notifications (refactored from IntegrationsCard)
├── LicenseSettingsPage.tsx     # /settings/license (refactored from LicenseCard)
├── AppearanceSettingsPage.tsx  # /settings/appearance (new, uses ThemeConfigurator)
└── components/
    └── SettingsSidebar.tsx     # Settings nav rendered inside AppSidebar

web/src/app/layout/
└── AppSidebar.tsx             # Modified: conditionally renders SettingsSidebar
```

### SettingsSidebar
- Rendered by `AppSidebar` when route matches `/settings/*`
- Contains: Back link, "Settings / Patch Manager" header, grouped nav items
- Nav items use `NavLink` with active state matching
- Identity & Access item shows green dot when IAM connection status is "connected"
- Uses `useIAMSettings()` to fetch connection status for the dot (lightweight, cached)

### SettingsLayout
- A thin wrapper (`<Outlet />` + shared padding/max-width)
- Provides the consistent content area styling (28px padding, 680px max-width)
- Used as a layout route in the router config

---

## Files Changed

| File | Change |
|------|--------|
| `web/src/app/routes.tsx` | Replace 3 settings routes with nested settings layout + 5 child routes |
| `web/src/app/layout/AppSidebar.tsx` | Add conditional: if `/settings/*`, render `SettingsSidebar` |
| `web/src/pages/settings/SettingsLayout.tsx` | **New**: Layout wrapper for settings pages |
| `web/src/pages/settings/SettingsSidebar.tsx` | **New**: Settings navigation component |
| `web/src/pages/settings/GeneralSettingsPage.tsx` | **Refactor** from `GeneralSettingsCard` → full page |
| `web/src/pages/settings/IdentitySettingsPage.tsx` | **Refactor** from `IAMCard` → full page, remove fake save |
| `web/src/pages/settings/NotificationSettingsPage.tsx` | **Refactor** from `IntegrationsCard` → full page, add toggles |
| `web/src/pages/settings/LicenseSettingsPage.tsx` | **Refactor** from `LicenseCard` → full page, remove fake button |
| `web/src/pages/settings/AppearanceSettingsPage.tsx` | **New**: Theme + accent + display toggles |
| `web/src/pages/settings/SettingsPage.tsx` | **Delete** (replaced by layout + pages) |
| `web/src/pages/settings/components/GeneralSettingsCard.tsx` | **Delete** (moved to page) |
| `web/src/pages/settings/components/IAMCard.tsx` | **Delete** (moved to page) |
| `web/src/pages/settings/components/IntegrationsCard.tsx` | **Delete** (moved to page) |
| `web/src/pages/settings/components/LicenseCard.tsx` | **Delete** (moved to page) |
| `web/src/pages/settings/iam.tsx` | **Delete** (superseded) |

---

## Design Tokens Used

All styling uses existing PatchIQ design tokens. No new tokens needed.

- Backgrounds: `--bg-page`, `--bg-card`, `--bg-inset`, `--bg-input`
- Borders: `--border`, `--border-divider`, `--border-faint`
- Text: `--text-emphasis`, `--text-primary`, `--text-secondary`, `--text-muted`, `--text-faint`
- Accent: `--accent`, `--accent-subtle`, `--accent-border`
- Signals: `--signal-healthy`, `--signal-critical`, `--signal-warning`, `--signal-info` + `-subtle` variants
- Typography: `--font-sans` (Geist), `--font-mono` (GeistMono)

---

## What's NOT Changing

- **Backend**: Zero backend changes. All existing API endpoints remain as-is.
- **API hooks**: All existing hooks (`useSettings`, `useIAMSettings`, `useTestIAMConnection`, `useChannelByType`, `useTestChannelByType`, `useLicenseStatus`) used unchanged.
- **Other pages**: No changes to any page outside settings.
- **AppLayout**: The layout component itself doesn't change — only `AppSidebar` gets conditional rendering.

---

## Cross-Platform Consistency

This sidebar-takeover pattern will be reused for:
- **web-hub/**: Settings sections = General, Identity, Feed Config, API & Webhooks
- **web-agent/**: Settings sections = General, Logging, Connection

Same component structure, same nav pattern, platform-specific sections. Hub and Agent implementation is out of scope for this spec but the pattern is designed for reuse.
