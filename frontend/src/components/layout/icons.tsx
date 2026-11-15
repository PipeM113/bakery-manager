// One set of line icons for both navigations: same 24 px grid, 1.5 stroke, the color of the text.
// (Text symbols were drawn by whatever font the device had, so they differed in size and two
// sections shared one.)
import type { ReactNode } from "react";

const paths: Record<string, ReactNode> = {
  dashboard: <><rect x="3.5" y="3.5" width="7" height="7" /><rect x="13.5" y="3.5" width="7" height="7" /><rect x="3.5" y="13.5" width="7" height="7" /><rect x="13.5" y="13.5" width="7" height="7" /></>,
  ingredients: <><path d="M9 3.5h6M10 3.5v5.2L5.2 17a2.5 2.5 0 0 0 2.2 3.5h9.2a2.5 2.5 0 0 0 2.2-3.5L14 8.7V3.5" /><path d="M7.5 14h9" /></>,
  recipes: <><path d="M6 10.5a3.5 3.5 0 0 1 2-6.3 4 4 0 0 1 8 0 3.5 3.5 0 0 1 2 6.3V19a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1z" /><path d="M6 16h12" /></>,
  costs: <><circle cx="12" cy="12" r="8.5" /><path d="M14.8 9.2c-.5-.8-1.5-1.2-2.8-1.2-1.7 0-2.8.8-2.8 2s1 1.7 2.8 2 2.8.8 2.8 2-1.1 2-2.8 2c-1.3 0-2.4-.4-2.9-1.3M12 6.5V8m0 8v1.5" /></>,
  fixed: <><rect x="3.5" y="5" width="17" height="15" /><path d="M3.5 10h17M8 3v4M16 3v4" /></>,
  sales: <><path d="M5 7h14l-1.2 11.2a1 1 0 0 1-1 .8H7.2a1 1 0 0 1-1-.8z" /><path d="M9 7V6a3 3 0 0 1 6 0v1" /></>,
  expenses: <><rect x="3.5" y="6" width="17" height="12" /><circle cx="12" cy="12" r="2.5" /><path d="M7 9.5v.01M17 14.5v.01" /></>,
  profit: <><path d="M4 19.5h16" /><path d="M6 16l4-4.5 3.5 3L19 7.5" /><path d="M15 7.5h4V11.5" /></>,
  business: <><path d="M4 20.5V9.5L12 4l8 5.5v11" /><path d="M9.5 20.5v-6h5v6" /></>,
  account: <><circle cx="12" cy="8.5" r="3.5" /><path d="M5 20c.6-3.6 3.5-5.5 7-5.5s6.4 1.9 7 5.5" /></>,
  logout: <><path d="M10 4.5H5.5a1 1 0 0 0-1 1v13a1 1 0 0 0 1 1H10" /><path d="M15 8l4 4-4 4M19 12H9.5" /></>,
  more: <><circle cx="6" cy="12" r="1" /><circle cx="12" cy="12" r="1" /><circle cx="18" cy="12" r="1" /></>,
};

export type IconName = keyof typeof paths;

export function Icon({ name, className = "w-5 h-5" }: { name: IconName; className?: string }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"
      strokeLinecap="round" strokeLinejoin="round" className={`shrink-0 ${className}`}>
      {paths[name]}
    </svg>
  );
}
