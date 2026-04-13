import type { CSSProperties } from 'react';

// Monochrome tier badges (Design Rule #2: No colored badge backgrounds)
export function tierBadgeStyle(_tier: string): CSSProperties {
  return {
    background: 'var(--bg-card-hover, #1a1a1a)',
    color: 'var(--text-secondary, #a1a1a1)',
    borderColor: 'var(--border-strong, #333333)',
  };
}

export function getTierFeatures(tier: string): string[] {
  switch (tier) {
    case 'community':
      return ['Basic patching', 'Up to 50 endpoints', 'Community support'];
    case 'professional':
      return ['Advanced patching', 'Workflow builder', 'Email support', 'Compliance reports'];
    case 'enterprise':
      return [
        'Everything in Professional',
        'SSO/SAML',
        'Multi-site',
        'HA/DR',
        'Custom RBAC',
        '24/7 support',
      ];
    case 'msp':
      return ['Everything in Enterprise', 'Multi-tenant', 'White-label', 'Partner support'];
    default:
      return [];
  }
}
