import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Activity,
  BarChart3,
  Zap,
  MessageSquare,
  Bot,
  Settings,
  ChevronLeft,
  ChevronRight,
  FlaskConical,
  Trophy,
  Network,
} from 'lucide-react';
import { cn } from '../lib/utils';

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

const navItems = [
  { to: '/', icon: LayoutDashboard, label: '概览', sublabel: 'Overview' },
  { to: '/traces', icon: Activity, label: '追踪', sublabel: 'Traces' },
  { to: '/metrics', icon: BarChart3, label: '指标', sublabel: 'Metrics' },
  { to: '/sessions', icon: MessageSquare, label: '会话', sublabel: 'Sessions' },
  { to: '/agents', icon: Bot, label: '智能体', sublabel: 'Agents' },
  { to: '/topology', icon: Network, label: '拓扑', sublabel: 'Topology' },
  { to: '/events', icon: Zap, label: '事件', sublabel: 'Events' },
  { to: '/evaluations', icon: FlaskConical, label: '评估', sublabel: 'Evaluations' },
  { to: '/benchmarks', icon: Trophy, label: '基准测试', sublabel: 'Benchmarks' },
  { to: '/settings', icon: Settings, label: '设置', sublabel: 'Settings' },
];

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  return (
    <aside
      className={cn(
        'flex flex-col h-full bg-[var(--color-surface)] border-r border-[var(--color-border)] transition-all duration-300',
        collapsed ? 'w-16' : 'w-56'
      )}
    >
      {/* Logo */}
      <div className="flex items-center h-14 px-4 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[var(--color-primary)] to-[var(--color-secondary)] flex items-center justify-center">
            <span className="text-white font-bold text-sm">A</span>
          </div>
          {!collapsed && (
            <span className="font-semibold text-[var(--color-text-primary)]">
              Aster Studio
            </span>
          )}
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-4">
        <ul className="space-y-1 px-2">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 px-3 py-2 rounded-lg transition-colors',
                    'hover:bg-[var(--color-surface-elevated)]',
                    isActive
                      ? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
                      : 'text-[var(--color-text-secondary)]'
                  )
                }
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                {!collapsed && (
                  <div className="flex flex-col">
                    <span className="text-sm font-medium">{item.label}</span>
                    <span className="text-xs text-[var(--color-text-muted)]">{item.sublabel}</span>
                  </div>
                )}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      {/* Collapse Toggle */}
      <div className="p-2 border-t border-[var(--color-border)]">
        <button
          onClick={onToggle}
          className="w-full flex items-center justify-center p-2 rounded-lg hover:bg-[var(--color-surface-elevated)] text-[var(--color-text-muted)] transition-colors"
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? (
            <ChevronRight className="w-5 h-5" />
          ) : (
            <ChevronLeft className="w-5 h-5" />
          )}
        </button>
      </div>
    </aside>
  );
}
