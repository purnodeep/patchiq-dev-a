# PatchIQ — License Management & Tiers

> License system architecture, tier definitions, and feature gating.

---

## 1. Architecture

```
┌──────────────────────────────────────┐
│         PatchIQ Hub Manager          │
│  ┌─────────────────────────────────┐ │
│  │     License Generation Service  │ │
│  │  - RSA-2048 key pair            │ │
│  │  - JSON payload + signature     │ │
│  │  - Feature flags per tier       │ │
│  └──────────────┬──────────────────┘ │
└─────────────────┼────────────────────┘
                  │ License file (.piq-license)
┌─────────────────┼────────────────────┐
│       Patch Manager Instance         │
│  ┌──────────────┴──────────────────┐ │
│  │    License Validator Service    │ │
│  │  - Embedded public key          │ │
│  │  - Verify signature             │ │
│  │  - Check expiry + features      │ │
│  │  - Enforce limits               │ │
│  │  - Grace period (30 days)       │ │
│  └─────────────────────────────────┘ │
└──────────────────────────────────────┘
```

---

## 2. License File Structure

```json
{
  "license_id": "LIC-2025-00142",
  "customer": {
    "id": "CUST-00042",
    "name": "Acme Corp",
    "contact_email": "it@acme.com"
  },
  "tier": "enterprise",
  "features": {
    "max_endpoints": 5000,
    "os_support": ["windows", "linux", "macos"],
    "visual_workflow_builder": true,
    "ai_assistant": true,
    "sso_saml": true,
    "sso_oidc": true,
    "approval_workflows": true,
    "compliance_reports": ["hipaa", "soc2", "iso27001"],
    "multi_site": true,
    "ha_dr": true,
    "custom_rbac": true,
    "api_access": true,
    "third_party_patching": true,
    "vulnerability_integration": true
  },
  "issued_at": "2025-03-01T00:00:00Z",
  "expires_at": "2026-03-01T00:00:00Z",
  "grace_period_days": 30,
  "hardware_fingerprint": null,
  "signature": "<RSA-2048 signature of above fields>"
}
```

---

## 3. Feature Gating Implementation

```go
// Every feature-gated API endpoint checks:
func (m *LicenseMiddleware) RequireFeature(feature string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !m.license.HasFeature(feature) {
            c.JSON(403, gin.H{
                "error": "feature_not_licensed",
                "feature": feature,
                "upgrade_url": "https://patchiq.io/pricing",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 4. License Tiers

| Feature | Community (Free) | Professional | Enterprise | MSP |
|---------|-----------------|-------------|-----------|-----|
| Max Endpoints | 25 | 1,000 | 10,000 | Unlimited |
| OS Support | Linux only | All | All | All |
| Visual Workflow Builder | Basic (3 nodes) | Full | Full | Full |
| AI Assistant | No | Basic (read-only) | Full | Full |
| SSO (SAML/OIDC) | No | No | Yes | Yes |
| Approval Workflows | No | Basic | Full | Full |
| Compliance Reports | Basic (CVE list) | SOC2 | All frameworks | All |
| Multi-site Deployment | No | No | Yes | Yes |
| HA/DR | No | No | Yes | Yes |
| Custom RBAC | 4 preset roles | 8 preset roles | Full custom | Full custom + tenant |
| API Access | Read-only | Full | Full | Full |
| 3rd Party Patching | No | Top 25 apps | Unlimited | Unlimited |
| Support | Community | Email (48h) | Priority (4h) | Dedicated |
| **Price** | Free | $3/endpoint/mo | $6/endpoint/mo | Custom |

---

## 5. Offline / Air-Gapped License Validation

For air-gapped deployments:
- License file contains all feature information in the signed payload
- Public key is embedded in the Patch Manager binary at build time
- No online validation required — signature verification is purely cryptographic
- Expiry checked against system clock (with clock-drift tolerance of 48h)
- Grace period: 30 days after expiry, features continue working with daily admin warning
- After grace: reverts to Community tier feature set (not a full lockout)

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| License generation (Hub) | `internal/hub/license/` |
| License validation (PM) | `internal/server/license/` |
| License CLI tool | `cmd/tools/licensegen/` |
