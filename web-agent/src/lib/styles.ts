import type React from 'react';

/**
 * Shared style constants used across multiple page components.
 * Import from here rather than duplicating in each file.
 */

/** Base card style — matches PM dashboard widget card pattern. */
export const CARD_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

/** Card content wrapper — use when card has a header + body split. */
export const CARD_HEADER_STYLE: React.CSSProperties = {
  padding: '16px 20px 0',
};

export const CARD_BODY_STYLE: React.CSSProperties = {
  padding: '12px 20px 16px',
  flex: 1,
};

/** For simple cards with uniform padding (no header/body split). */
export const CARD_PAD_STYLE: React.CSSProperties = {
  padding: '20px',
};

export const CARD_TITLE_STYLE: React.CSSProperties = {
  fontSize: 13,
  color: 'var(--text-muted)',
  fontWeight: 400,
  margin: 0,
};

export const SELECT_STYLE: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text-emphasis)',
  padding: '6px 10px',
  fontSize: '13px',
  cursor: 'pointer',
};
