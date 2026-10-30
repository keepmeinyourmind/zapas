import { useId } from 'react';
export function Field({ label, className = 'col-12', children, ...props }) {
  const id = useId();
  return <div className={className}><label className="form-label" htmlFor={id}>{label}</label>
    {children ? <select id={id} className="form-select" {...props}>{children}</select> : <input id={id} className="form-control" {...props} />}
  </div>;
}
export function Checkbox({ label, hint, ...props }) {
  const id = useId();
  return <div className="col-12"><div className="form-check"><input className="form-check-input" id={id} type="checkbox" {...props} /><label className="form-check-label small" htmlFor={id}>{label}</label></div>{hint && <div className="form-text">{hint}</div>}</div>;
}
