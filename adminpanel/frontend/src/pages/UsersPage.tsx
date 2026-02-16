import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { 
  Search, 
  Filter, 
  Ban, 
  Shield, 
  Edit, 
  Eye, 
  X,
  Heart,
  Star,
  Flag,
  Calendar,
  MapPin,
  Clock,
  Phone,
  ChevronLeft,
  ChevronRight
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { users } from '@/data/mockData';
import type { User } from '@/types';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

const statusConfig = {
  online: { bg: 'bg-[rgba(45,212,168,0.15)]', text: 'text-[#2DD4A8]', dot: 'bg-[#2DD4A8]' },
  away: { bg: 'bg-[rgba(255,209,102,0.15)]', text: 'text-[#FFD166]', dot: 'bg-[#FFD166]' },
  offline: { bg: 'bg-[rgba(167,177,200,0.15)]', text: 'text-[#A7B1C8]', dot: 'bg-[#A7B1C8]' },
};

type InteractionType = 'matches' | 'likes_sent' | 'likes_received';

const defaultInterests = [
  'Travel',
  'Music',
  'Fitness',
  'Coffee',
  'Photography',
  'Hiking',
  'Movies',
  'Cooking',
  'Reading',
  'Art',
];

function formatJoinedLabel(joined: string): string {
  const joinedDate = new Date(`${joined}T00:00:00`);
  if (Number.isNaN(joinedDate.getTime())) {
    return joined;
  }

  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const startOfJoined = new Date(
    joinedDate.getFullYear(),
    joinedDate.getMonth(),
    joinedDate.getDate(),
  );

  const daysSinceJoin = Math.max(
    0,
    Math.floor((startOfToday.getTime() - startOfJoined.getTime()) / (1000 * 60 * 60 * 24)),
  );

  const dd = String(startOfJoined.getDate()).padStart(2, '0');
  const mm = String(startOfJoined.getMonth() + 1).padStart(2, '0');
  const yyyy = String(startOfJoined.getFullYear());
  const dayLabel = daysSinceJoin === 1 ? 'day' : 'days';

  return `${daysSinceJoin} ${dayLabel} • ${dd}.${mm}.${yyyy}`;
}

function stableHash(value: string): number {
  return value.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
}

function resolveLikesSent(user: User): number {
  if (typeof user.likesSent === 'number') {
    return user.likesSent;
  }
  return Math.max(1, Math.round(user.likes * 0.62));
}

function resolveHeight(user: User): number {
  if (typeof user.heightCm === 'number') {
    return user.heightCm;
  }
  return 158 + (stableHash(user.id) % 28);
}

function resolveEyeColor(user: User): string {
  if (user.eyeColor) {
    return user.eyeColor;
  }
  const eyeColors = ['Brown', 'Blue', 'Green', 'Hazel', 'Gray'];
  return eyeColors[stableHash(user.id) % eyeColors.length];
}

function resolveInterests(user: User): string[] {
  if (user.interests && user.interests.length > 0) {
    return user.interests.slice(0, 5);
  }

  const start = stableHash(user.id) % defaultInterests.length;
  return Array.from({ length: 5 }, (_, index) => defaultInterests[(start + index) % defaultInterests.length]);
}

function resolveInteractionProfiles(currentUser: User, type: InteractionType): User[] {
  const candidates = users.filter((candidate) => candidate.id !== currentUser.id);
  if (candidates.length === 0) {
    return [];
  }

  const baseOffset =
    type === 'matches'
      ? 0
      : type === 'likes_sent'
        ? 2
        : 4;
  const hash = stableHash(currentUser.id);
  const start = (hash + baseOffset) % candidates.length;
  const length = Math.min(8, candidates.length);

  return Array.from({ length }, (_, index) => candidates[(start + index) % candidates.length]);
}

function UserProfileModal({
  user,
  onClose,
  onOpenUser,
}: {
  user: User;
  onClose: () => void;
  onOpenUser: (nextUser: User) => void;
}) {
  const [activeTab, setActiveTab] = useState<'activity' | 'limits' | 'moderation'>('activity');
  const [viewerOpen, setViewerOpen] = useState(false);
  const [viewerIndex, setViewerIndex] = useState(0);
  const [activeInteraction, setActiveInteraction] = useState<InteractionType | null>(null);
  const { hasPermission, role } = usePermissions();
  const canChangeLimits = hasPermission(ADMIN_PERMISSIONS.change_limits);
  const canBanUsers = hasPermission(ADMIN_PERMISSIONS.ban_users);
  const profilePhotos = user.photos?.length === 3 ? user.photos : [user.avatar, user.avatar, user.avatar];
  const likesSent = resolveLikesSent(user);
  const profileHeight = resolveHeight(user);
  const eyeColor = resolveEyeColor(user);
  const interests = resolveInterests(user);
  const interactionProfiles = activeInteraction ? resolveInteractionProfiles(user, activeInteraction) : [];
  const interactionTitle =
    activeInteraction === 'matches'
      ? 'Matches'
      : activeInteraction === 'likes_sent'
        ? 'Likes Sent'
        : 'Likes Received';

  const openViewer = (index: number) => {
    setViewerIndex(index);
    setViewerOpen(true);
  };

  const closeViewer = () => {
    setViewerOpen(false);
    setViewerIndex(0);
  };

  const nextViewer = () => {
    setViewerIndex((prev) => (prev + 1) % profilePhotos.length);
  };

  const prevViewer = () => {
    setViewerIndex((prev) => (prev - 1 + profilePhotos.length) % profilePhotos.length);
  };

  useEffect(() => {
    if (!viewerOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeViewer();
      }
      if (event.key === 'ArrowRight') {
        nextViewer();
      }
      if (event.key === 'ArrowLeft') {
        prevViewer();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [viewerOpen, profilePhotos.length]);

  useEffect(() => {
    setActiveTab('activity');
    setViewerOpen(false);
    setViewerIndex(0);
    setActiveInteraction(null);
  }, [user.id]);

  const handleEditLimits = () => {
    if (!canChangeLimits) {
      return;
    }

    logAdminAction(
      `edit_limits_${user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  const handleBanUser = () => {
    if (!canBanUsers) {
      return;
    }

    logAdminAction(
      `ban_user_${user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  const modalContent = (
    <div className="fixed inset-0 z-[80] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-2xl glass-panel max-h-[90vh] overflow-hidden flex flex-col animate-slide-up">
        {/* Header */}
        <div className="p-6 border-b border-[rgba(123,97,255,0.12)]">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={() => openViewer(0)}
                className="relative rounded-2xl transition hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.3)]"
                aria-label="Open profile photos"
              >
                <img
                  src={user.avatar}
                  alt={user.name}
                  className="w-20 h-20 rounded-2xl border-2 border-[rgba(123,97,255,0.25)] cursor-zoom-in"
                />
                <span
                  className={cn(
                    "absolute -bottom-1 -right-1 w-5 h-5 rounded-full border-2 border-[#0E1320]",
                    statusConfig[user.status].dot,
                  )}
                />
              </button>
              <div>
                <h3 className="text-xl font-bold text-[#F5F7FF]">
                  {user.name}, {user.age}
                </h3>
                <div className="mt-0.5 flex flex-wrap items-center gap-2 text-sm text-[#A7B1C8]">
                  <span>{user.handle}</span>
                  <span className="text-[rgba(167,177,200,0.5)]">•</span>
                  <span className="inline-flex items-center gap-1.5">
                    <Phone className="w-3.5 h-3.5" />
                    {user.phone}
                  </span>
                </div>
                <div className="flex items-center gap-3 mt-2">
                  <span
                    className={cn(
                      "px-2 py-0.5 rounded-full text-xs font-medium",
                      statusConfig[user.status].bg,
                      statusConfig[user.status].text,
                    )}
                  >
                    {user.status}
                  </span>
                  {user.isPremium && (
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[rgba(123,97,255,0.15)] text-[#7B61FF] flex items-center gap-1">
                      <Star className="w-3 h-3" />
                      {user.subscriptionTier}
                    </span>
                  )}
                  <span className="text-xs text-[#A7B1C8]">
                    {profileHeight} cm • {eyeColor} eyes
                  </span>
                </div>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[rgba(123,97,255,0.12)]">
          {(['activity', 'limits', 'moderation'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={cn(
                "flex-1 py-3 text-sm font-medium capitalize transition-colors relative",
                activeTab === tab
                  ? "text-[#F5F7FF]"
                  : "text-[#A7B1C8] hover:text-[#F5F7FF]",
              )}
            >
              {tab}
              {activeTab === tab && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-[#7B61FF]" />
              )}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto scrollbar-thin flex-1">
          {activeTab === 'activity' && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <button
                  onClick={() => setActiveInteraction('matches')}
                  className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                >
                  <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                    <Heart className="w-4 h-4" />
                    <span className="text-sm">Matches</span>
                  </div>
                  <p className="text-2xl font-bold text-[#F5F7FF]">{user.matches}</p>
                </button>
                <button
                  onClick={() => setActiveInteraction('likes_sent')}
                  className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                >
                  <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                    <Heart className="w-4 h-4" />
                    <span className="text-sm">Likes Sent</span>
                  </div>
                  <p className="text-2xl font-bold text-[#F5F7FF]">{likesSent}</p>
                </button>
                <button
                  onClick={() => setActiveInteraction('likes_received')}
                  className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                >
                  <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                    <Star className="w-4 h-4" />
                    <span className="text-sm">Likes Received</span>
                  </div>
                  <p className="text-2xl font-bold text-[#F5F7FF]">{user.likes}</p>
                </button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <Clock className="w-5 h-5 text-[#A7B1C8]" />
                  <div>
                    <p className="text-sm text-[#A7B1C8]">Last Active</p>
                    <p className="text-sm text-[#F5F7FF]">{user.lastActive}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <Calendar className="w-5 h-5 text-[#A7B1C8]" />
                  <div>
                    <p className="text-sm text-[#A7B1C8]">Joined</p>
                    <p className="text-sm text-[#F5F7FF]">{formatJoinedLabel(user.joined)}</p>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                <MapPin className="w-5 h-5 text-[#A7B1C8]" />
                <div>
                  <p className="text-sm text-[#A7B1C8]">Location</p>
                  <p className="text-sm text-[#F5F7FF]">{user.location}</p>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <p className="text-sm text-[#A7B1C8] mb-2">Profile</p>
                <div className="grid grid-cols-2 gap-3 text-sm mb-3">
                  <div>
                    <p className="text-[#A7B1C8] text-xs">Gender</p>
                    <p className="text-[#F5F7FF]">{user.gender}</p>
                  </div>
                  <div>
                    <p className="text-[#A7B1C8] text-xs">Age</p>
                    <p className="text-[#F5F7FF]">{user.age}</p>
                  </div>
                  <div>
                    <p className="text-[#A7B1C8] text-xs">Height</p>
                    <p className="text-[#F5F7FF]">{profileHeight} cm</p>
                  </div>
                  <div>
                    <p className="text-[#A7B1C8] text-xs">Eyes</p>
                    <p className="text-[#F5F7FF]">{eyeColor}</p>
                  </div>
                </div>
                <p className="text-sm text-[#A7B1C8] mb-1">Bio</p>
                <p className="text-sm text-[#F5F7FF]">{user.bio || 'No bio provided.'}</p>
              </div>

              <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <p className="text-sm text-[#A7B1C8] mb-3">Interests</p>
                <div className="flex flex-wrap gap-2">
                  {interests.map((interest) => (
                    <span
                      key={interest}
                      className="px-2.5 py-1 rounded-full text-xs bg-[rgba(123,97,255,0.14)] border border-[rgba(123,97,255,0.25)] text-[#CFC6FF]"
                    >
                      {interest}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'limits' && (
            <div className="space-y-4">
              <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm text-[#A7B1C8]">Daily Swipes</span>
                  <span className="text-sm text-[#F5F7FF]">Unlimited</span>
                </div>
                <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                  <div className="h-full w-full rounded-full bg-[#7B61FF]" />
                </div>
              </div>

              <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm text-[#A7B1C8]">Super Likes</span>
                  <span className="text-sm text-[#F5F7FF]">5 / 5 remaining</span>
                </div>
                <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                  <div className="h-full w-full rounded-full bg-[#4CC9F0]" />
                </div>
              </div>

              <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm text-[#A7B1C8]">Boosts</span>
                  <span className="text-sm text-[#F5F7FF]">2 / 3 remaining</span>
                </div>
                <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                  <div className="h-full w-[66%] rounded-full bg-[#2DD4A8]" />
                </div>
              </div>
            </div>
          )}

          {activeTab === 'moderation' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-[rgba(45,212,168,0.15)] flex items-center justify-center">
                    <Shield className="w-5 h-5 text-[#2DD4A8]" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-[#F5F7FF]">Trust Score</p>
                    <p className="text-xs text-[#A7B1C8]">Based on activity and reports</p>
                  </div>
                </div>
                <span
                  className={cn(
                    "text-2xl font-bold",
                    user.trustScore >= 90
                      ? "text-[#2DD4A8]"
                      : user.trustScore >= 70
                        ? "text-[#FFD166]"
                        : "text-[#FF6B6B]",
                  )}
                >
                  {user.trustScore}
                </span>
              </div>

              <div className="p-4 rounded-xl bg-[rgba(255,107,107,0.05)] border border-[rgba(255,107,107,0.2)]">
                <div className="flex items-center gap-2 mb-3">
                  <Flag className="w-4 h-4 text-[#FF6B6B]" />
                  <span className="text-sm font-medium text-[#FF6B6B]">Reports</span>
                </div>
                <p className="text-sm text-[#A7B1C8]">This user has no pending reports.</p>
              </div>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="p-4 border-t border-[rgba(123,97,255,0.12)] flex gap-3">
          <button
            onClick={handleEditLimits}
            disabled={!canChangeLimits}
            className="flex-1 btn-secondary flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Edit className="w-4 h-4" />
            Edit Limits
          </button>
          <button
            onClick={handleBanUser}
            disabled={!canBanUsers}
            className="flex-1 btn-danger flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Ban className="w-4 h-4" />
            Ban User
          </button>
        </div>
      </div>

      {activeInteraction && (
        <div
          className="fixed inset-0 z-[85] flex items-center justify-center p-4"
          onClick={() => setActiveInteraction(null)}
        >
          <div className="absolute inset-0 bg-black/55 backdrop-blur-[2px]" />
          <div
            className="relative w-full max-w-md glass-panel max-h-[72vh] overflow-hidden flex flex-col"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
              <div>
                <p className="text-sm text-[#A7B1C8]">{interactionTitle}</p>
                <p className="text-xs text-[#A7B1C8]">{interactionProfiles.length} profiles</p>
              </div>
              <button
                onClick={() => setActiveInteraction(null)}
                className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="p-3 space-y-2 overflow-y-auto scrollbar-thin">
              {interactionProfiles.map((profile) => (
                <button
                  key={`${activeInteraction}-${profile.id}`}
                  onClick={() => {
                    setActiveInteraction(null);
                    onOpenUser(profile);
                  }}
                  className="w-full text-left flex items-center gap-3 p-2 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                >
                  <img
                    src={profile.avatar}
                    alt={profile.name}
                    className="w-11 h-11 rounded-xl border border-[rgba(123,97,255,0.25)] object-cover"
                  />
                  <div>
                    <p className="text-sm font-medium text-[#F5F7FF]">
                      {profile.name}, {profile.age}
                    </p>
                    <p className="text-xs text-[#A7B1C8]">{profile.handle}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {viewerOpen && (
        <div
          className="fixed inset-0 z-[90] bg-[rgba(6,8,14,0.86)] backdrop-blur-sm flex items-center justify-center p-4"
          onClick={closeViewer}
        >
          <div className="relative w-full max-w-4xl" onClick={(event) => event.stopPropagation()}>
            <button
              onClick={closeViewer}
              className="absolute -top-11 right-0 px-3 py-2 rounded-lg bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.2)] text-[#F5F7FF] text-sm hover:bg-[rgba(123,97,255,0.12)]"
            >
              Close
            </button>

            <div className="relative rounded-2xl overflow-hidden border border-[rgba(123,97,255,0.2)] bg-[rgba(14,19,32,0.6)]">
              <img
                src={profilePhotos[viewerIndex]}
                alt={`${user.name} photo ${viewerIndex + 1}`}
                className="w-full max-h-[78vh] object-contain"
              />

              {profilePhotos.length > 1 && (
                <button
                  onClick={prevViewer}
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-[rgba(14,19,32,0.75)] border border-[rgba(123,97,255,0.2)] flex items-center justify-center text-[#F5F7FF] hover:bg-[rgba(123,97,255,0.14)]"
                >
                  <ChevronLeft className="w-5 h-5" />
                </button>
              )}

              {profilePhotos.length > 1 && (
                <button
                  onClick={nextViewer}
                  className="absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-[rgba(14,19,32,0.75)] border border-[rgba(123,97,255,0.2)] flex items-center justify-center text-[#F5F7FF] hover:bg-[rgba(123,97,255,0.14)]"
                >
                  <ChevronRight className="w-5 h-5" />
                </button>
              )}
            </div>

            <div className="mt-3 flex items-center justify-between text-xs text-[#A7B1C8]">
              <span>
                {viewerIndex + 1} / {profilePhotos.length}
              </span>
              <div className="flex items-center gap-2">
                {profilePhotos.map((_, index) => (
                  <button
                    key={`photo-dot-${index}`}
                    onClick={() => setViewerIndex(index)}
                    className={cn(
                      "w-2.5 h-2.5 rounded-full border border-[rgba(123,97,255,0.35)]",
                      index === viewerIndex
                        ? "bg-[rgba(123,97,255,0.9)]"
                        : "bg-[rgba(245,247,255,0.12)]",
                    )}
                  />
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );

  return createPortal(modalContent, document.body);
}

export function UsersPage() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());
  const { hasPermission, role } = usePermissions();
  const canEditUsers = hasPermission(ADMIN_PERMISSIONS.edit_users);
  const canBanUsers = hasPermission(ADMIN_PERMISSIONS.ban_users);
  const canViewPrivateData = hasPermission(ADMIN_PERMISSIONS.view_private_data);

  const filteredUsers = users.filter(user => 
    user.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.handle.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.email.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const toggleRow = (id: string) => {
    const newSet = new Set(selectedRows);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedRows(newSet);
  };

  const toggleAll = () => {
    if (selectedRows.size === filteredUsers.length) {
      setSelectedRows(new Set());
    } else {
      setSelectedRows(new Set(filteredUsers.map(u => u.id)));
    }
  };

  const handleBulkBan = () => {
    if (!canBanUsers || selectedRows.size === 0) {
      return;
    }

    logAdminAction(
      `bulk_ban_users_${selectedRows.size}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  const openUserProfile = (user: User) => {
    if (!canViewPrivateData) {
      return;
    }

    logAdminAction(
      `view_user_profile_${user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
    setSelectedUser(user);
  };

  const handleEditUser = (user: User) => {
    if (!canEditUsers) {
      return;
    }

    logAdminAction(
      `edit_user_${user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  const handleBanUser = (user: User) => {
    if (!canBanUsers) {
      return;
    }

    logAdminAction(
      `ban_user_${user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { label: 'Total Users', value: '125,420', icon: '📊', color: '#7B61FF' },
          { label: 'New Today', value: '+3,847', icon: '👤', color: '#2DD4A8' },
          { label: 'Premium Users', value: '182,400', icon: '⭐', color: '#FFD166' },
          { label: 'Active Now', value: '84,231', icon: '🔥', color: '#FF6B6B' },
        ].map((stat, i) => (
          <div key={i} className="glass-panel p-4 flex items-center gap-4">
            <div 
              className="w-12 h-12 rounded-xl flex items-center justify-center text-2xl"
              style={{ background: `${stat.color}20` }}
            >
              {stat.icon}
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">{stat.label}</p>
              <p className="text-xl font-bold text-[#F5F7FF]">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Table */}
      <div className="glass-panel overflow-hidden">
        {/* Table Toolbar */}
        <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center gap-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A7B1C8]" />
            <input
              type="text"
              placeholder="Search users..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)]"
            />
          </div>
          <button className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors">
            <Filter className="w-4 h-4" />
            Filter
          </button>
          {selectedRows.size > 0 && (
            <div className="flex items-center gap-2 ml-auto">
              <span className="text-sm text-[#A7B1C8]">{selectedRows.size} selected</span>
              <button
                onClick={handleBulkBan}
                disabled={!canBanUsers}
                className="btn-danger text-xs py-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Ban Selected
              </button>
            </div>
          )}
        </div>

        {/* Table Content */}
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[rgba(123,97,255,0.12)]">
                <th className="p-4 text-left">
                  <input
                    type="checkbox"
                    checked={selectedRows.size === filteredUsers.length && filteredUsers.length > 0}
                    onChange={toggleAll}
                    className="w-4 h-4 rounded border-[rgba(123,97,255,0.3)] bg-transparent text-[#7B61FF] focus:ring-[#7B61FF]"
                  />
                </th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">User</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">ID</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Status</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Location</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Trust Score</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Subscription</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => (
                <tr 
                  key={user.id} 
                  className="border-b border-[rgba(123,97,255,0.08)] table-row"
                >
                  <td className="p-4">
                    <input
                      type="checkbox"
                      checked={selectedRows.has(user.id)}
                      onChange={() => toggleRow(user.id)}
                      className="w-4 h-4 rounded border-[rgba(123,97,255,0.3)] bg-transparent text-[#7B61FF] focus:ring-[#7B61FF]"
                    />
                  </td>
                  <td className="p-4">
                    <div className="flex items-center gap-3">
                      <div className="relative">
                        <img 
                          src={user.avatar} 
                          alt={user.name}
                          className="w-10 h-10 rounded-full border-2 border-[rgba(123,97,255,0.25)]"
                        />
                        <span className={cn(
                          "absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[#0E1320]",
                          statusConfig[user.status].dot
                        )} />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-[#F5F7FF]">{user.name}</p>
                        <p className="text-xs text-[#A7B1C8]">{user.handle}</p>
                      </div>
                    </div>
                  </td>
                  <td className="p-4">
                    <span className="font-mono text-xs text-[#A7B1C8]">{user.id}</span>
                  </td>
                  <td className="p-4">
                    <span className={cn(
                      "px-2 py-1 rounded-full text-xs font-medium capitalize",
                      statusConfig[user.status].bg,
                      statusConfig[user.status].text
                    )}>
                      {user.status}
                    </span>
                  </td>
                  <td className="p-4">
                    <p className="text-sm text-[#F5F7FF]">{user.location}</p>
                  </td>
                  <td className="p-4">
                    <span className={cn(
                      "text-sm font-medium",
                      user.trustScore >= 90 ? "text-[#2DD4A8]" :
                      user.trustScore >= 70 ? "text-[#FFD166]" : "text-[#FF6B6B]"
                    )}>
                      {user.trustScore}
                    </span>
                  </td>
                  <td className="p-4">
                    {user.isPremium ? (
                      <span className="px-2 py-1 rounded-full text-xs font-medium bg-[rgba(123,97,255,0.15)] text-[#7B61FF] flex items-center gap-1 w-fit">
                        <Star className="w-3 h-3" />
                        {user.subscriptionTier}
                      </span>
                    ) : (
                      <span className="text-xs text-[#A7B1C8]">Free</span>
                    )}
                  </td>
                  <td className="p-4">
                    <div className="flex items-center gap-2">
                      <button 
                        onClick={() => openUserProfile(user)}
                        disabled={!canViewPrivateData}
                        className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        title="View Profile"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                      <button 
                        onClick={() => handleEditUser(user)}
                        disabled={!canEditUsers}
                        className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Edit User"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button 
                        onClick={() => handleBanUser(user)}
                        disabled={!canBanUsers}
                        className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(255,107,107,0.15)] hover:text-[#FF6B6B] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Ban User"
                      >
                        <Ban className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="p-4 border-t border-[rgba(123,97,255,0.12)] flex items-center justify-between">
          <p className="text-sm text-[#A7B1C8]">
            Showing <span className="text-[#F5F7FF]">1-10</span> of <span className="text-[#F5F7FF]">125,420</span> users
          </p>
          <div className="flex items-center gap-2">
            <button className="px-3 py-1.5 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors disabled:opacity-50" disabled>
              Previous
            </button>
            <button className="px-3 py-1.5 rounded-lg text-sm bg-[rgba(123,97,255,0.15)] text-[#7B61FF]">
              1
            </button>
            <button className="px-3 py-1.5 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors">
              2
            </button>
            <button className="px-3 py-1.5 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors">
              3
            </button>
            <span className="text-[#A7B1C8]">...</span>
            <button className="px-3 py-1.5 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors">
              12,542
            </button>
            <button className="px-3 py-1.5 rounded-lg text-sm text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] transition-colors">
              Next
            </button>
          </div>
        </div>
      </div>

      {/* User Profile Modal */}
      {selectedUser && (
        <UserProfileModal 
          key={selectedUser.id}
          user={selectedUser} 
          onClose={() => setSelectedUser(null)}
          onOpenUser={setSelectedUser}
        />
      )}
    </div>
  );
}
