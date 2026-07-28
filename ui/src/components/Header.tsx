import { useWebSocket } from '../hooks/useWebSocket';
import { useHealth } from '../hooks/useApi';
import StatusIndicator from './StatusIndicator';

interface HeaderProps {
  onToggleSidebar: () => void;
}

const Header: React.FC<HeaderProps> = ({ onToggleSidebar }) => {
  const { status: wsStatus } = useWebSocket(true);
  const { data: health } = useHealth();

  return (
    <header className="flex items-center justify-between h-16 px-6 bg-white border-b border-gray-200">
      <div className="flex items-center gap-4">
        <button
          onClick={onToggleSidebar}
          className="p-2 rounded-md hover:bg-gray-100 lg:hidden"
        >
          <svg
            className="w-5 h-5 text-gray-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        </button>
        <h2 className="text-lg font-semibold text-gray-900">Control Panel</h2>
      </div>

      <div className="flex items-center gap-6">
        {health && (
          <div className="text-sm text-gray-500">
            Last update: {new Date((health.timestamp || health.time || 0) * 1000).toLocaleTimeString()}
          </div>
        )}
        <StatusIndicator status={wsStatus} label="WebSocket" />
      </div>
    </header>
  );
};

export default Header;