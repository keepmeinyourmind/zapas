import { useId } from 'react';
export function Field({ label, className = 'col-12', children, ...props }) {
  const id = useId();
  const placeholder = props.placeholder ?? (props.type === 'number' ? 'Например, 1000' : !props.type || props.type === 'text' ? 'Введите значение' : undefined);
  return <div className={className}><label className="form-label" htmlFor={id}>{label}</label>
    {children ? <select id={id} className="form-select" {...props}>{children}</select> : <input id={id} className="form-control" placeholder={placeholder} {...props} />}
  </div>;
}
export function Checkbox({ label, hint, ...props }) {
  const id = useId();
  return <div className="col-12"><div className="form-check"><input className="form-check-input" id={id} type="checkbox" {...props} /><label className="form-check-label small" htmlFor={id}>{label}</label></div>{hint && <div className="form-text">{hint}</div>}</div>;
}
