import { useCallback, useEffect, useRef, useState } from 'react';
import { X } from 'lucide-react';
import { cn } from '@patchiq/ui';

interface SlidePanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  /** Default width in pixels */
  defaultWidth?: number;
  /** Minimum width in pixels */
  minWidth?: number;
  /** Maximum width as fraction of viewport (0-1) */
  maxWidthFraction?: number;
}

export function SlidePanel({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  defaultWidth = 480,
  minWidth = 360,
  maxWidthFraction = 0.85,
}: SlidePanelProps) {
  const [width, setWidth] = useState(defaultWidth);
  const [isResizing, setIsResizing] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  // Animate in on open
  useEffect(() => {
    if (open) {
      // Force a reflow before setting visible for animation
      requestAnimationFrame(() => setVisible(true));
    } else {
      setVisible(false);
    }
  }, [open]);

  // Body scroll lock
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [open]);

  // Track if panel just opened to avoid re-focusing on every render
  const justOpened = useRef(false);
  useEffect(() => {
    if (open) justOpened.current = true;
    else justOpened.current = false;
  }, [open]);

  // Focus panel on open + focus trap + Escape
  useEffect(() => {
    if (!open) return;
    // Only focus panel on initial open, not on re-renders
    if (justOpened.current) {
      panelRef.current?.focus();
      justOpened.current = false;
    }
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onOpenChange(false);
        return;
      }
      if (e.key !== 'Tab') return;
      const panel = panelRef.current;
      if (!panel) return;
      const focusable = Array.from(
        panel.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])',
        ),
      ).filter((el) => !el.closest('[aria-hidden="true"]'));
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [open, onOpenChange]);

  // Resize logic (drag left edge like VS Code terminal)
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);
      const startX = e.clientX;
      const startWidth = width;
      const maxWidth = window.innerWidth * maxWidthFraction;

      const handleMouseMove = (e: MouseEvent) => {
        const delta = startX - e.clientX;
        const newWidth = Math.min(Math.max(startWidth + delta, minWidth), maxWidth);
        setWidth(newWidth);
      };

      const handleMouseUp = () => {
        setIsResizing(false);
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };

      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    },
    [width, minWidth, maxWidthFraction],
  );

  if (!open) return null;

  return (
    <>
      {/* Overlay */}
      <div
        className={cn(
          'fixed inset-0 z-50 bg-black/50 transition-opacity duration-300',
          visible ? 'opacity-100' : 'opacity-0',
        )}
        onClick={() => onOpenChange(false)}
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="slide-panel-title"
        tabIndex={-1}
        className={cn(
          'fixed inset-y-0 right-0 z-50 flex flex-col bg-background border-l border-border shadow-xl transition-transform duration-300 ease-in-out',
          visible ? 'translate-x-0' : 'translate-x-full',
          isResizing && 'transition-none',
        )}
        style={{ width }}
      >
        {/* Resize handle — left edge */}
        <div
          onMouseDown={handleMouseDown}
          className={cn(
            'absolute inset-y-0 left-0 w-1 cursor-col-resize hover:bg-primary/40 transition-colors z-10',
            isResizing && 'bg-primary/40',
          )}
        />

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border shrink-0">
          <div>
            <h2
              id="slide-panel-title"
              className="text-lg font-semibold"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {title}
            </h2>
            {description && <p className="text-sm text-muted-foreground mt-0.5">{description}</p>}
          </div>
          <button
            onClick={() => onOpenChange(false)}
            className="rounded-sm p-1 opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-none"
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Close</span>
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4">{children}</div>

        {/* Footer */}
        {footer && <div className="shrink-0 border-t border-border px-6 py-4">{footer}</div>}
      </div>
    </>
  );
}
