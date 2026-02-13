import { useState } from 'react';
import { 
  Search, 
  Bell, 
  AlertTriangle, 
  Command,
  Check,
  Info,
  AlertCircle
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { alerts } from '@/data/mockData';
import type { Alert } from '@/types';

interface TopBarProps {
  pageTitle: string;
}

export function TopBar({ pageTitle }: TopBarProps) {
  const [searchFocused, setSearchFocused] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const [showAlerts, setShowAlerts] = useState(false);
  const [notificationList, setNotificationList] = useState<Alert[]>(alerts);

  const unreadCount = notificationList.filter(n => !n.isRead).length;

  const markAsRead = (id: string) => {
    setNotificationList(prev => prev.map(n => n.id === id ? { ...n, isRead: true } : n));
  };

  const getAlertIcon = (type: Alert['type']) => {
    switch (type) {
      case 'error': return <AlertCircle className="w-4 h-4 text-[#FF6B6B]" />;
      case 'warning': return <AlertTriangle className="w-4 h-4 text-[#FFD166]" />;
      case 'success': return <Check className="w-4 h-4 text-[#2DD4A8]" />;
      case 'info': return <Info className="w-4 h-4 text-[#4CC9F0]" />;
    }
  };

  return (
    <header className="h-16 bg-[#0E1320] border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between px-6 sticky top-0 z-40">
      {/* Page Title */}
      <div>
        <h2 className="text-xl font-semibold text-[#F5F7FF]">{pageTitle}</h2>
      </div>

      {/* Right Section */}
      <div className="flex items-center gap-4">
        {/* Search */}
        <div className={cn(
          "relative transition-all duration-300",
          searchFocused ? "w-80" : "w-64"
        )}>
          <Search className={cn(
            "absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 transition-colors",
            searchFocused ? "text-[#7B61FF]" : "text-[#A7B1C8]"
          )} />
          <input
            type="text"
            placeholder="Search users, reports, settings..."
            className={cn(
              "w-full pl-10 pr-10 py-2 rounded-lg text-sm transition-all duration-200",
              "bg-[rgba(14,19,32,0.8)] border text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)]",
              searchFocused 
                ? "border-[#7B61FF] shadow-[0_0_0_3px_rgba(123,97,255,0.15)]" 
                : "border-[rgba(123,97,255,0.18)]"
            )}
            onFocus={() => setSearchFocused(true)}
            onBlur={() => setSearchFocused(false)}
          />
          <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1 text-[#A7B1C8]">
            <Command className="w-3 h-3" />
            <span className="text-xs">K</span>
          </div>
        </div>

        {/* Alerts Button */}
        <div className="relative">
          <button
            onClick={() => {
              setShowAlerts(!showAlerts);
              setShowNotifications(false);
            }}
            className={cn(
              "relative p-2.5 rounded-lg transition-all duration-200",
              showAlerts 
                ? "bg-[rgba(123,97,255,0.15)] text-[#7B61FF]" 
                : "text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] hover:text-[#F5F7FF]"
            )}
          >
            <AlertTriangle className="w-5 h-5" />
            {unreadCount > 0 && (
              <span className="absolute -top-1 -right-1 w-5 h-5 bg-[#FF6B6B] rounded-full text-[10px] font-medium text-white flex items-center justify-center">
                {unreadCount}
              </span>
            )}
          </button>

          {/* Alerts Dropdown */}
          {showAlerts && (
            <div className="absolute right-0 top-full mt-2 w-96 glass-panel overflow-hidden animate-fade-in">
              <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
                <h3 className="font-semibold text-[#F5F7FF]">Alerts</h3>
                <button 
                  onClick={() => setNotificationList(prev => prev.map(n => ({ ...n, isRead: true })))}
                  className="text-xs text-[#7B61FF] hover:underline"
                >
                  Mark all read
                </button>
              </div>
              <div className="max-h-80 overflow-y-auto scrollbar-thin">
                {notificationList.map((alert) => (
                  <div 
                    key={alert.id}
                    className={cn(
                      "p-4 border-b border-[rgba(123,97,255,0.08)] hover:bg-[rgba(123,97,255,0.04)] transition-colors cursor-pointer",
                      !alert.isRead && "bg-[rgba(123,97,255,0.03)]"
                    )}
                    onClick={() => markAsRead(alert.id)}
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5">{getAlertIcon(alert.type)}</div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-[#F5F7FF]">{alert.title}</p>
                          {!alert.isRead && (
                            <span className="w-2 h-2 bg-[#7B61FF] rounded-full" />
                          )}
                        </div>
                        <p className="text-xs text-[#A7B1C8] mt-0.5 line-clamp-2">{alert.message}</p>
                        <p className="text-[10px] text-[#A7B1C8] mt-1.5">{alert.timestamp}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Notifications Button */}
        <div className="relative">
          <button
            onClick={() => {
              setShowNotifications(!showNotifications);
              setShowAlerts(false);
            }}
            className={cn(
              "relative p-2.5 rounded-lg transition-all duration-200",
              showNotifications 
                ? "bg-[rgba(123,97,255,0.15)] text-[#7B61FF]" 
                : "text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] hover:text-[#F5F7FF]"
            )}
          >
            <Bell className="w-5 h-5" />
            <span className="absolute -top-1 -right-1 w-5 h-5 bg-[#7B61FF] rounded-full text-[10px] font-medium text-white flex items-center justify-center">
              3
            </span>
          </button>

          {/* Notifications Dropdown */}
          {showNotifications && (
            <div className="absolute right-0 top-full mt-2 w-80 glass-panel overflow-hidden animate-fade-in">
              <div className="p-4 border-b border-[rgba(123,97,255,0.12)]">
                <h3 className="font-semibold text-[#F5F7FF]">Notifications</h3>
              </div>
              <div className="p-8 text-center">
                <Bell className="w-12 h-12 text-[rgba(123,97,255,0.3)] mx-auto mb-3" />
                <p className="text-sm text-[#A7B1C8]">No new notifications</p>
              </div>
            </div>
          )}
        </div>

        {/* Role Indicator */}
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[rgba(123,97,255,0.1)] border border-[rgba(123,97,255,0.2)]">
          <div className="w-2 h-2 bg-[#2DD4A8] rounded-full animate-pulse" />
          <span className="text-xs font-medium text-[#F5F7FF]">Live</span>
        </div>
      </div>
    </header>
  );
}
