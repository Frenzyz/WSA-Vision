import React, { useState, useEffect } from 'react';
import { cn } from '../lib/utils';
import { ArrowLeft, ArrowRight, Check, ChevronsDownUp, ChevronsUpDown, Eye } from 'lucide-react';

const CompleteInput = ({
  label,
  type,
  placeholder,
  options,
  hideLabel,
  disabled,
  expanded,
  value,
  onChange
}) => {
  const id = `field-${label.toLowerCase().replace(/\s+/g, '-')}`;

  if (type === 'text' || type === 'textarea') {
    const Component = type === 'textarea' ? 'textarea' : 'input';
    return (
      <div className="flex w-full flex-col gap-1">
        <label
          htmlFor={id}
          className={cn('text-sm font-medium text-white', (!expanded || hideLabel) && 'opacity-0 select-none')}
        >
          {label}
        </label>
        <div>
          <Component
            id={id}
            type={type === 'text' ? 'text' : undefined}
            placeholder={placeholder}
            className="w-full bg-black/60 text-white border border-green-500/30 rounded-lg px-3 py-2 focus:border-green-500 focus:outline-none backdrop-blur-sm"
            autoComplete="off"
            disabled={disabled}
            value={value || ''}
            onChange={(e) => onChange?.(e.target.value)}
            rows={type === 'textarea' ? 3 : undefined}
            style={type === 'textarea' ? { resize: 'none' } : undefined}
          />
        </div>
      </div>
    );
  } else if (type === 'select') {
    return (
      <div className="flex w-full flex-col gap-1">
        <label
          htmlFor={id}
          className={cn('text-sm font-medium text-white', (!expanded || hideLabel) && 'opacity-0 select-none')}
        >
          {label}
        </label>
        <div>
          <select
            id={id}
            className="w-full bg-black/60 text-white border border-green-500/30 rounded-lg px-3 py-2 focus:border-green-500 focus:outline-none backdrop-blur-sm"
            disabled={disabled}
            value={value || ''}
            onChange={(e) => onChange?.(e.target.value)}
          >
            <option value="">{placeholder ?? 'Select…'}</option>
            {options?.map((opt) => (
              <option key={opt.value} value={opt.value} className="bg-black text-white">
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>
    );
  } else if (type === 'checkbox') {
    return (
      <div className="flex w-full flex-col gap-1">
        <label
          htmlFor={id}
          className={cn('text-sm font-medium text-white', (!expanded || hideLabel) && 'opacity-0 select-none')}
        >
          {label}
        </label>
        <div className="flex items-center gap-3 p-3 bg-black/40 border border-green-500/30 rounded-lg backdrop-blur-sm">
          <input
            type="checkbox"
            id={id}
            checked={value || false}
            onChange={(e) => onChange?.(e.target.checked)}
            disabled={disabled}
            className="w-4 h-4 text-green-500 bg-transparent border-green-500/50 rounded focus:ring-green-500"
          />
          <Eye className="w-5 h-5 text-green-500" />
          <div>
            <div className="text-white font-medium">Vision Mode</div>
            <div className="text-white/60 text-sm">Enable visual analysis and screen interaction</div>
          </div>
        </div>
      </div>
    );
  }

  return null;
};

export const Field = {
  TEXT: 'text',
  TEXTAREA: 'textarea', 
  SELECT: 'select',
  CHECKBOX: 'checkbox'
};

const StackedInputForm = ({ fields, title = "Configure Task", onFieldChange }) => {
  const [currentFieldIndex, setCurrentFieldIndex] = useState(0);
  const [showAll, setShowAll] = useState(false);
  const [fieldValues, setFieldValues] = useState({});

  useEffect(() => {
    const handleKeyDown = (e) => {
      const activeTag = (document.activeElement?.tagName || '').toLowerCase();
      if (activeTag === 'input' || activeTag === 'textarea' || activeTag === 'select') return;
      
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        setCurrentFieldIndex((i) => Math.max(0, i - 1));
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        setCurrentFieldIndex((i) => Math.min(fields.length - 1, i + 1));
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [fields.length]);

  const handleFieldChange = (fieldLabel, value) => {
    const newValues = { ...fieldValues, [fieldLabel]: value };
    setFieldValues(newValues);
    onFieldChange?.(newValues);
  };

  const layoutTransition = showAll
    ? {
        type: 'spring',
        stiffness: 700,
        damping: 50
      }
    : {
        type: 'spring',
        stiffness: 300,
        damping: 30
      };

  return (
    <div className="flex flex-col w-full max-w-md">
      <div className="z-10">
        <div className={cn('flex items-center justify-between w-full mb-4')}>
          <label className="text-left text-lg font-medium text-white">{title}</label>
          <button
            onClick={() => setShowAll(!showAll)}
            className="text-xs text-white/70 hover:text-white flex items-center gap-1 px-2 py-1 rounded-md hover:bg-white/10 transition-colors"
          >
            <span className="flex items-center gap-1">
              {showAll ? (
                <>Collapse <ChevronsDownUp size={14} /></>
              ) : (
                <>Expand <ChevronsUpDown size={14} /></>
              )}
            </span>
          </button>
        </div>
      </div>

      <div className={cn(
        'relative w-full flex flex-col flex-nowrap items-start justify-start gap-2 mt-0 mb-4',
        !showAll && 'min-h-[80px]'
      )}>
        {fields.map((field, index) => {
          if (index > currentFieldIndex || (!showAll && currentFieldIndex - index >= 5))
            return null;

          return (
            <div
              key={field.label + index}
              style={{ 
                zIndex: index + 1,
                transform: !showAll ? `translateY(${-(currentFieldIndex - index) * 6}px)` : 'none',
                opacity: !showAll ? [1, 0.75, 0.5, 0.25, 0][currentFieldIndex - index] : 1,
              }}
              className={cn(!showAll && 'absolute w-full', 'shrink-0 transition-all duration-300')}
            >
              <CompleteInput
                label={field.label}
                type={field.type}
                placeholder={field.placeholder}
                options={field.options}
                hideLabel={showAll ? false : index !== currentFieldIndex}
                expanded={showAll}
                disabled={field.disabled}
                value={fieldValues[field.label]}
                onChange={(value) => handleFieldChange(field.label, value)}
              />
            </div>
          );
        })}
      </div>

      <div className={cn(
        'flex w-full items-center justify-between z-10',
        showAll
          ? 'mt-0'
          : ['textarea', 'slider'].includes(fields[currentFieldIndex]?.type)
          ? 'mt-6'
          : 'mt-0'
      )}>
        {currentFieldIndex > 0 ? (
          <button
            onClick={() => setCurrentFieldIndex((i) => Math.max(0, i - 1))}
            disabled={currentFieldIndex === 0}
            className="rounded-full select-none px-3 py-2 bg-white/10 hover:bg-white/20 text-white disabled:opacity-50 flex items-center gap-2 transition-colors"
          >
            <ArrowLeft size={14} />
            Back
          </button>
        ) : (
          <div></div>
        )}

        <button
          onClick={() => setCurrentFieldIndex((i) => Math.min(i + 1, fields.length - 1))}
          disabled={currentFieldIndex === fields.length - 1}
          className="rounded-full select-none px-3 py-2 bg-green-500 hover:bg-green-600 text-black disabled:opacity-50 flex items-center gap-2 font-medium transition-colors"
        >
          <span className="flex items-center gap-1 text-sm">
            {currentFieldIndex === fields.length - 1 ? (
              <>Done <Check size={14} /></>
            ) : (
              <>Next <ArrowRight size={14} /></>
            )}
          </span>
        </button>
      </div>
    </div>
  );
};

export default StackedInputForm;