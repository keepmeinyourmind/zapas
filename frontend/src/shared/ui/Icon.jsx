const paths = {
  home: ['m3.5 10 8.5-7 8.5 7', 'M5.5 9v11h13V9M9.5 20v-6h5v6'],
  savings: ['M4 8.5h16v11H4z', 'M4 8.5 6 5h12l2 3.5M8 12h8', 'M12 12v4'],
  transactions: ['M5 6h14M5 12h14M5 18h14', 'm8 4-3 2 3 2M16 10l3 2-3 2'],
  recurring: ['M8 3v4M16 3v4M4 10h16M8 14h3M8 17h6'],
  decisions: ['M12 3 20 7v5c0 4.5-3 7.5-8 9-5-1.5-8-4.5-8-9V7z', 'm8.5 12 2.2 2.2 4.8-4.8'],
  onboarding: ['M7 3h8l4 4v14H7z', 'M15 3v5h5M10 13h6M10 17h6'],
  plus: ['M12 5v14M5 12h14'],
  withdraw: ['M12 4v13M7 12l5 5 5-5', 'M5 20h14'],
  back: ['m15 18-6-6 6-6'],
  info: ['M12 11v6M12 7h.01'],
};
export default function Icon({ name }) {
  return <svg viewBox="0 0 24 24" aria-hidden="true">
    {name === 'info' && <circle cx="12" cy="12" r="9" />}
    {name === 'recurring' && <rect x="4" y="5" width="16" height="15" rx="2" />}
    {(paths[name] || paths.info).map((path) => <path key={path} d={path} />)}
  </svg>;
}
