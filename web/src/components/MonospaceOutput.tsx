interface MonospaceOutputProps {
  content: string | null | undefined;
  className?: string;
}

export const MonospaceOutput = ({ content }: MonospaceOutputProps) => {
  if (!content) {
    return (
      <p
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 13,
          color: 'var(--text-muted)',
          margin: 0,
        }}
      >
        No output
      </p>
    );
  }
  return (
    <pre
      style={{
        maxHeight: 300,
        overflow: 'auto',
        borderRadius: 8,
        background: 'var(--bg-page)',
        border: '1px solid var(--border)',
        padding: '14px 16px',
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--text-primary)',
        margin: 0,
        lineHeight: 1.6,
      }}
    >
      {content}
    </pre>
  );
};
