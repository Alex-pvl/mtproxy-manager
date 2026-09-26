import { useEffect, useRef, useState } from 'react';

export interface PaymentMethod {
  id: string;
  label: string;
  icon: string; // image path (public/)
  disabled?: boolean;
  badge?: string;
}

interface Props {
  methods: PaymentMethod[];
  value: string;
  onChange: (id: string) => void;
  placeholder: string;
}

function ChevronDown({ open }: { open: boolean }) {
  return (
    <svg
      className="spend-pill__chevron"
      style={{ transform: open ? 'rotate(180deg)' : undefined }}
      width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
    >
      <polyline points="6 9 12 15 18 9" />
    </svg>
  );
}

export default function PaymentMethodPicker({ methods, value, onChange, placeholder }: Props) {
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const selected = methods.find((m) => m.id === value);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!wrapperRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [open]);

  return (
    <div ref={wrapperRef}>
      <button
        type="button"
        className="spend-pill"
        aria-expanded={open ? 'true' : 'false'}
        onClick={() => setOpen((v) => !v)}
      >
        {selected ? (
          <>
            <img src={selected.icon} alt="" className="w-7 h-7 rounded-full object-cover flex-shrink-0" />
            <span className="spend-pill__label">{selected.label}</span>
          </>
        ) : (
          <>
            <span className="spend-pill__icon">$</span>
            <span className="spend-pill__label spend-pill__placeholder">{placeholder}</span>
          </>
        )}
        <ChevronDown open={open} />
      </button>

      {open && (
        <div className="spend-sheet">
          {methods.map((m) => (
            <button
              key={m.id}
              type="button"
              disabled={m.disabled}
              onClick={() => {
                if (!m.disabled) {
                  onChange(m.id);
                  setOpen(false);
                }
              }}
              className={`spend-sheet__item ${m.id === value ? 'spend-sheet__item--active' : ''} ${m.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <img src={m.icon} alt="" className="w-7 h-7 rounded-full object-cover flex-shrink-0" />
              <span className="flex-1">{m.label}</span>
              {m.badge && (
                <span className="text-[10px] uppercase tracking-wider bg-[#07acff1a] text-[#3aa8fc] px-2 py-0.5 rounded-full">
                  {m.badge}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
