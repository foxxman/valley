import { useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

type TooltipProps = {
  label: string;
  children: ReactNode;
};

export function Tooltip({ label, children }: TooltipProps) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [open, setOpen] = useState(false);
  const id = useId();

  useEffect(() => {
    if (!open) {
      return;
    }
    const close = () => setOpen(false);
    window.addEventListener('scroll', close, true);
    window.addEventListener('resize', close);
    return () => {
      window.removeEventListener('scroll', close, true);
      window.removeEventListener('resize', close);
    };
  }, [open]);

  return (
    <span
      className="tooltip"
      ref={setAnchor}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
      aria-describedby={open ? id : undefined}
    >
      {children}
      {open && anchor ? <TooltipBubble id={id} label={label} anchor={anchor} /> : null}
    </span>
  );
}

function TooltipBubble({ id, label, anchor }: { id: string; label: string; anchor: HTMLElement }) {
  const bubbleRef = useRef<HTMLSpanElement>(null);
  const [style, setStyle] = useState<CSSProperties>({ visibility: 'hidden' });

  useLayoutEffect(() => {
    const bubble = bubbleRef.current;
    if (!bubble) {
      return;
    }
    const rect = anchor.getBoundingClientRect();
    const width = bubble.offsetWidth;
    const height = bubble.offsetHeight;
    const margin = 8;
    const gap = 6;
    const left = Math.min(
      window.innerWidth - margin - width,
      Math.max(margin, rect.left + rect.width / 2 - width / 2),
    );
    const above = rect.top - gap - height;
    const top = above >= margin ? above : rect.bottom + gap;
    setStyle({ left, top, visibility: 'visible' });
  }, [anchor, label]);

  return createPortal(
    <span ref={bubbleRef} id={id} role="tooltip" className="tooltip-bubble" style={style}>
      {label}
    </span>,
    document.body,
  );
}
