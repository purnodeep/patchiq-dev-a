export interface FeedIconConfig {
  emoji: string;
  bgClass: string;
}

export function getFeedIconConfig(name: string): FeedIconConfig {
  const n = name.toUpperCase();
  if (n.includes('NVD'))
    return { emoji: '🛡', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  if (n.includes('KEV'))
    return { emoji: '🔴', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  if (n.includes('MSRC'))
    return { emoji: '🟣', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  if (n.includes('USN') || n.includes('UBUNTU'))
    return { emoji: '🟠', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  if (n.includes('OVAL') || n.includes('RHEL') || n.includes('REDHAT'))
    return { emoji: '🔴', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  if (n.includes('APPLE') || n.includes('MACOS'))
    return { emoji: '🍎', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
  return { emoji: '⚙', bgClass: 'bg-neutral-800 border border-[var(--border)]' };
}
