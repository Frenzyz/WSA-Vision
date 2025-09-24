import React, { useState, useEffect, useRef } from 'react';

const platformDefault = () => {
  const isMac = navigator.platform.toUpperCase().includes('MAC');
  return isMac ? 'Alt+C' : 'Alt+C';
};

export default function ShortcutCapture({ value, onChange }) {
  const [display, setDisplay] = useState(value || platformDefault());
  const ref = useRef(null);

  useEffect(() => { setDisplay(value || platformDefault()); }, [value]);

  function normalize(e) {
    const parts = [];
    const isMac = navigator.platform.toUpperCase().includes('MAC');
    if (e.ctrlKey) parts.push('Ctrl');
    if (e.metaKey) parts.push(isMac ? 'Cmd' : 'Meta');
    if (e.altKey) parts.push(isMac ? 'Alt' : 'Alt');
    if (e.shiftKey) parts.push('Shift');
    const key = e.key.length === 1 ? e.key.toUpperCase() : e.key[0].toUpperCase() + e.key.slice(1);
    if (!['Shift','Alt','Control','Meta'].includes(key)) parts.push(key);
    return parts.join('+');
  }

  return (
    <input
      ref={ref}
      type="text"
      value={display}
      onFocus={(e) => e.target.select()}
      onKeyDown={(e) => {
        e.preventDefault();
        const accel = normalize(e);
        setDisplay(accel);
        if (onChange) onChange(accel);
      }}
      style={{ width: '100%', padding: 8, borderRadius: 8, background: 'rgba(255,255,255,0.08)', border: '1px solid rgba(255,255,255,0.2)', color: 'var(--fg)' }}
    />
  );
}



