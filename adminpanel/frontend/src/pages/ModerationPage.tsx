import { useState } from 'react';
import { moderationQueue } from '@/data/mockData';
import { 
  Check, 
  X, 
  AlertTriangle, 
  Shield, 
  Ban, 
  ChevronRight,
  Image as ImageIcon,
  MessageSquare,
  FileText,
  Filter
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ModerationItem } from '@/types';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

const filterOptions = ['All', 'Photos', 'Bios', 'Messages'];

type ModerationAction = 'approve' | 'remove' | 'escalate' | 'ban';

const actionPermissions = {
  approve: ADMIN_PERMISSIONS.approve_profiles,
  remove: ADMIN_PERMISSIONS.reject_profiles,
  escalate: ADMIN_PERMISSIONS.moderate_profiles,
  ban: ADMIN_PERMISSIONS.ban_users,
} as const;

function ModerationDetail({
  item,
  onAction,
  canAction,
}: {
  item: ModerationItem;
  onAction: (action: ModerationAction) => void;
  canAction: (action: ModerationAction) => boolean;
}) {
  const getIcon = (type: ModerationItem['type']) => {
    switch (type) {
      case 'photo': return <ImageIcon className="w-5 h-5" />;
      case 'bio': return <FileText className="w-5 h-5" />;
      case 'message': return <MessageSquare className="w-5 h-5" />;
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Content Preview */}
      <div className="flex-1 p-6">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-lg bg-[rgba(123,97,255,0.15)] flex items-center justify-center text-[#7B61FF]">
            {getIcon(item.type)}
          </div>
          <div>
            <p className="text-sm font-medium text-[#F5F7FF] capitalize">{item.type} Report</p>
            <p className="text-xs text-[#A7B1C8]">Reported {item.timestamp}</p>
          </div>
        </div>

        {item.thumbnail && (
          <div className="mb-4">
            <img 
              src={item.thumbnail} 
              alt="Reported content"
              className="max-w-md rounded-xl border border-[rgba(123,97,255,0.2)]"
            />
          </div>
        )}

        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] mb-4">
          <p className="text-sm text-[#F5F7FF]">{item.content}</p>
        </div>

        <div className="space-y-2">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-[#A7B1C8]">Reported by:</span>
            <span className="text-[#F5F7FF]">{item.reportedBy}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-[#A7B1C8]">Reason:</span>
            <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(255,107,107,0.15)] text-[#FF6B6B]">
              {item.reason}
            </span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-[#A7B1C8]">User:</span>
            <span className="text-[#F5F7FF]">{item.userName}</span>
            <span className="font-mono text-xs text-[#A7B1C8]">{item.userId}</span>
          </div>
        </div>
      </div>

      {/* Action Bar */}
      <div className="p-4 border-t border-[rgba(123,97,255,0.12)] flex gap-3">
        <button 
          onClick={() => onAction('approve')}
          disabled={!canAction('approve')}
          className="flex-1 btn-primary flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Check className="w-4 h-4" />
          Approve
        </button>
        <button 
          onClick={() => onAction('remove')}
          disabled={!canAction('remove')}
          className="flex-1 btn-danger flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <X className="w-4 h-4" />
          Remove
        </button>
        <button 
          onClick={() => onAction('escalate')}
          disabled={!canAction('escalate')}
          className="px-4 py-2 rounded-lg text-sm font-medium bg-[rgba(255,209,102,0.15)] text-[#FFD166] border border-[rgba(255,209,102,0.25)] hover:bg-[rgba(255,209,102,0.25)] transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <AlertTriangle className="w-4 h-4" />
          Escalate
        </button>
        <button 
          onClick={() => onAction('ban')}
          disabled={!canAction('ban')}
          className="btn-danger flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Ban className="w-4 h-4" />
          Ban User
        </button>
      </div>
    </div>
  );
}

export function ModerationPage() {
  const [selectedFilter, setSelectedFilter] = useState('All');
  const [selectedItem, setSelectedItem] = useState<ModerationItem | null>(moderationQueue[0]);
  const [queue, setQueue] = useState(moderationQueue);
  const { hasPermission, role } = usePermissions();

  const filteredQueue = selectedFilter === 'All' 
    ? queue 
    : queue.filter(item => item.type === selectedFilter.toLowerCase().slice(0, -1));

  const canAction = (action: ModerationAction) => hasPermission(actionPermissions[action]);

  const handleAction = (action: ModerationAction) => {
    if (!selectedItem || !canAction(action)) {
      return;
    }

    logAdminAction(
      `moderation_${action}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );

    setQueue(prev => prev.filter(item => item.id !== selectedItem.id));
    const remaining = queue.filter(item => item.id !== selectedItem.id);
    setSelectedItem(remaining[0] || null);
  };

  return (
    <div className="p-6 h-[calc(100vh-64px)] animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-[#F5F7FF]">Moderation</h2>
          <p className="text-sm text-[#A7B1C8]">Review and moderate reported content</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[rgba(255,107,107,0.1)] border border-[rgba(255,107,107,0.2)]">
            <AlertTriangle className="w-4 h-4 text-[#FF6B6B]" />
            <span className="text-sm text-[#FF6B6B]">{queue.length} pending</span>
          </div>
          <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[rgba(45,212,168,0.1)] border border-[rgba(45,212,168,0.2)]">
            <Shield className="w-4 h-4 text-[#2DD4A8]" />
            <span className="text-sm text-[#2DD4A8]">Auto-approved: 94%</span>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="glass-panel h-[calc(100%-80px)] overflow-hidden flex">
        {/* Queue List */}
        <div className="w-80 border-r border-[rgba(123,97,255,0.12)] flex flex-col">
          {/* Filters */}
          <div className="p-3 border-b border-[rgba(123,97,255,0.12)]">
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-[#A7B1C8]" />
              <div className="flex gap-1">
                {filterOptions.map((filter) => (
                  <button
                    key={filter}
                    onClick={() => setSelectedFilter(filter)}
                    className={cn(
                      "px-2 py-1 rounded text-xs font-medium transition-colors",
                      selectedFilter === filter
                        ? "bg-[rgba(123,97,255,0.2)] text-[#7B61FF]"
                        : "text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)]"
                    )}
                  >
                    {filter}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Queue Items */}
          <div className="flex-1 overflow-y-auto scrollbar-thin">
            {filteredQueue.map((item) => (
              <button
                key={item.id}
                onClick={() => setSelectedItem(item)}
                className={cn(
                  "w-full p-4 text-left border-b border-[rgba(123,97,255,0.08)] transition-colors",
                  selectedItem?.id === item.id
                    ? "bg-[rgba(123,97,255,0.1)]"
                    : "hover:bg-[rgba(123,97,255,0.05)]"
                )}
              >
                <div className="flex items-start gap-3">
                  {item.thumbnail ? (
                    <img 
                      src={item.thumbnail} 
                      alt=""
                      className="w-12 h-12 rounded-lg object-cover border border-[rgba(123,97,255,0.2)]"
                    />
                  ) : (
                    <div className="w-12 h-12 rounded-lg bg-[rgba(123,97,255,0.1)] flex items-center justify-center">
                      <FileText className="w-5 h-5 text-[#A7B1C8]" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium text-[#7B61FF] uppercase">{item.type}</span>
                      <span className="text-xs text-[#A7B1C8]">{item.timestamp}</span>
                    </div>
                    <p className="text-sm text-[#F5F7FF] truncate mt-1">{item.content}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-[#A7B1C8]">{item.userName}</span>
                      <span className="px-1.5 py-0.5 rounded text-[10px] bg-[rgba(255,107,107,0.15)] text-[#FF6B6B]">
                        {item.reason}
                      </span>
                    </div>
                  </div>
                  <ChevronRight className={cn(
                    "w-4 h-4 text-[#A7B1C8] flex-shrink-0",
                    selectedItem?.id === item.id && "text-[#7B61FF]"
                  )} />
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Review Workspace */}
        <div className="flex-1">
          {selectedItem ? (
            <ModerationDetail 
              item={selectedItem} 
              onAction={handleAction}
              canAction={canAction}
            />
          ) : (
            <div className="h-full flex items-center justify-center">
              <div className="text-center">
                <Shield className="w-16 h-16 text-[rgba(123,97,255,0.2)] mx-auto mb-4" />
                <p className="text-lg font-medium text-[#F5F7FF]">All caught up!</p>
                <p className="text-sm text-[#A7B1C8]">No more items in the moderation queue</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
