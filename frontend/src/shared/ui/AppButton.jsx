export default function AppButton({ children, pending = false, disabled, className = '', type = 'button', ...props }) {
  return <button type={type} className={`reference-button ${className}`} disabled={disabled || pending} aria-busy={pending} {...props}>
    {pending ? 'Подождите…' : children}
  </button>;
}
