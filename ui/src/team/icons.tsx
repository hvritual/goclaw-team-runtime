import { SVGProps } from 'react';

export type IconName =
  | 'overview'
  | 'chat'
  | 'spec'
  | 'memory'
  | 'approval'
  | 'development'
  | 'team'
  | 'progress'
  | 'harness'
  | 'search'
  | 'bell'
  | 'user'
  | 'logout'
  | 'refresh'
  | 'arrow'
  | 'menu'
  | 'close'
  | 'shield'
  | 'check'
  | 'warning'
  | 'branch'
  | 'runner'
  | 'document';

const paths: Record<IconName, JSX.Element> = {
  overview: <><path d="M3 11.5 12 4l9 7.5" /><path d="M5.5 10.5V21h13V10.5M9.5 21v-6h5v6" /></>,
  chat: <><path d="M21 12a8.5 8.5 0 0 1-9 8.5 9.6 9.6 0 0 1-4-.9L3 21l1.6-4.2A8.1 8.1 0 0 1 3.5 12 8.5 8.5 0 0 1 12 3.5 8.5 8.5 0 0 1 21 12Z" /></>,
  spec: <><path d="M6 3h9l4 4v14H6Z" /><path d="M14.5 3v5H19M9 12h6M9 16h6" /></>,
  memory: <><path d="M7 6.5A3.5 3.5 0 0 1 10.5 3h3A3.5 3.5 0 0 1 17 6.5v11a3.5 3.5 0 0 1-3.5 3.5h-3A3.5 3.5 0 0 1 7 17.5Z" /><path d="M7 8H4.5M7 12H4M7 16H4.5M17 8h2.5M17 12h3M17 16h2.5M10 9.5h4M10 13h4" /></>,
  approval: <><path d="M7 3h10v4H7Z" /><path d="M5 5h14v16H5Z" /><path d="m8.5 14 2.2 2.2 4.8-5" /></>,
  development: <><path d="m8.5 8.5-5 3.5 5 3.5M15.5 8.5l5 3.5-5 3.5M13.5 5l-3 14" /></>,
  team: <><path d="M16 20v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 10a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 20v-2a4 4 0 0 0-3-3.8M16 2.2a4 4 0 0 1 0 7.6" /></>,
  progress: <><path d="M4 20V10M10 20V4M16 20v-7M22 20V7" /><path d="M2 20h22" /></>,
  harness: <><path d="M14.7 6.3a4.5 4.5 0 0 0-5.8 5.8L3 18l3 3 5.9-5.9a4.5 4.5 0 0 0 5.8-5.8l-2.8 2.8-3-3Z" /></>,
  search: <><circle cx="11" cy="11" r="7" /><path d="m20 20-4-4" /></>,
  bell: <><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></>,
  user: <><circle cx="12" cy="7" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></>,
  logout: <><path d="M10 4H4v16h6M14 8l4 4-4 4M8 12h10" /></>,
  refresh: <><path d="M20 6v5h-5M4 18v-5h5" /><path d="M18.5 10A7 7 0 0 0 6 6.5L4 11M5.5 14A7 7 0 0 0 18 17.5l2-4.5" /></>,
  arrow: <><path d="M5 12h14M14 7l5 5-5 5" /></>,
  menu: <><path d="M4 7h16M4 12h16M4 17h16" /></>,
  close: <><path d="m6 6 12 12M18 6 6 18" /></>,
  shield: <><path d="M12 3 4.5 6v5.5c0 4.8 3.2 8 7.5 9.5 4.3-1.5 7.5-4.7 7.5-9.5V6Z" /><path d="m8.5 12 2.2 2.2 4.8-5" /></>,
  check: <><path d="m5 12 4 4L19 6" /></>,
  warning: <><path d="M12 3 2.5 20h19Z" /><path d="M12 9v5M12 17.5v.5" /></>,
  branch: <><circle cx="6" cy="5" r="2" /><circle cx="18" cy="7" r="2" /><circle cx="6" cy="19" r="2" /><path d="M6 7v10M8 7h4a6 6 0 0 1 6 6v-4" /></>,
  runner: <><rect x="3" y="5" width="18" height="14" rx="2" /><path d="M7 9h4M7 13h7M17 13h.1" /></>,
  document: <><path d="M6 3h9l4 4v14H6Z" /><path d="M14.5 3v5H19M9 12h6M9 16h6" /></>,
};

export function Icon({ name, ...props }: SVGProps<SVGSVGElement> & { name: IconName }) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="1.8"
      viewBox="0 0 24 24"
      {...props}
    >
      {paths[name]}
    </svg>
  );
}
