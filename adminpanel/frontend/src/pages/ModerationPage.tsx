import { useEffect, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { 
  Check, 
  X, 
  AlertTriangle, 
  Ban, 
  ChevronRight,
  ChevronLeft,
  Image as ImageIcon,
  MessageSquare,
  Filter,
  Shield,
  Phone,
  Star,
  Heart,
  Clock,
  Calendar,
  MapPin,
  Edit,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';
import {
  isSupportApiConfigured,
  listSupportConversations,
  listSupportMessages,
  markSupportConversationRead,
  sendSupportMessage,
  setSupportConversationStatus,
  type SupportConversationDTO,
  type SupportConversationStatus,
  type SupportMessageDTO,
} from '@/lib/supportApi';

type ModerationCaseType = 'onboarding' | 'report' | 'support';
type ModerationPriority = 'low' | 'med' | 'high';
type ModerationStatus =
  | 'new'
  | 'in_review'
  | 'changes_requested'
  | 'approved'
  | 'rejected'
  | 'waiting_user'
  | 'resolved';

type ReportSubType = 'fake_profile' | 'scammer' | 'under_18' | 'other';
type OnboardingSubType = 'new_user' | 'profile_update';
type SupportSubType = 'payments' | 'account' | 'bugs' | 'safety' | 'other';

type SupportMsgFrom = 'user' | 'admin' | 'system';
type ProfileInteractionType = 'matches' | 'likes_sent' | 'likes_received';
type ProfileLimitKind = 'daily_swipes' | 'super_likes' | 'boosts';

type ProfileLimitsState = {
  dailySwipesRemaining: number;
  dailySwipesTotal: number;
  superLikesRemaining: number;
  superLikesTotal: number;
  boostsRemaining: number;
  boostsTotal: number;
};

interface UserSummary {
  id: string;
  name: string;
  username?: string;
  avatar?: string;
  phone?: string;
  age?: number;
  birthday?: string;
  zodiac?: string;
  city?: string;
  premium?: boolean;
  gender?: string;
  lookingFor?: string;
  datingGoal?: string;
  language?: string;
  bio?: string;
  photos?: string[];
  heightCm?: number;
  eyeColor?: string;
  interests?: string[];
  joinedAt?: string;
  lastActiveLabel?: string;
}

interface OnboardingProfileData {
  photos: string[]; // always 3
  displayName: string;
  birthday: string;
  zodiac: string;
  gender: string;
  lookingFor: string;
  datingGoal: string;
  city: string;
  language: string;
  bio: string;
}

interface SupportMessage {
  id: string;
  from: SupportMsgFrom;
  text: string;
  timestampLabel: string;
}

interface HistoryEvent {
  id: string;
  text: string;
  timestampLabel: string;
}

interface ModerationCase {
  id: string;
  type: ModerationCaseType;
  subType: ReportSubType | OnboardingSubType | SupportSubType;
  status: ModerationStatus;
  priority: ModerationPriority;
  createdAtLabel: string;
  createdAtTs: number;

  title: string;
  preview: string;

  user: UserSummary;

  // Reports
  reportReason?: string;
  reportComment?: string;
  reportedBy?: string;
  reporter?: UserSummary;
  contentText?: string;
  contentMediaUrl?: string;

  // Onboarding
  onboarding?: OnboardingProfileData;

  // Support
  category?: SupportSubType;
  slaLabel?: string;
  messages?: SupportMessage[];

  // Shared
  tags?: string[];
  attachments?: string[];
  history?: HistoryEvent[];
}

const unifiedCases: ModerationCase[] = [
  {
    id: 'ob_001',
    type: 'onboarding',
    subType: 'new_user',
    status: 'new',
    priority: 'high',
    createdAtLabel: '5m ago',
    createdAtTs: 1760572500000,
    title: 'Onboarding profile review: Anna Kim',
    preview: 'Profile completed with 3 photos and full bio fields.',
    user: {
      id: 'u_1001',
      name: 'Anna Kim',
      username: '@annak',
      avatar: 'https://i.pravatar.cc/150?img=11',
      age: 24,
      zodiac: 'Taurus',
      city: 'Warsaw',
      premium: false,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob001a/600/800',
        'https://picsum.photos/seed/ob001b/600/800',
        'https://picsum.photos/seed/ob001c/600/800',
      ],
      displayName: 'Anna',
      birthday: '2001-05-14',
      zodiac: 'Taurus',
      gender: 'Female',
      lookingFor: 'Male',
      datingGoal: 'Long-term relationship',
      city: 'Warsaw',
      language: 'English',
      bio: 'Product designer, coffee lover, weekend hiking fan.',
    },
    history: [
      { id: 'ob_001_h1', text: 'Registration started in Telegram app', timestampLabel: '8m ago' },
      { id: 'ob_001_h2', text: 'Profile form completed', timestampLabel: '6m ago' },
      { id: 'ob_001_h3', text: 'Queued for manual moderation', timestampLabel: '5m ago' },
    ],
  },
  {
    id: 'ob_002',
    type: 'onboarding',
    subType: 'new_user',
    status: 'in_review',
    priority: 'med',
    createdAtLabel: '14m ago',
    createdAtTs: 1760571960000,
    title: 'Onboarding photos check: Maksim D',
    preview: 'Photo set passed blur check, waiting final decision.',
    user: {
      id: 'u_1002',
      name: 'Maksim Druzhin',
      username: '@maks_d',
      avatar: 'https://i.pravatar.cc/150?img=12',
      age: 28,
      zodiac: 'Virgo',
      city: 'Vilnius',
      premium: true,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob002a/600/800',
        'https://picsum.photos/seed/ob002b/600/800',
        'https://picsum.photos/seed/ob002c/600/800',
      ],
      displayName: 'Maks',
      birthday: '1997-09-02',
      zodiac: 'Virgo',
      gender: 'Male',
      lookingFor: 'Female',
      datingGoal: 'Meet new people',
      city: 'Vilnius',
      language: 'Russian',
      bio: 'Gym, books, road trips and black coffee.',
    },
    tags: ['Manual review'],
    history: [
      { id: 'ob_002_h1', text: 'Photos uploaded', timestampLabel: '16m ago' },
      { id: 'ob_002_h2', text: 'Face detection passed on all photos', timestampLabel: '15m ago' },
      { id: 'ob_002_h3', text: 'Assigned to moderator queue', timestampLabel: '14m ago' },
    ],
  },
  {
    id: 'ob_003',
    type: 'onboarding',
    subType: 'profile_update',
    status: 'in_review',
    priority: 'low',
    createdAtLabel: '33m ago',
    createdAtTs: 1760570820000,
    title: 'Profile update moderation: Elena P',
    preview: 'Updated profile bio requires language policy check.',
    user: {
      id: 'u_1003',
      name: 'Elena Petrova',
      username: '@elena_p',
      avatar: 'https://i.pravatar.cc/150?img=13',
      age: 30,
      zodiac: 'Libra',
      city: 'Riga',
      premium: false,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob003a/600/800',
        'https://picsum.photos/seed/ob003b/600/800',
        'https://picsum.photos/seed/ob003c/600/800',
      ],
      displayName: 'Elena',
      birthday: '1995-10-07',
      zodiac: 'Libra',
      gender: 'Female',
      lookingFor: 'Male',
      datingGoal: 'Long-term relationship',
      city: 'Riga',
      language: 'English',
      bio: 'Traveler, dog person, and board game host.',
    },
    history: [
      { id: 'ob_003_h1', text: 'Onboarding completed', timestampLabel: '40m ago' },
      { id: 'ob_003_h2', text: 'Bio language flagged for recheck', timestampLabel: '36m ago' },
      { id: 'ob_003_h3', text: 'Moved to waiting state', timestampLabel: '33m ago' },
    ],
  },
  {
    id: 'ob_004',
    type: 'onboarding',
    subType: 'profile_update',
    status: 'in_review',
    priority: 'high',
    createdAtLabel: '48m ago',
    createdAtTs: 1760569920000,
    title: 'Profile update moderation: Lina S',
    preview: 'Updated photos require age-confidence revalidation.',
    user: {
      id: 'u_1004',
      name: 'Lina S',
      username: '@lina_s',
      avatar: 'https://i.pravatar.cc/150?img=14',
      age: 18,
      zodiac: 'Cancer',
      city: 'Tallinn',
      premium: false,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob004a/600/800',
        'https://picsum.photos/seed/ob004b/600/800',
        'https://picsum.photos/seed/ob004c/600/800',
      ],
      displayName: 'Lina',
      birthday: '2007-07-09',
      zodiac: 'Cancer',
      gender: 'Female',
      lookingFor: 'Male',
      datingGoal: 'Friendship',
      city: 'Tallinn',
      language: 'English',
      bio: 'Art school student and amateur photographer.',
    },
    tags: ['Underage suspicion'],
    history: [
      { id: 'ob_004_h1', text: 'Photo age-check score low', timestampLabel: '55m ago' },
      { id: 'ob_004_h2', text: 'Sent to senior moderation', timestampLabel: '50m ago' },
      { id: 'ob_004_h3', text: 'Waiting for final decision', timestampLabel: '48m ago' },
    ],
  },
  {
    id: 'ob_005',
    type: 'onboarding',
    subType: 'new_user',
    status: 'in_review',
    priority: 'med',
    createdAtLabel: '1h ago',
    createdAtTs: 1760569200000,
    title: 'Onboarding profile review: Victor M',
    preview: 'Complete profile, city and language fields verified.',
    user: {
      id: 'u_1005',
      name: 'Victor Melnik',
      username: '@victorm',
      avatar: 'https://i.pravatar.cc/150?img=15',
      age: 27,
      zodiac: 'Capricorn',
      city: 'Prague',
      premium: true,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob005a/600/800',
        'https://picsum.photos/seed/ob005b/600/800',
        'https://picsum.photos/seed/ob005c/600/800',
      ],
      displayName: 'Victor',
      birthday: '1999-01-03',
      zodiac: 'Capricorn',
      gender: 'Male',
      lookingFor: 'Female',
      datingGoal: 'Long-term relationship',
      city: 'Prague',
      language: 'English',
      bio: 'Backend engineer, cycling, and live music.',
    },
    history: [
      { id: 'ob_005_h1', text: 'User submitted profile', timestampLabel: '1h 8m ago' },
      { id: 'ob_005_h2', text: 'Auto checks passed', timestampLabel: '1h 4m ago' },
      { id: 'ob_005_h3', text: 'Manual review started', timestampLabel: '1h ago' },
    ],
  },
  {
    id: 'ob_006',
    type: 'onboarding',
    subType: 'profile_update',
    status: 'new',
    priority: 'high',
    createdAtLabel: '1h ago',
    createdAtTs: 1760569199000,
    title: 'Profile update moderation: Kate V',
    preview: 'Updated photos require explicit-content revalidation.',
    user: {
      id: 'u_1006',
      name: 'Kate Voronova',
      username: '@kate_v',
      avatar: 'https://i.pravatar.cc/150?img=16',
      age: 25,
      zodiac: 'Scorpio',
      city: 'Berlin',
      premium: false,
    },
    onboarding: {
      photos: [
        'https://picsum.photos/seed/ob006a/600/800',
        'https://picsum.photos/seed/ob006b/600/800',
        'https://picsum.photos/seed/ob006c/600/800',
      ],
      displayName: 'Kate',
      birthday: '2000-11-19',
      zodiac: 'Scorpio',
      gender: 'Female',
      lookingFor: 'Female',
      datingGoal: 'Meet new people',
      city: 'Berlin',
      language: 'English',
      bio: 'Illustrator, vinyl collector, fan of night walks.',
    },
    tags: ['NSFW risk'],
    history: [
      { id: 'ob_006_h1', text: 'Photo set uploaded', timestampLabel: '1h 12m ago' },
      { id: 'ob_006_h2', text: 'NSFW model confidence elevated', timestampLabel: '1h 6m ago' },
      { id: 'ob_006_h3', text: 'Queued as high priority', timestampLabel: '1h ago' },
    ],
  },
  {
    id: 'rp_001',
    type: 'report',
    subType: 'fake_profile',
    status: 'new',
    priority: 'high',
    createdAtLabel: '3m ago',
    createdAtTs: 1760572620000,
    title: 'Reported photo in feed card',
    preview: 'Reported for explicit content by another user.',
    user: {
      id: 'u_2001',
      name: 'Daniel K',
      username: '@danielk',
      avatar: 'https://i.pravatar.cc/150?img=21',
      age: 29,
      zodiac: 'Leo',
      city: 'Budapest',
      premium: true,
    },
    reportReason: 'Fake profile',
    reportComment: 'Photos look AI-generated and profile details feel copied.',
    reportedBy: '@mila23',
    reporter: {
      id: 'u_9101',
      name: 'Mila Romanova',
      username: '@mila23',
      avatar: 'https://i.pravatar.cc/150?img=41',
      age: 27,
      city: 'Budapest',
      gender: 'Female',
    },
    contentMediaUrl: 'https://picsum.photos/seed/rp001/720/720',
    tags: ['NSFW risk'],
    history: [
      { id: 'rp_001_h1', text: 'Report created from profile viewer', timestampLabel: '4m ago' },
      { id: 'rp_001_h2', text: 'Auto-priority set to high', timestampLabel: '3m ago' },
    ],
  },
  {
    id: 'rp_002',
    type: 'report',
    subType: 'scammer',
    status: 'in_review',
    priority: 'med',
    createdAtLabel: '12m ago',
    createdAtTs: 1760572080000,
    title: 'Bio reported for external links',
    preview: 'Bio likely contains promotional outbound links.',
    user: {
      id: 'u_2002',
      name: 'Ira K',
      username: '@ira_k',
      avatar: 'https://i.pravatar.cc/150?img=22',
      age: 26,
      zodiac: 'Gemini',
      city: 'Lisbon',
      premium: false,
    },
    reportReason: 'Scammer',
    reportComment: 'Keeps sending promo links and asks to move to external chat immediately.',
    reportedBy: '@roman_88',
    reporter: {
      id: 'u_9102',
      name: 'Roman Vasilev',
      username: '@roman_88',
      avatar: 'https://i.pravatar.cc/150?img=42',
      age: 31,
      city: 'Lisbon',
      gender: 'Male',
    },
    contentText: 'DM me on insta and check my channel: t.me/fastmoneygroup',
    tags: ['Link detected'],
    history: [
      { id: 'rp_002_h1', text: 'Keyword detector found URL pattern', timestampLabel: '14m ago' },
      { id: 'rp_002_h2', text: 'Moderator opened case', timestampLabel: '12m ago' },
      { id: 'rp_002_h3', text: 'Pending action', timestampLabel: '11m ago' },
    ],
  },
  {
    id: 'rp_003',
    type: 'report',
    subType: 'other',
    status: 'in_review',
    priority: 'high',
    createdAtLabel: '26m ago',
    createdAtTs: 1760571240000,
    title: 'Direct message reported for harassment',
    preview: 'Reported conversation includes repeated insults.',
    user: {
      id: 'u_2003',
      name: 'Anton L',
      username: '@antonl',
      avatar: 'https://i.pravatar.cc/150?img=23',
      age: 31,
      zodiac: 'Aries',
      city: 'Warsaw',
      premium: true,
    },
    reportReason: 'Other',
    reportComment: 'Aggressive behavior in chat, repeated insults after match.',
    reportedBy: '@irina_m',
    reporter: {
      id: 'u_9103',
      name: 'Irina M',
      username: '@irina_m',
      avatar: 'https://i.pravatar.cc/150?img=43',
      age: 28,
      city: 'Warsaw',
      gender: 'Female',
    },
    contentText: 'You are pathetic, reply now or I will spam your account.',
    history: [
      { id: 'rp_003_h1', text: 'Report received from chat screen', timestampLabel: '30m ago' },
      { id: 'rp_003_h2', text: 'Toxicity score above threshold', timestampLabel: '28m ago' },
      { id: 'rp_003_h3', text: 'Sent to senior moderator', timestampLabel: '26m ago' },
    ],
  },
  {
    id: 'rp_004',
    type: 'report',
    subType: 'under_18',
    status: 'new',
    priority: 'high',
    createdAtLabel: '42m ago',
    createdAtTs: 1760570280000,
    title: 'Photo reported for possible underage subject',
    preview: 'Reporter claims profile uses school-age photos.',
    user: {
      id: 'u_2004',
      name: 'Mariya O',
      username: '@mariya_o',
      avatar: 'https://i.pravatar.cc/150?img=24',
      age: 20,
      zodiac: 'Pisces',
      city: 'Bratislava',
      premium: false,
    },
    reportReason: 'Under 18',
    reportComment: 'Profile claims to be 20 but photos and text look much younger.',
    reportedBy: '@wolf_17',
    reporter: {
      id: 'u_9104',
      name: 'Wolf K',
      username: '@wolf_17',
      avatar: 'https://i.pravatar.cc/150?img=44',
      age: 24,
      city: 'Bratislava',
      gender: 'Male',
    },
    contentMediaUrl: 'https://picsum.photos/seed/rp004/720/720',
    tags: ['Underage suspicion'],
    history: [
      { id: 'rp_004_h1', text: 'Report submitted with screenshot', timestampLabel: '45m ago' },
      { id: 'rp_004_h2', text: 'Identity confidence check requested', timestampLabel: '43m ago' },
      { id: 'rp_004_h3', text: 'Waiting for moderator verification', timestampLabel: '42m ago' },
    ],
  },
  {
    id: 'rp_005',
    type: 'report',
    subType: 'scammer',
    status: 'new',
    priority: 'med',
    createdAtLabel: '52m ago',
    createdAtTs: 1760569680000,
    title: 'Message reported for spam links',
    preview: 'Conversation contains suspicious referral links.',
    user: {
      id: 'u_2005',
      name: 'Sergey P',
      username: '@sergp',
      avatar: 'https://i.pravatar.cc/150?img=25',
      age: 34,
      zodiac: 'Aquarius',
      city: 'Krakow',
      premium: true,
    },
    reportReason: 'Scammer',
    reportComment: 'Promises easy money, pushes suspicious links in the first messages.',
    reportedBy: '@anya_t',
    reporter: {
      id: 'u_9105',
      name: 'Anya T',
      username: '@anya_t',
      avatar: 'https://i.pravatar.cc/150?img=45',
      age: 26,
      city: 'Krakow',
      gender: 'Female',
    },
    contentText: 'Join now: https://short.example/earn-fast and get bonus.',
    tags: ['Link detected'],
    history: [
      { id: 'rp_005_h1', text: 'Auto detector flagged suspicious URL', timestampLabel: '55m ago' },
      { id: 'rp_005_h2', text: 'Case created in unified inbox', timestampLabel: '52m ago' },
    ],
  },
  {
    id: 'rp_006',
    type: 'report',
    subType: 'fake_profile',
    status: 'in_review',
    priority: 'low',
    createdAtLabel: '2h ago',
    createdAtTs: 1760565600000,
    title: 'Bio reported for off-topic promotion',
    preview: 'Promotion text previously removed, case closed.',
    user: {
      id: 'u_2006',
      name: 'Nikita R',
      username: '@nik_r',
      avatar: 'https://i.pravatar.cc/150?img=26',
      age: 27,
      zodiac: 'Sagittarius',
      city: 'Prague',
      premium: false,
    },
    reportReason: 'Fake profile',
    reportComment: 'Bio is generic ad text and photo set seems stock.',
    reportedBy: '@mia_x',
    reporter: {
      id: 'u_9106',
      name: 'Mia Xu',
      username: '@mia_x',
      avatar: 'https://i.pravatar.cc/150?img=46',
      age: 25,
      city: 'Prague',
      gender: 'Female',
    },
    contentText: 'Daily crypto tips in my channel. Subscribe for profit.',
    history: [
      { id: 'rp_006_h1', text: 'Bio warning issued to user', timestampLabel: '2h 20m ago' },
      { id: 'rp_006_h2', text: 'Content edited by user', timestampLabel: '2h 8m ago' },
      { id: 'rp_006_h3', text: 'Case marked as done', timestampLabel: '2h ago' },
    ],
  },
  {
    id: 'sp_001',
    type: 'support',
    subType: 'payments',
    status: 'in_review',
    priority: 'high',
    createdAtLabel: '18m ago',
    createdAtTs: 1760571720000,
    title: 'Payment charged but premium not activated',
    preview: 'Telegram support bot forwarded billing complaint.',
    user: {
      id: 'u_3001',
      name: 'Olga B',
      username: '@olga_b',
      avatar: 'https://i.pravatar.cc/150?img=31',
      age: 29,
      zodiac: 'Cancer',
      city: 'Warsaw',
      premium: false,
    },
    category: 'payments',
    slaLabel: '2h left',
    messages: [
      { id: 'sp_001_m1', from: 'user', text: 'I paid for Premium but my plan is still free.', timestampLabel: '09:41' },
      { id: 'sp_001_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '09:42' },
      { id: 'sp_001_m3', from: 'admin', text: 'Thanks, checking transaction logs now.', timestampLabel: '09:50' },
      { id: 'sp_001_m4', from: 'user', text: 'Card was charged 10 minutes ago.', timestampLabel: '09:52' },
      { id: 'sp_001_m5', from: 'system', text: 'Gateway callback delayed warning triggered.', timestampLabel: '09:54' },
    ],
    attachments: ['receipt_84421.png'],
    history: [
      { id: 'sp_001_h1', text: 'Support ticket created', timestampLabel: '18m ago' },
      { id: 'sp_001_h2', text: 'Assigned to payments queue', timestampLabel: '17m ago' },
      { id: 'sp_001_h3', text: 'Admin joined thread', timestampLabel: '10m ago' },
    ],
  },
  {
    id: 'sp_002',
    type: 'support',
    subType: 'account',
    status: 'new',
    priority: 'med',
    createdAtLabel: '22m ago',
    createdAtTs: 1760571480000,
    title: 'Cannot login after Telegram reconnect',
    preview: 'User gets session expired error on each launch.',
    user: {
      id: 'u_3002',
      name: 'Roma V',
      username: '@romav',
      avatar: 'https://i.pravatar.cc/150?img=32',
      age: 32,
      zodiac: 'Aries',
      city: 'Berlin',
      premium: true,
    },
    category: 'account',
    slaLabel: '12h left',
    messages: [
      { id: 'sp_002_m1', from: 'user', text: 'App logs me out every time I reopen Telegram.', timestampLabel: '08:11' },
      { id: 'sp_002_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '08:12' },
      { id: 'sp_002_m3', from: 'user', text: 'I already reinstalled Telegram.', timestampLabel: '08:14' },
      { id: 'sp_002_m4', from: 'admin', text: 'We are checking token refresh logs.', timestampLabel: '08:25' },
    ],
    history: [
      { id: 'sp_002_h1', text: 'Ticket created', timestampLabel: '22m ago' },
      { id: 'sp_002_h2', text: 'Queued in account support', timestampLabel: '20m ago' },
    ],
  },
  {
    id: 'sp_003',
    type: 'support',
    subType: 'bugs',
    status: 'waiting_user',
    priority: 'med',
    createdAtLabel: '34m ago',
    createdAtTs: 1760570760000,
    title: 'Likes screen freezes on scroll',
    preview: 'Client reports UI freeze in Telegram webview.',
    user: {
      id: 'u_3003',
      name: 'Daria N',
      username: '@darian',
      avatar: 'https://i.pravatar.cc/150?img=33',
      age: 23,
      zodiac: 'Leo',
      city: 'Riga',
      premium: false,
    },
    category: 'bugs',
    slaLabel: '8h left',
    messages: [
      { id: 'sp_003_m1', from: 'user', text: 'Likes tab freezes after 2-3 swipes.', timestampLabel: '07:50' },
      { id: 'sp_003_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '07:51' },
      { id: 'sp_003_m3', from: 'admin', text: 'Could you share device model and app version?', timestampLabel: '08:02' },
      { id: 'sp_003_m4', from: 'user', text: 'iPhone 13, Telegram 11.8, iOS 18.', timestampLabel: '08:05' },
      { id: 'sp_003_m5', from: 'system', text: 'Crash fingerprint attached to ticket.', timestampLabel: '08:06' },
    ],
    attachments: ['crash-log-2026-02-15.txt'],
    history: [
      { id: 'sp_003_h1', text: 'Ticket created', timestampLabel: '34m ago' },
      { id: 'sp_003_h2', text: 'Bug triage requested extra data', timestampLabel: '30m ago' },
      { id: 'sp_003_h3', text: 'Waiting for user response', timestampLabel: '28m ago' },
    ],
  },
  {
    id: 'sp_004',
    type: 'support',
    subType: 'safety',
    status: 'in_review',
    priority: 'high',
    createdAtLabel: '47m ago',
    createdAtTs: 1760569980000,
    title: 'Stalking concern after profile match',
    preview: 'User asks for urgent safety action and block audit.',
    user: {
      id: 'u_3004',
      name: 'Alina T',
      username: '@alina_t',
      avatar: 'https://i.pravatar.cc/150?img=34',
      age: 27,
      zodiac: 'Virgo',
      city: 'Vienna',
      premium: true,
    },
    category: 'safety',
    slaLabel: '1h left',
    messages: [
      { id: 'sp_004_m1', from: 'user', text: 'Someone I blocked keeps creating new profiles.', timestampLabel: '06:44' },
      { id: 'sp_004_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '06:44' },
      { id: 'sp_004_m3', from: 'admin', text: 'We can send this to the trust and safety team now.', timestampLabel: '06:49' },
      { id: 'sp_004_m4', from: 'user', text: 'Please do, I feel unsafe.', timestampLabel: '06:50' },
      { id: 'sp_004_m5', from: 'system', text: 'Emergency policy checklist attached.', timestampLabel: '06:52' },
      { id: 'sp_004_m6', from: 'admin', text: 'Escalation submitted, we will update you shortly.', timestampLabel: '06:55' },
    ],
    tags: ['Underage suspicion'],
    history: [
      { id: 'sp_004_h1', text: 'Safety ticket created', timestampLabel: '47m ago' },
      { id: 'sp_004_h2', text: 'Sent to trust and safety', timestampLabel: '45m ago' },
      { id: 'sp_004_h3', text: 'Temporary shadow block applied', timestampLabel: '43m ago' },
      { id: 'sp_004_h4', text: 'Awaiting final review', timestampLabel: '39m ago' },
    ],
  },
  {
    id: 'sp_005',
    type: 'support',
    subType: 'other',
    status: 'new',
    priority: 'low',
    createdAtLabel: '58m ago',
    createdAtTs: 1760569320000,
    title: 'Question about profile badge meaning',
    preview: 'General support question from Telegram bot.',
    user: {
      id: 'u_3005',
      name: 'Pavel I',
      username: '@pavel_i',
      avatar: 'https://i.pravatar.cc/150?img=35',
      age: 35,
      zodiac: 'Sagittarius',
      city: 'Brno',
      premium: false,
    },
    category: 'other',
    slaLabel: '20h left',
    messages: [
      { id: 'sp_005_m1', from: 'user', text: 'What does the purple shield badge mean?', timestampLabel: '06:12' },
      { id: 'sp_005_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '06:12' },
      { id: 'sp_005_m3', from: 'admin', text: 'It marks verified profiles after additional checks.', timestampLabel: '06:21' },
      { id: 'sp_005_m4', from: 'user', text: 'Thanks, understood.', timestampLabel: '06:23' },
    ],
    history: [
      { id: 'sp_005_h1', text: 'Ticket created', timestampLabel: '58m ago' },
      { id: 'sp_005_h2', text: 'Queued in general support', timestampLabel: '57m ago' },
    ],
  },
  {
    id: 'sp_006',
    type: 'support',
    subType: 'payments',
    status: 'waiting_user',
    priority: 'med',
    createdAtLabel: '1h ago',
    createdAtTs: 1760569198000,
    title: 'Refund request after accidental renewal',
    preview: 'User requests refund under grace period.',
    user: {
      id: 'u_3006',
      name: 'Igor F',
      username: '@igorf',
      avatar: 'https://i.pravatar.cc/150?img=36',
      age: 33,
      zodiac: 'Pisces',
      city: 'Munich',
      premium: true,
    },
    category: 'payments',
    slaLabel: '6h left',
    messages: [
      { id: 'sp_006_m1', from: 'user', text: 'My subscription renewed by mistake, need refund.', timestampLabel: '05:40' },
      { id: 'sp_006_m2', from: 'system', text: 'Ticket imported from Telegram support bot.', timestampLabel: '05:41' },
      { id: 'sp_006_m3', from: 'admin', text: 'Please confirm if you used premium features after renewal.', timestampLabel: '05:50' },
      { id: 'sp_006_m4', from: 'user', text: 'No, I did not use any premium options.', timestampLabel: '05:53' },
      { id: 'sp_006_m5', from: 'system', text: 'Refund policy checklist attached.', timestampLabel: '05:56' },
    ],
    history: [
      { id: 'sp_006_h1', text: 'Ticket created', timestampLabel: '1h 2m ago' },
      { id: 'sp_006_h2', text: 'Refund flow initiated', timestampLabel: '58m ago' },
      { id: 'sp_006_h3', text: 'Waiting for admin final response', timestampLabel: '55m ago' },
    ],
  },
];

const staticNonSupportCases: ModerationCase[] = unifiedCases.filter((caseItem) => caseItem.type !== 'support');
const staticSupportFallbackCases: ModerationCase[] = unifiedCases.filter((caseItem) => caseItem.type === 'support');

function toRelativeTimeLabel(value: Date): string {
  const diffMs = Date.now() - value.getTime();
  if (diffMs < 60_000) {
    return 'just now';
  }
  if (diffMs < 3_600_000) {
    return `${Math.max(1, Math.floor(diffMs / 60_000))}m ago`;
  }
  if (diffMs < 86_400_000) {
    return `${Math.floor(diffMs / 3_600_000)}h ago`;
  }
  return `${Math.floor(diffMs / 86_400_000)}d ago`;
}

function toClockLabel(value: Date): string {
  const hours = String(value.getHours()).padStart(2, '0');
  const minutes = String(value.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}

function parseDateSafe(value: string | undefined): Date | null {
  if (!value) {
    return null;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  return parsed;
}

function supportUsername(value: string): string | undefined {
  const normalized = value.trim();
  if (!normalized) {
    return undefined;
  }
  return normalized.startsWith('@') ? normalized : `@${normalized}`;
}

function mapSupportStatusToModeration(status: SupportConversationStatus): ModerationStatus {
  if (status === 'waiting') {
    return 'waiting_user';
  }
  if (status === 'done') {
    return 'resolved';
  }
  if (status === 'escalated') {
    return 'in_review';
  }
  return status;
}

function mapSupportConversationToCase(conversation: SupportConversationDTO): ModerationCase {
  const createdAt = parseDateSafe(conversation.created_at) ?? new Date();
  const lastMessageAt = parseDateSafe(conversation.last_message_at) ?? createdAt;
  const userName =
    conversation.display_name.trim() ||
    `${conversation.first_name.trim()} ${conversation.last_name.trim()}`.trim() ||
    supportUsername(conversation.username) ||
    `User ${conversation.user_tg_id}`;

  return {
    id: `sp_live_${conversation.id}`,
    type: 'support',
    subType: conversation.category,
    category: conversation.category,
    status: mapSupportStatusToModeration(conversation.status),
    priority: conversation.priority,
    createdAtLabel: toRelativeTimeLabel(createdAt),
    createdAtTs: createdAt.getTime(),
    title: conversation.title.trim() || `Support ticket #${conversation.id}`,
    preview: conversation.preview.trim() || 'Support ticket from Telegram support bot.',
    user: {
      id: conversation.app_user_id ? `u_${conversation.app_user_id}` : `tg_${conversation.user_tg_id}`,
      name: userName,
      username: supportUsername(conversation.app_username) ?? supportUsername(conversation.username),
      age: conversation.age,
      zodiac: conversation.zodiac.trim() || undefined,
      city: conversation.city.trim() || undefined,
      premium: conversation.premium,
    },
    history: [
      {
        id: `sp_live_${conversation.id}_last`,
        text: `Last message ${toRelativeTimeLabel(lastMessageAt)}`,
        timestampLabel: toRelativeTimeLabel(lastMessageAt),
      },
    ],
  };
}

function mapSupportMessageToUI(message: SupportMessageDTO): SupportMessage {
  const createdAt = parseDateSafe(message.created_at) ?? new Date();
  const from: SupportMsgFrom =
    message.from === 'admin' || message.from === 'system' || message.from === 'user'
      ? message.from
      : 'system';

  return {
    id: `spm_${message.id}`,
    from,
    text: message.text,
    timestampLabel: toClockLabel(createdAt),
  };
}

function supportConversationIDFromCaseID(caseID: string): number | null {
  const match = caseID.match(/^sp_live_(\d+)$/);
  if (!match) {
    return null;
  }
  const id = Number(match[1]);
  if (!Number.isFinite(id) || id <= 0) {
    return null;
  }
  return Math.trunc(id);
}

type ModerationViewType = 'all' | ModerationCaseType;

type FilterCasesParams = {
  cases: ModerationCase[];
  selectedType: ModerationViewType;
  selectedSubType: string;
  query: string;
};

const typeTabs: Array<{ value: ModerationViewType; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'onboarding', label: 'Onboarding' },
  { value: 'report', label: 'Reports' },
  { value: 'support', label: 'Support' },
];

const reportSubTypeLabels: Record<ReportSubType, string> = {
  fake_profile: 'Fake profile',
  scammer: 'Scammer',
  under_18: 'Under 18',
  other: 'Other',
};

const onboardingSubTypeLabels: Record<OnboardingSubType, string> = {
  new_user: 'New user',
  profile_update: 'Profile update',
};

const supportSubTypeLabels: Record<SupportSubType, string> = {
  payments: 'Payments',
  account: 'Account',
  bugs: 'Bugs',
  safety: 'Safety',
  other: 'Other',
};

const subTypeTabs: Record<Exclude<ModerationViewType, 'all'>, Array<{ value: string; label: string }>> = {
  report: [
    { value: 'fake_profile', label: reportSubTypeLabels.fake_profile },
    { value: 'scammer', label: reportSubTypeLabels.scammer },
    { value: 'under_18', label: reportSubTypeLabels.under_18 },
    { value: 'other', label: reportSubTypeLabels.other },
  ],
  onboarding: [
    { value: 'new_user', label: onboardingSubTypeLabels.new_user },
    { value: 'profile_update', label: onboardingSubTypeLabels.profile_update },
  ],
  support: [
    { value: 'payments', label: supportSubTypeLabels.payments },
    { value: 'account', label: supportSubTypeLabels.account },
    { value: 'bugs', label: supportSubTypeLabels.bugs },
    { value: 'safety', label: supportSubTypeLabels.safety },
    { value: 'other', label: supportSubTypeLabels.other },
  ],
};

const statusLabels: Record<ModerationStatus, string> = {
  new: 'New',
  in_review: 'In review',
  changes_requested: 'Changes requested',
  approved: 'Approved',
  rejected: 'Rejected',
  waiting_user: 'Waiting user',
  resolved: 'Resolved',
};

const statusBadgeClassByStatus: Record<ModerationStatus, string> = {
  new: 'bg-[rgba(245,247,255,0.08)] border-[rgba(245,247,255,0.16)] text-[#D5DCEE]',
  in_review: 'bg-[rgba(123,97,255,0.16)] border-[rgba(123,97,255,0.28)] text-[#E7E3FF]',
  changes_requested: 'bg-[rgba(255,209,102,0.16)] border-[rgba(255,209,102,0.28)] text-[#FFE2A6]',
  approved: 'bg-[rgba(45,212,168,0.16)] border-[rgba(45,212,168,0.28)] text-[#9EF2DA]',
  rejected: 'bg-[rgba(255,107,107,0.16)] border-[rgba(255,107,107,0.28)] text-[#FFB8B8]',
  waiting_user: 'bg-[rgba(255,209,102,0.16)] border-[rgba(255,209,102,0.28)] text-[#FFE2A6]',
  resolved: 'bg-[rgba(45,212,168,0.16)] border-[rgba(45,212,168,0.28)] text-[#9EF2DA]',
};

function getSubTypeLabel(caseItem: ModerationCase): string {
  if (caseItem.type === 'report') {
    return reportSubTypeLabels[caseItem.subType as ReportSubType];
  }
  if (caseItem.type === 'onboarding') {
    return onboardingSubTypeLabels[caseItem.subType as OnboardingSubType];
  }
  return supportSubTypeLabels[caseItem.subType as SupportSubType];
}

function getCountsByType(cases: ModerationCase[]) {
  return cases.reduce(
    (acc, caseItem) => {
      acc[caseItem.type] += 1;
      return acc;
    },
    { onboarding: 0, report: 0, support: 0 }
  );
}

function filterCases({ cases, selectedType, selectedSubType, query }: FilterCasesParams) {
  const normalizedQuery = query.trim().toLowerCase();

  return [...cases]
    .filter((caseItem) => selectedType === 'all' || caseItem.type === selectedType)
    .filter(
      (caseItem) =>
        !selectedSubType ||
        selectedType === 'all' ||
        caseItem.subType === selectedSubType
    )
    .filter((caseItem) => {
      if (!normalizedQuery) {
        return true;
      }

      return [
        caseItem.id,
        caseItem.title,
        caseItem.user.id,
        caseItem.user.username ?? '',
        caseItem.user.name,
      ].some((field) => field.toLowerCase().includes(normalizedQuery));
    })
    .sort((a, b) => a.createdAtTs - b.createdAtTs);
}

function formatDateToEuropean(value: string): string {
  if (/^\d{2}\.\d{2}\.\d{4}$/.test(value)) {
    return value;
  }

  const isoMatch = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (isoMatch) {
    return `${isoMatch[3]}.${isoMatch[2]}.${isoMatch[1]}`;
  }

  const parsedDate = new Date(value);
  if (!Number.isNaN(parsedDate.getTime())) {
    const day = String(parsedDate.getDate()).padStart(2, '0');
    const month = String(parsedDate.getMonth() + 1).padStart(2, '0');
    const year = String(parsedDate.getFullYear());
    return `${day}.${month}.${year}`;
  }

  return value;
}

type ProfilePresence = 'online' | 'away' | 'offline';

const profileStatusConfig: Record<
  ProfilePresence,
  { dot: string; bg: string; text: string; label: string }
> = {
  online: {
    dot: 'bg-[#2DD4A8]',
    bg: 'bg-[rgba(45,212,168,0.15)]',
    text: 'text-[#2DD4A8]',
    label: 'online',
  },
  away: {
    dot: 'bg-[#FFD166]',
    bg: 'bg-[rgba(255,209,102,0.18)]',
    text: 'text-[#FFD166]',
    label: 'away',
  },
  offline: {
    dot: 'bg-[#A7B1C8]',
    bg: 'bg-[rgba(167,177,200,0.18)]',
    text: 'text-[#A7B1C8]',
    label: 'offline',
  },
};

function deriveProfilePresence(lastActiveLabel?: string): ProfilePresence {
  if (!lastActiveLabel) {
    return 'online';
  }
  const normalized = lastActiveLabel.toLowerCase();
  if (normalized.includes('just now')) {
    return 'online';
  }
  const minutesMatch = normalized.match(/(\d+)\s*m/);
  if (minutesMatch) {
    const minutes = Number(minutesMatch[1]);
    if (Number.isFinite(minutes) && minutes <= 15) {
      return 'online';
    }
    if (Number.isFinite(minutes) && minutes <= 180) {
      return 'away';
    }
    return 'offline';
  }
  const hoursMatch = normalized.match(/(\d+)\s*h/);
  if (hoursMatch) {
    const hours = Number(hoursMatch[1]);
    if (Number.isFinite(hours) && hours <= 2) {
      return 'away';
    }
    return 'offline';
  }
  return 'offline';
}

function resolveTelegramId(value: string): string {
  const explicitMatch = value.match(/^tg_(\d+)$/);
  if (explicitMatch) {
    return explicitMatch[1];
  }

  const digits = value.replace(/\D/g, '');
  if (digits) {
    return String(700000000 + Number(digits));
  }

  return 'unknown';
}

function formatJoinedSummary(value: string | undefined, fallbackTs: number): string {
  const sourceDate = value ? new Date(value) : new Date(fallbackTs);
  if (Number.isNaN(sourceDate.getTime())) {
    return 'N/A';
  }

  const startOfToday = new Date();
  startOfToday.setHours(0, 0, 0, 0);
  const startOfJoined = new Date(
    sourceDate.getFullYear(),
    sourceDate.getMonth(),
    sourceDate.getDate(),
  );

  const days = Math.max(
    0,
    Math.floor((startOfToday.getTime() - startOfJoined.getTime()) / (1000 * 60 * 60 * 24)),
  );
  return `${days} days • ${formatDateToEuropean(sourceDate.toISOString().slice(0, 10))}`;
}

function seedMetric(seed: string, min: number, max: number): number {
  const hash = Array.from(seed).reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return min + (hash % (max - min + 1));
}

function createInitialProfileLimits(userId: string): ProfileLimitsState {
  return {
    dailySwipesTotal: 120,
    dailySwipesRemaining: seedMetric(`${userId}_swipes`, 20, 120),
    superLikesTotal: 5,
    superLikesRemaining: seedMetric(`${userId}_super`, 0, 5),
    boostsTotal: 3,
    boostsRemaining: seedMetric(`${userId}_boost`, 0, 3),
  };
}

function resolveProfileInteractionProfiles(
  user: UserSummary,
  interaction: ProfileInteractionType,
): UserSummary[] {
  const names = [
    'Emma Wilson',
    'James Chen',
    'Sofia Rodriguez',
    'Michael Park',
    'Isabella Martinez',
    'Lucas Brown',
    'Olivia Stone',
    'Noah Carter',
  ];

  const handleBase = ['@emma_w', '@jamesc', '@sofia_r', '@mpark', '@bella_m', '@lucas_b', '@olivia_s', '@ncarter'];
  const seed = `${user.id}_${interaction}`;
  const count = seedMetric(seed, 4, 8);
  const offset = seedMetric(`${seed}_offset`, 0, names.length - 1);

  return Array.from({ length: count }).map((_, index) => {
    const idx = (offset + index) % names.length;
    const age = seedMetric(`${seed}_${index}_age`, 20, 39);
    const idNumber = 500000 + seedMetric(`${seed}_${index}_id`, 1000, 9999);

    return {
      id: `u_${idNumber}`,
      name: names[idx],
      username: handleBase[idx],
      avatar: `https://i.pravatar.cc/150?img=${60 + ((idx + index) % 20)}`,
      age,
      city: user.city ?? 'Unknown',
      gender: index % 2 === 0 ? 'Female' : 'Male',
      premium: index % 3 === 0,
      phone: `+1 917 55${String(seedMetric(`${seed}_${index}_ph`, 100, 999)).padStart(3, '0')} ${String(seedMetric(`${seed}_${index}_ph2`, 1000, 9999)).padStart(4, '0')}`,
      bio: `Interaction profile from ${interaction.replace('_', ' ')} list.`,
      interests: ['Travel', 'Music', 'Gym', 'Cinema', 'Coffee'].slice(
        0,
        seedMetric(`${seed}_${index}_int`, 3, 5),
      ),
      joinedAt: new Date(
        Date.now() - seedMetric(`${seed}_${index}_joined`, 10, 900) * 24 * 60 * 60 * 1000,
      )
        .toISOString()
        .slice(0, 10),
      lastActiveLabel: `${seedMetric(`${seed}_${index}_last`, 1, 180)}m ago`,
      heightCm: seedMetric(`${seed}_${index}_height`, 155, 195),
      eyeColor: ['Hazel', 'Blue', 'Brown', 'Green'][seedMetric(`${seed}_${index}_eye`, 0, 3)],
      photos: [
        `https://picsum.photos/seed/${seed}_${index}_a/600/800`,
        `https://picsum.photos/seed/${seed}_${index}_b/600/800`,
        `https://picsum.photos/seed/${seed}_${index}_c/600/800`,
      ],
    };
  });
}

let moderationCasesSnapshot: ModerationCase[] = unifiedCases;
const moderationCasesListeners = new Set<(cases: ModerationCase[]) => void>();

function syncModerationCasesSnapshot(cases: ModerationCase[]) {
  moderationCasesSnapshot = cases;
  moderationCasesListeners.forEach((listener) => listener(cases));
}

function useModerationCasesSnapshot() {
  const [cases, setCases] = useState<ModerationCase[]>(moderationCasesSnapshot);

  useEffect(() => {
    moderationCasesListeners.add(setCases);
    return () => {
      moderationCasesListeners.delete(setCases);
    };
  }, []);

  return cases;
}

export function useModerationPendingCount() {
  return useModerationCasesSnapshot().length;
}

type ModerationChangeLogEntry = {
  id: string;
  createdAtISO: string;
  type: 'action' | 'reply';
  caseId: string;
  caseType: ModerationCaseType;
  subType: string;
  targetUserId: string;
  actorId: string;
  payload: Record<string, string>;
};

const MODERATION_CHANGE_LOG_KEY = 'moderation_change_log_v1';

function appendModerationChangeLog(entry: Omit<ModerationChangeLogEntry, 'id' | 'createdAtISO'>) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    const raw = window.localStorage.getItem(MODERATION_CHANGE_LOG_KEY);
    const current: ModerationChangeLogEntry[] = raw ? (JSON.parse(raw) as ModerationChangeLogEntry[]) : [];
    const nextEntry: ModerationChangeLogEntry = {
      ...entry,
      id: `mod_log_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`,
      createdAtISO: new Date().toISOString(),
    };
    window.localStorage.setItem(MODERATION_CHANGE_LOG_KEY, JSON.stringify([nextEntry, ...current].slice(0, 500)));
  } catch (error) {
    console.warn('Failed to persist moderation change log', error);
  }
}

function ModerationHeaderSlot({ cases }: { cases: ModerationCase[] }) {
  const counts = getCountsByType(cases);
  const totalPending = cases.length;

  return (
    <div className="flex items-center gap-2 shrink-0 whitespace-nowrap pl-3 sm:pl-0">
      <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[rgba(255,107,107,0.1)] border border-[rgba(255,107,107,0.2)]">
        <AlertTriangle className="w-4 h-4 text-[#FF6B6B]" />
        <span className="text-sm text-[#FF6B6B]">{totalPending} pending</span>
      </div>
      <div className="hidden lg:flex items-center gap-2">
        <span className="px-2 py-1 rounded-lg text-xs bg-[rgba(123,97,255,0.12)] border border-[rgba(123,97,255,0.2)] text-[#B7A9FF]">
          Onboarding {counts.onboarding}
        </span>
        <span className="px-2 py-1 rounded-lg text-xs bg-[rgba(123,97,255,0.12)] border border-[rgba(123,97,255,0.2)] text-[#B7A9FF]">
          Reports {counts.report}
        </span>
        <span className="px-2 py-1 rounded-lg text-xs bg-[rgba(123,97,255,0.12)] border border-[rgba(123,97,255,0.2)] text-[#B7A9FF]">
          Support {counts.support}
        </span>
      </div>
    </div>
  );
}

/** Badge for TopBar when on Moderation page */
export function ModerationPendingBadge() {
  const cases = useModerationCasesSnapshot();
  return <ModerationHeaderSlot cases={cases} />;
}

type ModerationAction =
  | 'approve'
  | 'reject'
  | 'request_changes'
  | 'dismiss'
  | 'warn'
  | 'ban'
  | 'resolve'
  | 'request_info';

type ReportDecision = 'warn' | 'ban';

type ActionTemplate = {
  id: string;
  title: string;
  message: string;
};

const actionPermissions = {
  approve: ADMIN_PERMISSIONS.approve_profiles,
  reject: ADMIN_PERMISSIONS.reject_profiles,
  request_changes: ADMIN_PERMISSIONS.moderate_profiles,
  dismiss: ADMIN_PERMISSIONS.reject_profiles,
  warn: ADMIN_PERMISSIONS.moderate_profiles,
  ban: ADMIN_PERMISSIONS.ban_users,
  resolve: ADMIN_PERMISSIONS.moderate_profiles,
  request_info: ADMIN_PERMISSIONS.moderate_profiles,
} as const;

const actionMeta: Record<ModerationAction, { label: string; icon: ReactNode; tone: 'primary' | 'danger' | 'warn' | 'neutral' }> = {
  approve: { label: 'Approve', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  reject: { label: 'Reject', icon: <X className="w-4 h-4" />, tone: 'danger' },
  request_changes: { label: 'Request changes', icon: <MessageSquare className="w-4 h-4" />, tone: 'neutral' },
  dismiss: { label: 'Dismiss', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  warn: { label: 'Warn', icon: <AlertTriangle className="w-4 h-4" />, tone: 'warn' },
  ban: { label: 'Ban', icon: <Ban className="w-4 h-4" />, tone: 'danger' },
  resolve: { label: 'Resolve', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  request_info: { label: 'Request info', icon: <MessageSquare className="w-4 h-4" />, tone: 'neutral' },
};

const actionsByType: Record<ModerationCaseType, ModerationAction[]> = {
  onboarding: ['approve', 'reject', 'request_changes'],
  report: ['dismiss', 'warn', 'ban'],
  support: ['resolve', 'request_info'],
};

const onboardingRejectTemplates: ActionTemplate[] = [
  {
    id: 'ob_reject_1',
    title: 'Profile policy mismatch',
    message: 'Your profile was rejected due to policy mismatch. Please update your details and submit again.',
  },
  {
    id: 'ob_reject_2',
    title: 'Identity verification failed',
    message: 'We could not verify your profile information. Please re-submit with accurate and complete details.',
  },
  {
    id: 'ob_reject_3',
    title: 'Content safety violation',
    message: 'Your profile content violates safety rules. Please correct it and submit a new profile review.',
  },
];

const onboardingRequestChangesTemplates: ActionTemplate[] = [
  {
    id: 'ob_changes_1',
    title: 'Update photos',
    message: 'Please update your profile photos to meet our quality and safety guidelines.',
  },
  {
    id: 'ob_changes_2',
    title: 'Fix bio text',
    message: 'Please edit your bio to remove links/promotional text and resubmit for moderation.',
  },
  {
    id: 'ob_changes_3',
    title: 'Complete profile fields',
    message: 'Please complete or correct required profile fields and submit again.',
  },
];

const reportWarnTemplates: ActionTemplate[] = [
  {
    id: 'rp_warn_1',
    title: 'First warning',
    message: 'Your account received a warning due to reported behavior. Further violations may lead to restrictions.',
  },
  {
    id: 'rp_warn_2',
    title: 'Scam warning',
    message: 'Reported scam-like behavior was detected. Any repeated violations can result in account suspension.',
  },
  {
    id: 'rp_warn_3',
    title: 'Profile integrity warning',
    message: 'Your profile was reported for authenticity concerns. Please ensure all information is accurate.',
  },
];

const reportBanTemplates: ActionTemplate[] = [
  {
    id: 'rp_ban_1',
    title: 'Safety ban',
    message: 'Your account has been banned due to severe safety policy violations.',
  },
  {
    id: 'rp_ban_2',
    title: 'Scam ban',
    message: 'Your account has been banned due to confirmed scam activity.',
  },
  {
    id: 'rp_ban_3',
    title: 'Underage policy ban',
    message: 'Your account has been banned due to underage policy violations.',
  },
];

function FiltersBar({
  counts,
  selectedType,
  onSelectType,
  selectedSubType,
  onSelectSubType,
  query,
  onChangeQuery,
}: {
  counts: ReturnType<typeof getCountsByType>;
  selectedType: ModerationViewType;
  onSelectType: (type: ModerationViewType) => void;
  selectedSubType: string;
  onSelectSubType: (subType: string) => void;
  query: string;
  onChangeQuery: (value: string) => void;
}) {
  const activeSubTabs = selectedType === 'all' ? [] : subTypeTabs[selectedType];

  return (
    <div className="p-3 border-b border-[rgba(123,97,255,0.12)] space-y-2">
      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-[#A7B1C8]" />
        <div className="flex flex-wrap gap-1">
          {typeTabs.map((tab) => {
            const tabCount =
              tab.value === 'all'
                ? counts.onboarding + counts.report + counts.support
                : counts[tab.value];
            const isActive = selectedType === tab.value;
            const showCount = isActive;
            const countLabel =
              tab.value === 'all'
                ? counts.onboarding + counts.report + counts.support
                : tabCount;

            return (
              <button
                key={tab.value}
                onClick={() => onSelectType(tab.value)}
                className={cn(
                  'px-2 py-1 rounded text-xs font-medium transition-colors',
                  isActive
                    ? 'bg-[rgba(123,97,255,0.2)] text-[#7B61FF]'
                    : 'text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)]'
                )}
              >
                <span className="flex items-center gap-2">
                  <span>{tab.label}</span>
                  {showCount && <span className="text-[#A7B1C8]">({countLabel})</span>}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {activeSubTabs.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {activeSubTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => onSelectSubType(tab.value)}
              className={cn(
                'px-2 py-1 text-xs font-medium transition-colors',
                selectedSubType === tab.value
                  ? 'rounded bg-[rgba(123,97,255,0.16)] text-[#B7A9FF] border border-[rgba(123,97,255,0.35)]'
                  : 'text-[#A7B1C8] hover:text-[#F5F7FF]'
              )}
            >
              <span className={cn(selectedSubType === tab.value && 'border-b border-[rgba(183,169,255,0.65)] pb-0.5')}>
                {tab.label}
              </span>
            </button>
          ))}
        </div>
      )}

      <input
        value={query}
        onChange={(event) => onChangeQuery(event.target.value)}
        placeholder="Search by case ID, title, or user..."
        className="w-full px-3 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.55)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[#A7B1C8] focus:outline-none focus:border-[rgba(123,97,255,0.4)]"
      />
    </div>
  );
}

function QueueList({
  cases,
  selectedCaseId,
  onSelectCase,
}: {
  cases: ModerationCase[];
  selectedCaseId: string | null;
  onSelectCase: (caseId: string) => void;
}) {
  return (
    <div className="flex-1 overflow-y-auto scrollbar-thin">
      {cases.length === 0 ? (
        <div className="p-6 text-center text-sm text-[#A7B1C8]">No cases matched current filters</div>
      ) : (
        cases.map((caseItem) => {
          const isOnboarding = caseItem.type === 'onboarding';

          return (
            <button
              key={caseItem.id}
              onClick={() => onSelectCase(caseItem.id)}
              className={cn(
                'w-full p-4 text-left border-b border-[rgba(123,97,255,0.08)] transition-colors',
                selectedCaseId === caseItem.id
                  ? 'bg-[rgba(123,97,255,0.1)]'
                  : 'hover:bg-[rgba(123,97,255,0.05)]'
              )}
            >
              <div className="flex items-start gap-3">
                {caseItem.user.avatar ? (
                  <img
                    src={caseItem.user.avatar}
                    alt={caseItem.user.name}
                    className="w-12 h-12 rounded-lg object-cover border border-[rgba(123,97,255,0.2)]"
                  />
                ) : (
                  <div className="w-12 h-12 rounded-lg bg-[rgba(123,97,255,0.1)] flex items-center justify-center text-sm text-[#B7A9FF]">
                    {caseItem.user.name.slice(0, 1).toUpperCase()}
                  </div>
                )}

                <div className="flex-1 min-w-0 space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm font-medium text-[#F5F7FF] truncate">
                      {isOnboarding ? 'Onboarding review' : caseItem.title}
                    </p>
                    <span className="text-xs text-[#A7B1C8] shrink-0">{caseItem.createdAtLabel}</span>
                  </div>

                  <p className="text-xs text-[#A7B1C8] line-clamp-2">
                    {isOnboarding ? 'New profile submitted for review.' : caseItem.preview}
                  </p>

                  <div className="flex flex-wrap gap-1.5">
                    {isOnboarding ? (
                      <>
                        <span className="px-3 py-1 rounded-lg text-xs bg-[rgba(123,97,255,0.14)] border border-[rgba(123,97,255,0.25)] text-[#CFC6FF]">
                          ONBOARDING
                        </span>
                        <span
                          className={cn(
                            'px-2 py-1 rounded-lg text-[11px] border',
                            caseItem.id === selectedCaseId
                              ? 'bg-[rgba(123,97,255,0.16)] border-[rgba(123,97,255,0.25)] text-[#E7E3FF]'
                              : 'bg-[rgba(245,247,255,0.06)] border-[rgba(245,247,255,0.10)] text-[#A7B1C8]'
                          )}
                        >
                          {caseItem.id === selectedCaseId ? 'IN REVIEW' : 'NEW'}
                        </span>
                      </>
                    ) : (
                      <>
                        <span
                          className={cn(
                            'px-3 py-1 rounded-lg text-xs border uppercase',
                            caseItem.type === 'report'
                              ? 'bg-[rgba(255,107,107,0.12)] border-[rgba(255,107,107,0.22)] text-[#FFB8B8]'
                              : 'bg-[rgba(76,201,240,0.12)] border-[rgba(76,201,240,0.22)] text-[#B6EFFF]',
                          )}
                        >
                          {caseItem.type}/{getSubTypeLabel(caseItem)}
                        </span>
                        <span
                          className={cn(
                            'px-3 py-1 rounded-lg text-xs border',
                            statusBadgeClassByStatus[caseItem.status],
                          )}
                        >
                          {statusLabels[caseItem.status]}
                        </span>
                      </>
                    )}
                  </div>
                </div>

                <ChevronRight
                  className={cn(
                    'w-4 h-4 text-[#A7B1C8] flex-shrink-0',
                    selectedCaseId === caseItem.id && 'text-[#7B61FF]'
                  )}
                />
              </div>
            </button>
          );
        })
      )}
    </div>
  );
}

function ActionBar({
  caseType,
  onAction,
  canAction,
  reportDecision,
}: {
  caseType: ModerationCaseType;
  onAction: (action: ModerationAction) => void;
  canAction: (action: ModerationAction) => boolean;
  reportDecision: ReportDecision | null;
}) {
  const buttonClassByTone = {
    primary: 'btn-primary',
    danger: 'btn-danger',
    warn:
      'bg-[rgba(255,209,102,0.15)] text-[#FFD166] border border-[rgba(255,209,102,0.25)] hover:bg-[rgba(255,209,102,0.25)]',
    neutral:
      'bg-[rgba(123,97,255,0.15)] text-[#B7A9FF] border border-[rgba(123,97,255,0.25)] hover:bg-[rgba(123,97,255,0.25)]',
  } as const;

  const actions = actionsByType[caseType];

  return (
    <div className="p-4 border-t border-[rgba(123,97,255,0.12)] space-y-2">
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => {
          const meta = actionMeta[action];
          const isWideAction =
            action === 'approve' ||
            action === 'reject' ||
            action === 'dismiss' ||
            action === 'resolve';
          const isReportDecisionButton =
            caseType === 'report' && (action === 'warn' || action === 'ban');
          const isSelectedReportDecision =
            isReportDecisionButton && reportDecision === action;
          const isDismissBlocked =
            caseType === 'report' &&
            action === 'dismiss' &&
            !reportDecision;
          const isDisabled = !canAction(action) || isDismissBlocked;

          return (
            <button
              key={action}
              onClick={() => onAction(action)}
              disabled={isDisabled}
              className={cn(
                isWideAction ? 'flex-1' : '',
                action === 'ban' && 'ml-auto',
                'px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed',
                buttonClassByTone[meta.tone],
                isSelectedReportDecision &&
                  'ring-1 ring-[rgba(123,97,255,0.65)] border-[rgba(123,97,255,0.45)]'
              )}
            >
              {meta.icon}
              {meta.label}
            </button>
          );
        })}
      </div>

      {caseType === 'report' && (
        <p className="text-xs text-[#A7B1C8]">
          Select <span className="text-[#FFD166]">Warn</span> or{' '}
          <span className="text-[#FF6B6B]">Ban</span>, then press{' '}
          <span className="text-[#B7A9FF]">Dismiss</span>.
        </p>
      )}
    </div>
  );
}

function DetailPanel({
  caseItem,
  onAction,
  canAction,
  reportDecision,
  onUpdateSupportStatus,
  onSendSupportReply,
  onOpenViewer,
  onOpenUserProfile,
}: {
  caseItem: ModerationCase | null;
  onAction: (action: ModerationAction) => void;
  canAction: (action: ModerationAction) => boolean;
  reportDecision: ReportDecision | null;
  onUpdateSupportStatus: (
    caseItem: ModerationCase,
    nextStatus: Extract<ModerationStatus, 'in_review' | 'waiting_user'>,
  ) => void;
  onSendSupportReply: (caseId: string, text: string) => void;
  onOpenViewer: (photos: string[], startIndex: number) => void;
  onOpenUserProfile: (user: UserSummary, contextCase: ModerationCase) => void;
}) {
  type DetailTab = 'evidence' | 'user' | 'history' | 'notes';

  const [activeTab, setActiveTab] = useState<DetailTab>('evidence');
  const [notesByCaseId, setNotesByCaseId] = useState<Record<string, string>>({});
  const [replyDraftByCaseId, setReplyDraftByCaseId] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!caseItem) {
      return;
    }

    setActiveTab('evidence');
  }, [caseItem]);

  const getIcon = (type: ModerationCaseType) => {
    switch (type) {
      case 'report':
        return <ImageIcon className="w-5 h-5" />;
      case 'support':
        return <MessageSquare className="w-5 h-5" />;
      case 'onboarding':
      default:
        return <Shield className="w-5 h-5" />;
    }
  };

  const tabLabels: Array<{ id: DetailTab; label: string }> = [
    { id: 'evidence', label: 'Evidence' },
    { id: 'user', label: 'User' },
    { id: 'history', label: 'History' },
    { id: 'notes', label: 'Notes' },
  ];

  if (!caseItem) {
    return (
      <div className="flex-1 min-h-0 flex items-center justify-center">
        <div className="text-center">
          <Shield className="w-16 h-16 text-[rgba(123,97,255,0.2)] mx-auto mb-4" />
          <p className="text-lg font-medium text-[#F5F7FF]">All caught up!</p>
          <p className="text-sm text-[#A7B1C8]">No more items in the moderation queue</p>
        </div>
      </div>
    );
  }

  const currentNote = notesByCaseId[caseItem.id] ?? '';

  const renderEvidence = () => {
    if (caseItem.type === 'onboarding' && caseItem.onboarding) {
      const [mainPhoto, secondPhoto, thirdPhoto] = caseItem.onboarding.photos;

      return (
        <div className="space-y-4">
          <div className="grid grid-cols-3 grid-rows-2 gap-2 h-52">
            {mainPhoto && (
              <img
                src={mainPhoto}
                alt="Onboarding primary"
                onClick={() => onOpenViewer(caseItem.onboarding!.photos, 0)}
                className="col-span-2 row-span-2 h-full w-full object-cover rounded-xl border border-[rgba(123,97,255,0.2)] cursor-pointer hover:opacity-95 transition"
              />
            )}
            {secondPhoto && (
              <img
                src={secondPhoto}
                alt="Onboarding secondary"
                onClick={() => onOpenViewer(caseItem.onboarding!.photos, 1)}
                className="h-full w-full object-cover rounded-xl border border-[rgba(123,97,255,0.2)] cursor-pointer hover:opacity-95 transition"
              />
            )}
            {thirdPhoto && (
              <img
                src={thirdPhoto}
                alt="Onboarding tertiary"
                onClick={() => onOpenViewer(caseItem.onboarding!.photos, 2)}
                className="h-full w-full object-cover rounded-xl border border-[rgba(123,97,255,0.2)] cursor-pointer hover:opacity-95 transition"
              />
            )}
          </div>

          <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
            <p className="text-xs uppercase tracking-wide text-[#A7B1C8] mb-3">Profile</p>
            <div className="grid grid-cols-2 gap-2">
              {[
                ['Display Name', caseItem.onboarding.displayName],
                ['Birthday', formatDateToEuropean(caseItem.onboarding.birthday)],
                ['Zodiac', caseItem.onboarding.zodiac],
                ['Gender', caseItem.onboarding.gender],
                ['Looking For', caseItem.onboarding.lookingFor],
                ['Dating Goal', caseItem.onboarding.datingGoal],
                ['City', caseItem.onboarding.city],
                ['Language', caseItem.onboarding.language],
              ].map(([label, value]) => (
                <div
                  key={`${caseItem.id}_profile_${label}`}
                  className="p-2 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]"
                >
                  <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">{label}</p>
                  <p className="text-sm text-[#F5F7FF]">{value}</p>
                </div>
              ))}
            </div>
            <div className="mt-2 p-2 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]">
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Bio</p>
              <p className="text-sm text-[#F5F7FF]">{caseItem.onboarding.bio}</p>
            </div>
          </div>

        </div>
      );
    }

    if (caseItem.type === 'report') {
      const reporter: UserSummary = caseItem.reporter ?? {
        id: `reporter_${caseItem.id}`,
        name: caseItem.reportedBy ?? 'Unknown reporter',
        username: caseItem.reportedBy,
      };

      return (
        <div className="space-y-4">
          {caseItem.contentMediaUrl && (
            <img
              src={caseItem.contentMediaUrl}
              alt="Reported content"
              className="max-w-md rounded-xl border border-[rgba(123,97,255,0.2)]"
            />
          )}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <button
              onClick={() => onOpenUserProfile(reporter, caseItem)}
              className="text-left p-3 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.12)] hover:border-[rgba(123,97,255,0.3)] transition-colors"
            >
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Reported by</p>
              <p className="text-sm text-[#F5F7FF]">{reporter.name}</p>
              <p className="text-xs text-[#A7B1C8]">{reporter.username ?? reporter.id}</p>
            </button>
            <button
              onClick={() => onOpenUserProfile(caseItem.user, caseItem)}
              className="text-left p-3 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.12)] hover:border-[rgba(123,97,255,0.3)] transition-colors"
            >
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Reported profile</p>
              <p className="text-sm text-[#F5F7FF]">{caseItem.user.name}</p>
              <p className="text-xs text-[#A7B1C8]">{caseItem.user.username ?? caseItem.user.id}</p>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <div className="p-3 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]">
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Reason</p>
              <p className="text-sm text-[#FF6B6B]">{caseItem.reportReason ?? getSubTypeLabel(caseItem)}</p>
            </div>
            <div className="p-3 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]">
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Reporter comment</p>
              <p className="text-sm text-[#F5F7FF]">{caseItem.reportComment ?? 'No comment provided.'}</p>
            </div>
          </div>

          {caseItem.contentText && (
            <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
              <p className="text-xs uppercase tracking-wide text-[#A7B1C8] mb-1">Reported content</p>
              <p className="text-sm text-[#F5F7FF]">{caseItem.contentText}</p>
            </div>
          )}

          <div className="text-xs text-[#A7B1C8]">
            Click reporter or target profile to open full user details.
          </div>
        </div>
      );
    }

    const firstUserMessage = (caseItem.messages ?? []).find((message) => message.from === 'user');
    const supportDescription = firstUserMessage?.text ?? caseItem.preview;
    const replyDraft = replyDraftByCaseId[caseItem.id] ?? '';
    const canSendReply = replyDraft.trim().length > 0;

    const handleSendReply = () => {
      const text = replyDraft.trim();
      if (!text) {
        return;
      }

      onSendSupportReply(caseItem.id, text);
      setReplyDraftByCaseId((prev) => ({ ...prev, [caseItem.id]: '' }));
    };

    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
          <div className="p-3 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]">
            <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Category</p>
            <p className="text-sm text-[#F5F7FF]">
              {supportSubTypeLabels[(caseItem.category ?? caseItem.subType) as SupportSubType]}
            </p>
          </div>
          <div className="p-3 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]">
            <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">Status</p>
            <p className="text-sm text-[#F5F7FF]">{statusLabels[caseItem.status]}</p>
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
          <p className="text-xs uppercase tracking-wide text-[#A7B1C8] mb-1">Description</p>
          <p className="text-sm text-[#F5F7FF]">{supportDescription}</p>
        </div>

        <button
          onClick={() => onOpenUserProfile(caseItem.user, caseItem)}
          className="w-full text-left p-3 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.12)] hover:border-[rgba(123,97,255,0.3)] transition-colors"
        >
          <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">User</p>
          <p className="text-sm text-[#F5F7FF]">{caseItem.user.name}</p>
          <p className="text-xs text-[#A7B1C8]">{caseItem.user.username ?? caseItem.user.id}</p>
        </button>

        <div className="space-y-2">
          <p className="text-xs uppercase tracking-wide text-[#A7B1C8]">Thread</p>
          {(caseItem.messages ?? []).map((message) => (
            <div
              key={message.id}
              className={cn(
                'max-w-[85%] p-3 rounded-xl border',
                message.from === 'admin' &&
                  'ml-auto bg-[rgba(123,97,255,0.12)] border-[rgba(123,97,255,0.2)]',
                message.from === 'user' &&
                  'mr-auto bg-[rgba(14,19,32,0.55)] border-[rgba(123,97,255,0.12)]',
                message.from === 'system' &&
                  'mr-auto bg-[rgba(167,177,200,0.08)] border-[rgba(167,177,200,0.2)]'
              )}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="text-[10px] uppercase text-[#A7B1C8]">{message.from}</span>
                <span className="text-[10px] text-[#A7B1C8]">{message.timestampLabel}</span>
              </div>
              <p className={cn('text-sm', message.from === 'system' ? 'text-[#C5CCDD]' : 'text-[#F5F7FF]')}>
                {message.text}
              </p>
            </div>
          ))}
        </div>

        {(caseItem.attachments ?? []).length > 0 && (
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-wide text-[#A7B1C8]">Attachments</p>
            <div className="flex flex-wrap gap-2">
              {(caseItem.attachments ?? []).map((attachment) => (
                <span
                  key={`${caseItem.id}_attachment_${attachment}`}
                  className="px-2 py-1 rounded-lg text-xs bg-[rgba(123,97,255,0.12)] text-[#B7A9FF] border border-[rgba(123,97,255,0.2)]"
                >
                  {attachment}
                </span>
              ))}
            </div>
          </div>
        )}

        <div className="space-y-2">
          <p className="text-xs uppercase tracking-wide text-[#A7B1C8]">Reply</p>
          <textarea
            value={replyDraft}
            onChange={(event) =>
              setReplyDraftByCaseId((prev) => ({ ...prev, [caseItem.id]: event.target.value }))
            }
            placeholder="Write a reply…"
            className="w-full min-h-24 resize-y px-3 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.55)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[#A7B1C8] focus:outline-none focus:border-[rgba(123,97,255,0.4)]"
          />
          <div className="flex justify-end">
            <button
              onClick={handleSendReply}
              disabled={!canSendReply}
              className="px-4 py-2 rounded-lg text-sm font-medium bg-[rgba(123,97,255,0.2)] text-[#7B61FF] border border-[rgba(123,97,255,0.25)] hover:bg-[rgba(123,97,255,0.3)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Send
            </button>
          </div>
        </div>
      </div>
    );
  };

  const renderUser = () => (
    <button
      onClick={() => onOpenUserProfile(caseItem.user, caseItem)}
      className="w-full text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
    >
      <div className="flex items-start gap-3">
        {caseItem.user.avatar ? (
          <img
            src={caseItem.user.avatar}
            alt={caseItem.user.name}
            className="w-14 h-14 rounded-lg object-cover border border-[rgba(123,97,255,0.2)]"
          />
        ) : (
          <div className="w-14 h-14 rounded-lg bg-[rgba(123,97,255,0.1)] flex items-center justify-center text-[#B7A9FF]">
            {caseItem.user.name.slice(0, 1).toUpperCase()}
          </div>
        )}
        <div className="space-y-1">
          <p className="text-base font-medium text-[#F5F7FF]">{caseItem.user.name}</p>
          <p className="text-sm text-[#A7B1C8]">{caseItem.user.username ?? 'No username'}</p>
          <p className="text-xs text-[#A7B1C8]">{caseItem.user.id}</p>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {caseItem.user.premium && (
          <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(255,209,102,0.15)] text-[#FFD166]">
            Premium
          </span>
        )}
        <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(123,97,255,0.15)] text-[#B7A9FF]">
          Age: {caseItem.user.age ?? 'N/A'}
        </span>
        <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(123,97,255,0.15)] text-[#B7A9FF]">
          Zodiac: {caseItem.user.zodiac ?? 'N/A'}
        </span>
        <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(123,97,255,0.15)] text-[#B7A9FF]">
          City: {caseItem.user.city ?? 'N/A'}
        </span>
      </div>
      <p className="mt-3 text-xs text-[#A7B1C8]">Click to open full profile</p>
    </button>
  );

  const renderHistory = () => (
    <div className="space-y-3">
      {(caseItem.history ?? []).length > 0 ? (
        (caseItem.history ?? []).map((event, index) => (
          <div key={event.id} className="flex gap-3">
            <div className="flex flex-col items-center">
              <span className="w-2 h-2 rounded-full bg-[#7B61FF] mt-1" />
              {index < (caseItem.history ?? []).length - 1 && (
                <span className="w-px h-8 bg-[rgba(123,97,255,0.2)] mt-1" />
              )}
            </div>
            <div className="flex-1 pb-2">
              <p className="text-sm text-[#F5F7FF]">{event.text}</p>
              <p className="text-xs text-[#A7B1C8] mt-0.5">{event.timestampLabel}</p>
            </div>
          </div>
        ))
      ) : (
        <p className="text-sm text-[#A7B1C8]">No history available</p>
      )}
    </div>
  );

  const renderNotes = () => (
    <div className="space-y-3">
      <label className="block">
        <span className="text-xs uppercase tracking-wide text-[#A7B1C8]">Internal note</span>
        <textarea
          value={currentNote}
          onChange={(event) =>
            setNotesByCaseId((prev) => ({ ...prev, [caseItem.id]: event.target.value }))
          }
          placeholder="Add internal context for moderators..."
          className="mt-1 w-full min-h-28 resize-y px-3 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.55)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[#A7B1C8] focus:outline-none focus:border-[rgba(123,97,255,0.4)]"
        />
      </label>
    </div>
  );

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <div className="flex flex-col h-full min-h-0">
        <div className="flex-1 p-6 min-h-0 overflow-auto space-y-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-[rgba(123,97,255,0.15)] flex items-center justify-center text-[#7B61FF]">
              {getIcon(caseItem.type)}
            </div>
            <div>
              <p className="text-sm font-medium text-[#F5F7FF]">{caseItem.title}</p>
              <p className="text-xs text-[#A7B1C8]">
                {caseItem.id} • {caseItem.createdAtLabel}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            {caseItem.type === 'onboarding' ? (
              <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(123,97,255,0.15)] text-[#B7A9FF] uppercase">
                onboarding
              </span>
            ) : (
              <>
                <span
                  className={cn(
                    'px-3 py-1 rounded-lg text-xs border uppercase',
                    caseItem.type === 'report'
                      ? 'bg-[rgba(255,107,107,0.12)] border-[rgba(255,107,107,0.22)] text-[#FFB8B8]'
                      : 'bg-[rgba(76,201,240,0.12)] border-[rgba(76,201,240,0.22)] text-[#B6EFFF]',
                  )}
                >
                  {caseItem.type}/{getSubTypeLabel(caseItem)}
                </span>
                <span
                  className={cn(
                    'px-3 py-1 rounded-lg text-xs border',
                    statusBadgeClassByStatus[caseItem.status],
                  )}
                >
                  {statusLabels[caseItem.status]}
                </span>
              </>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2 justify-between w-full">
            <div className="flex flex-wrap gap-1">
              {tabLabels.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    'px-3 py-1 rounded-lg text-xs font-medium transition-colors',
                    activeTab === tab.id
                      ? 'bg-[rgba(123,97,255,0.2)] text-[#7B61FF]'
                      : 'text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)]'
                  )}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            {caseItem.type === 'support' && (
              <div className="flex flex-wrap gap-2">
                <button
                  onClick={() => onUpdateSupportStatus(caseItem, 'in_review')}
                  className={cn(
                    'px-3 py-1.5 rounded-lg text-xs border transition-colors',
                    caseItem.status === 'in_review'
                      ? 'bg-[rgba(123,97,255,0.2)] border-[rgba(123,97,255,0.35)] text-[#E7E3FF]'
                      : 'bg-[rgba(123,97,255,0.08)] border-[rgba(123,97,255,0.2)] text-[#B7A9FF] hover:bg-[rgba(123,97,255,0.14)]',
                  )}
                >
                  In review
                </button>
                <button
                  onClick={() => onUpdateSupportStatus(caseItem, 'waiting_user')}
                  className={cn(
                    'px-3 py-1.5 rounded-lg text-xs border transition-colors',
                    caseItem.status === 'waiting_user'
                      ? 'bg-[rgba(255,209,102,0.2)] border-[rgba(255,209,102,0.35)] text-[#FFE2A6]'
                      : 'bg-[rgba(255,209,102,0.1)] border-[rgba(255,209,102,0.22)] text-[#FFD166] hover:bg-[rgba(255,209,102,0.16)]',
                  )}
                >
                  Waiting user
                </button>
              </div>
            )}
          </div>

          {activeTab === 'evidence' && renderEvidence()}
          {activeTab === 'user' && renderUser()}
          {activeTab === 'history' && renderHistory()}
          {activeTab === 'notes' && renderNotes()}
        </div>

        <ActionBar
          caseType={caseItem.type}
          onAction={onAction}
          canAction={canAction}
          reportDecision={reportDecision}
        />
      </div>
    </div>
  );
}

export function ModerationPage() {
  const initialCases = [...staticNonSupportCases, ...staticSupportFallbackCases];
  const [cases, setCases] = useState<ModerationCase[]>(initialCases);
  const [selectedType, setSelectedType] = useState<ModerationViewType>('all');
  const [selectedSubType, setSelectedSubType] = useState('');
  const [selectedCaseId, setSelectedCaseId] = useState<string | null>(initialCases[0]?.id ?? null);
  const [query, setQuery] = useState('');
  const [viewerOpen, setViewerOpen] = useState(false);
  const [viewerPhotos, setViewerPhotos] = useState<string[]>([]);
  const [viewerIndex, setViewerIndex] = useState(0);
  const [profileViewer, setProfileViewer] = useState<{ user: UserSummary; contextCase: ModerationCase } | null>(null);
  const [profileActiveTab, setProfileActiveTab] = useState<'activity' | 'limits' | 'moderation'>('activity');
  const [profileActiveInteraction, setProfileActiveInteraction] = useState<ProfileInteractionType | null>(null);
  const [reportDecisionByCaseId, setReportDecisionByCaseId] = useState<Record<string, ReportDecision>>({});
  const [templateSheet, setTemplateSheet] = useState<{
    caseId: string;
    action: 'reject' | 'request_changes' | 'dismiss';
    resolution?: ReportDecision;
    title: string;
    subtitle: string;
    templates: ActionTemplate[];
    selectedTemplateId: string;
    message: string;
  } | null>(null);
  const [profileLimitsByUserId, setProfileLimitsByUserId] = useState<Record<string, ProfileLimitsState>>({});
  const [profileLimitsEditMode, setProfileLimitsEditMode] = useState(false);
  const supportAPIEnabled = isSupportApiConfigured();
  const { hasPermission, role } = usePermissions();
  const totalPending = cases.length;
  const canChangeLimits = hasPermission(ADMIN_PERMISSIONS.change_limits);

  const openViewer = (photos: string[], startIndex: number) => {
    if (photos.length === 0) {
      return;
    }

    setViewerPhotos(photos);
    setViewerIndex(startIndex);
    setViewerOpen(true);
  };

  const closeViewer = () => {
    setViewerOpen(false);
    setViewerPhotos([]);
    setViewerIndex(0);
  };

  const openProfileViewer = (user: UserSummary, contextCase: ModerationCase) => {
    setProfileViewer({ user, contextCase });
    setProfileActiveTab('activity');
    setProfileActiveInteraction(null);
    setProfileLimitsEditMode(false);
  };

  const closeProfileViewer = () => {
    setProfileViewer(null);
    setProfileActiveInteraction(null);
    setProfileLimitsEditMode(false);
  };

  const nextViewer = () => {
    setViewerIndex((prev) => {
      if (viewerPhotos.length === 0) {
        return 0;
      }

      return (prev + 1) % viewerPhotos.length;
    });
  };

  const prevViewer = () => {
    setViewerIndex((prev) => {
      if (viewerPhotos.length === 0) {
        return 0;
      }

      return (prev - 1 + viewerPhotos.length) % viewerPhotos.length;
    });
  };

  const countsByType = getCountsByType(cases);
  const filteredCases = filterCases({ cases, selectedType, selectedSubType, query });
  const selectedCase = filteredCases.find((caseItem) => caseItem.id === selectedCaseId) ?? null;
  const selectedReportDecision =
    selectedCase && selectedCase.type === 'report'
      ? reportDecisionByCaseId[selectedCase.id] ?? null
      : null;

  useEffect(() => {
    syncModerationCasesSnapshot(cases);
  }, [cases, totalPending]);

  useEffect(() => {
    if (!selectedCaseId) {
      setSelectedCaseId(filteredCases[0]?.id ?? null);
      return;
    }

    const stillExists = filteredCases.some((caseItem) => caseItem.id === selectedCaseId);
    if (!stillExists) {
      setSelectedCaseId(filteredCases[0]?.id ?? null);
    }
  }, [selectedType, selectedSubType, query, cases]);

  useEffect(() => {
    if (!templateSheet) {
      return;
    }

    const stillExists = cases.some((caseItem) => caseItem.id === templateSheet.caseId);
    if (!stillExists) {
      setTemplateSheet(null);
    }
  }, [cases, templateSheet]);

  useEffect(() => {
    if (!templateSheet) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setTemplateSheet(null);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [templateSheet]);

  useEffect(() => {
    if (!viewerOpen) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
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

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [viewerOpen, viewerPhotos.length]);

  useEffect(() => {
    if (!profileViewer || viewerOpen) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeProfileViewer();
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [profileViewer, viewerOpen]);

  useEffect(() => {
    if (!profileViewer) {
      return;
    }

    const userId = profileViewer.user.id;
    setProfileLimitsByUserId((prev) =>
      prev[userId] ? prev : { ...prev, [userId]: createInitialProfileLimits(userId) },
    );
  }, [profileViewer?.user.id]);

  useEffect(() => {
    if (!supportAPIEnabled) {
      return;
    }

    let cancelled = false;

    const syncSupportConversations = async () => {
      try {
        const conversations = await listSupportConversations({ limit: 200, offset: 0 });
        if (cancelled) {
          return;
        }

        const liveSupportCases = conversations
          .filter((conversation) => conversation.status !== 'done')
          .map(mapSupportConversationToCase);
        setCases((prevCases) => {
          const existingMessagesByCase = new Map(
            prevCases
              .filter((caseItem) => caseItem.type === 'support')
              .map((caseItem) => [caseItem.id, caseItem.messages]),
          );

          const mergedSupportCases = liveSupportCases.map((caseItem) => {
            const existingMessages = existingMessagesByCase.get(caseItem.id);
            if (!existingMessages || existingMessages.length === 0) {
              return caseItem;
            }
            return {
              ...caseItem,
              messages: existingMessages,
            };
          });

          return [...prevCases.filter((caseItem) => caseItem.type !== 'support'), ...mergedSupportCases];
        });
      } catch (error) {
        console.warn('Failed to sync support conversations', error);
      }
    };

    void syncSupportConversations();
    const intervalID = window.setInterval(() => {
      void syncSupportConversations();
    }, 10_000);

    return () => {
      cancelled = true;
      window.clearInterval(intervalID);
    };
  }, [supportAPIEnabled]);

  useEffect(() => {
    if (!supportAPIEnabled || !selectedCase || selectedCase.type !== 'support') {
      return;
    }

    const conversationID = supportConversationIDFromCaseID(selectedCase.id);
    if (!conversationID) {
      return;
    }

    let cancelled = false;

    const loadMessages = async () => {
      try {
        const messages = await listSupportMessages(conversationID, 200);
        if (cancelled) {
          return;
        }

        const mappedMessages = messages.map(mapSupportMessageToUI);
        setCases((prevCases) =>
          prevCases.map((caseItem) =>
            caseItem.id === selectedCase.id
              ? {
                  ...caseItem,
                  messages: mappedMessages,
                }
              : caseItem,
          ),
        );

        await markSupportConversationRead(conversationID);
      } catch (error) {
        console.warn('Failed to load support messages', error);
      }
    };

    void loadMessages();

    return () => {
      cancelled = true;
    };
  }, [supportAPIEnabled, selectedCase?.id, selectedCase?.type]);

  const canAction = (action: ModerationAction) => hasPermission(actionPermissions[action]);

  const handleSupportStatusUpdate = async (
    caseItem: ModerationCase,
    nextStatus: Extract<ModerationStatus, 'in_review' | 'waiting_user'>,
  ) => {
    if (caseItem.type !== 'support' || caseItem.status === nextStatus) {
      return;
    }

    logAdminAction(
      `moderation_support_status_${nextStatus}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
    appendModerationChangeLog({
      type: 'action',
      caseId: caseItem.id,
      caseType: caseItem.type,
      subType: caseItem.subType,
      targetUserId: caseItem.user.id,
      actorId: 'current-admin',
      payload: {
        action: 'set_status',
        statusBefore: caseItem.status,
        statusAfter: nextStatus,
      },
    });

    setCases((prevCases) =>
      prevCases.map((queueCase) =>
        queueCase.id === caseItem.id
          ? {
              ...queueCase,
              status: nextStatus,
            }
          : queueCase,
      ),
    );

    if (!supportAPIEnabled) {
      return;
    }

    const conversationID = supportConversationIDFromCaseID(caseItem.id);
    if (!conversationID) {
      return;
    }

    try {
      const targetStatus: SupportConversationStatus =
        nextStatus === 'waiting_user' ? 'waiting' : 'in_review';
      await setSupportConversationStatus(conversationID, targetStatus);
    } catch (error) {
      console.warn('Failed to update support case status', error);
    }
  };

  const removeCaseFromQueue = (caseID: string) => {
    setReportDecisionByCaseId((prev) => {
      if (!(caseID in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[caseID];
      return next;
    });

    setCases((prevCases) => {
      const nextCases = prevCases.filter((caseItem) => caseItem.id !== caseID);
      const nextFilteredCases = filterCases({
        cases: nextCases,
        selectedType,
        selectedSubType,
        query,
      });
      const nextSelectedCaseID = nextFilteredCases[0]?.id ?? null;

      queueMicrotask(() => {
        setSelectedCaseId(nextSelectedCaseID);
      });

      return nextCases;
    });
  };

  const openTemplateSheetForAction = ({
    caseItem,
    action,
    resolution,
    templates,
    title,
    subtitle,
  }: {
    caseItem: ModerationCase;
    action: 'reject' | 'request_changes' | 'dismiss';
    resolution?: ReportDecision;
    templates: ActionTemplate[];
    title: string;
    subtitle: string;
  }) => {
    if (templates.length === 0) {
      return;
    }

    setTemplateSheet({
      caseId: caseItem.id,
      action,
      resolution,
      title,
      subtitle,
      templates,
      selectedTemplateId: templates[0].id,
      message: templates[0].message,
    });
  };

  const performModerationAction = async (
    caseItem: ModerationCase,
    action: ModerationAction,
    extra?: {
      reportDecision?: ReportDecision;
      templateId?: string;
      outboundMessage?: string;
    },
  ) => {
    logAdminAction(
      `moderation_${action}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
    appendModerationChangeLog({
      type: 'action',
      caseId: caseItem.id,
      caseType: caseItem.type,
      subType: caseItem.subType,
      targetUserId: caseItem.user.id,
      actorId: 'current-admin',
      payload: {
        action,
        statusBefore: caseItem.status,
        reportDecision: extra?.reportDecision ?? '',
        templateId: extra?.templateId ?? '',
        outboundMessage: extra?.outboundMessage ?? '',
      },
    });

    if (caseItem.type === 'support' && supportAPIEnabled) {
      const conversationID = supportConversationIDFromCaseID(caseItem.id);
      if (!conversationID) {
        removeCaseFromQueue(caseItem.id);
        return;
      }

      try {
        if (action === 'resolve') {
          await setSupportConversationStatus(conversationID, 'done');
          removeCaseFromQueue(caseItem.id);
          return;
        }

        if (action === 'request_info') {
          const nextStatus: SupportConversationStatus = 'waiting';
          await setSupportConversationStatus(conversationID, nextStatus);
          setCases((prevCases) =>
            prevCases.map((queueCase) =>
              queueCase.id === caseItem.id
                ? {
                    ...queueCase,
                    status: mapSupportStatusToModeration(nextStatus),
                  }
                : queueCase,
            ),
          );
          return;
        }
      } catch (error) {
        console.warn('Failed to update support conversation status', error);
        return;
      }
    }

    removeCaseFromQueue(caseItem.id);
  };

  const handleTemplateSelect = (templateId: string) => {
    setTemplateSheet((prev) => {
      if (!prev) {
        return prev;
      }

      const template = prev.templates.find((item) => item.id === templateId);
      if (!template) {
        return prev;
      }

      return {
        ...prev,
        selectedTemplateId: templateId,
        message: template.message,
      };
    });
  };

  const handleTemplateMessageChange = (value: string) => {
    setTemplateSheet((prev) => (prev ? { ...prev, message: value } : prev));
  };

  const closeTemplateSheet = () => {
    setTemplateSheet(null);
  };

  const handleConfirmTemplateAction = async () => {
    if (!templateSheet) {
      return;
    }

    const targetCase = cases.find((caseItem) => caseItem.id === templateSheet.caseId);
    if (!targetCase) {
      setTemplateSheet(null);
      return;
    }

    if (!canAction(templateSheet.action)) {
      return;
    }

    const outboundMessage = templateSheet.message.trim();
    if (!outboundMessage) {
      return;
    }

    if (templateSheet.action === 'dismiss' && !templateSheet.resolution) {
      return;
    }

    console.log('moderation_template_payload', {
      caseId: targetCase.id,
      userId: targetCase.user.id,
      username: targetCase.user.username,
      action: templateSheet.action,
      reportDecision: templateSheet.resolution ?? null,
      templateId: templateSheet.selectedTemplateId,
      message: outboundMessage,
    });

    await performModerationAction(targetCase, templateSheet.action, {
      reportDecision: templateSheet.resolution,
      templateId: templateSheet.selectedTemplateId,
      outboundMessage,
    });

    setTemplateSheet(null);
  };

  const handleAction = async (action: ModerationAction) => {
    if (!selectedCase || !canAction(action)) {
      return;
    }

    if (selectedCase.type === 'report' && (action === 'warn' || action === 'ban')) {
      setReportDecisionByCaseId((prev) => ({ ...prev, [selectedCase.id]: action }));
      logAdminAction(
        `moderation_select_${action}`,
        { id: 'current-admin', role },
        '127.0.0.1',
        getClientDevice(),
      );
      return;
    }

    if (selectedCase.type === 'onboarding' && action === 'reject') {
      openTemplateSheetForAction({
        caseItem: selectedCase,
        action: 'reject',
        templates: onboardingRejectTemplates,
        title: 'Reject Templates',
        subtitle: 'Choose a message template to notify the user about rejection.',
      });
      return;
    }

    if (selectedCase.type === 'onboarding' && action === 'request_changes') {
      openTemplateSheetForAction({
        caseItem: selectedCase,
        action: 'request_changes',
        templates: onboardingRequestChangesTemplates,
        title: 'Request Changes Templates',
        subtitle: 'Choose what to request from the user before next moderation pass.',
      });
      return;
    }

    if (selectedCase.type === 'report' && action === 'dismiss') {
      const decision = reportDecisionByCaseId[selectedCase.id];
      if (!decision) {
        return;
      }

      openTemplateSheetForAction({
        caseItem: selectedCase,
        action: 'dismiss',
        resolution: decision,
        templates: decision === 'ban' ? reportBanTemplates : reportWarnTemplates,
        title: decision === 'ban' ? 'Ban Notification Templates' : 'Warn Notification Templates',
        subtitle: 'Choose a Telegram message template, then dismiss this report case.',
      });
      return;
    }

    await performModerationAction(selectedCase, action);
  };

  const handleSelectType = (type: ModerationViewType) => {
    setSelectedType(type);
    setSelectedSubType('');
  };

  const handleSendSupportReply = async (caseId: string, text: string) => {
    const message = text.trim();
    if (!message) {
      return;
    }

    const targetCase = cases.find((caseItem) => caseItem.id === caseId);
    if (!targetCase) {
      return;
    }

    appendModerationChangeLog({
      type: 'reply',
      caseId: targetCase.id,
      caseType: targetCase.type,
      subType: targetCase.subType,
      targetUserId: targetCase.user.id,
      actorId: 'current-admin',
      payload: {
        message,
      },
    });

    const conversationID = supportConversationIDFromCaseID(caseId);
    if (supportAPIEnabled && conversationID) {
      try {
        const response = await sendSupportMessage(conversationID, message);
        setCases((prevCases) =>
          prevCases.map((caseItem) =>
            caseItem.id === caseId
              ? {
                  ...caseItem,
                  status: 'in_review',
                  preview: message,
                  messages: [...(caseItem.messages ?? []), mapSupportMessageToUI(response.message)],
                }
              : caseItem,
          ),
        );
        return;
      } catch (error) {
        console.warn('Failed to send support reply', error);
        return;
      }
    }

    setCases((prev) =>
      prev.map((caseItem) =>
        caseItem.id === caseId
          ? {
              ...caseItem,
              messages: [
                ...(caseItem.messages ?? []),
                { id: `m_${Date.now()}`, from: 'admin', text: message, timestampLabel: 'just now' },
              ],
            }
          : caseItem
      )
    );
  };

  const profileContextOnboarding = profileViewer?.contextCase.onboarding;
  const profilePresence = deriveProfilePresence(
    profileViewer?.user.lastActiveLabel ?? profileViewer?.contextCase.createdAtLabel,
  );
  const profileStatus = profileStatusConfig[profilePresence];
  const profileTelegramId = profileViewer ? resolveTelegramId(profileViewer.user.id) : 'unknown';
  const profilePhotoSource =
    profileViewer?.user.photos && profileViewer.user.photos.length > 0
      ? profileViewer.user.photos
      : profileContextOnboarding?.photos ?? [];
  const profilePhotoFallback =
    profileViewer?.user.avatar ?? 'https://picsum.photos/seed/mod_profile_fallback/600/800';
  const profilePhotos =
    profilePhotoSource.length >= 3
      ? profilePhotoSource.slice(0, 3)
      : [
          ...profilePhotoSource,
          ...Array.from(
            { length: 3 - profilePhotoSource.length },
            () => profilePhotoSource[profilePhotoSource.length - 1] ?? profilePhotoFallback,
          ),
        ];
  const profilePrimaryPhoto = profilePhotos[0];
  const profileHeight = profileViewer?.user.heightCm ?? seedMetric(`${profileViewer?.user.id ?? 'u'}_h`, 155, 195);
  const profileEyeColor = profileViewer?.user.eyeColor ?? ['Hazel', 'Blue', 'Brown', 'Green'][seedMetric(`${profileViewer?.user.id ?? 'u'}_e`, 0, 3)];
  const profileMatches = seedMetric(`${profileViewer?.user.id ?? 'u'}_m`, 12, 96);
  const profileLikesSent = seedMetric(`${profileViewer?.user.id ?? 'u'}_ls`, 25, 180);
  const profileLikesReceived = seedMetric(`${profileViewer?.user.id ?? 'u'}_lr`, 35, 220);
  const profileTrustScore = seedMetric(`${profileViewer?.user.id ?? 'u'}_ts`, 72, 99);
  const profileInterests =
    profileViewer?.user.interests && profileViewer.user.interests.length > 0
      ? profileViewer.user.interests
      : ['Travel', 'Music', 'Gym', 'Cinema', 'Coffee'];
  const profileBio = profileViewer?.user.bio ?? profileContextOnboarding?.bio ?? '';
  const profileGender = profileViewer?.user.gender ?? profileContextOnboarding?.gender ?? 'N/A';
  const profileAge = profileViewer?.user.age ?? seedMetric(`${profileViewer?.user.id ?? 'u'}_a`, 21, 39);
  const profileLocation = profileViewer?.user.city ?? profileContextOnboarding?.city ?? 'Unknown';
  const profileLastActive = profileViewer?.user.lastActiveLabel ?? profileViewer?.contextCase.createdAtLabel ?? 'N/A';
  const profileJoined = profileViewer
    ? formatJoinedSummary(profileViewer.user.joinedAt, profileViewer.contextCase.createdAtTs)
    : 'N/A';
  const profileFields = profileViewer
    ? [
        ['Display name', profileContextOnboarding?.displayName ?? profileViewer.user.name],
        [
          'Birthday',
          profileViewer.user.birthday
            ? formatDateToEuropean(profileViewer.user.birthday)
            : profileContextOnboarding?.birthday
            ? formatDateToEuropean(profileContextOnboarding.birthday)
            : 'N/A',
        ],
        ['Gender', profileViewer.user.gender ?? profileContextOnboarding?.gender ?? 'N/A'],
        ['Looking for', profileViewer.user.lookingFor ?? profileContextOnboarding?.lookingFor ?? 'N/A'],
        ['Dating goal', profileViewer.user.datingGoal ?? profileContextOnboarding?.datingGoal ?? 'N/A'],
        ['Language', profileViewer.user.language ?? profileContextOnboarding?.language ?? 'N/A'],
        ['City', profileViewer.user.city ?? profileContextOnboarding?.city ?? 'N/A'],
        ['Age', String(profileAge)],
        ['Zodiac', profileViewer.user.zodiac ?? profileContextOnboarding?.zodiac ?? 'N/A'],
        ['Phone', profileViewer.user.phone ?? 'N/A'],
      ]
    : [];
  const profileLimits = profileViewer
    ? profileLimitsByUserId[profileViewer.user.id] ?? createInitialProfileLimits(profileViewer.user.id)
    : null;
  const profileInteractionProfiles =
    profileViewer && profileActiveInteraction
      ? resolveProfileInteractionProfiles(profileViewer.user, profileActiveInteraction)
      : [];
  const profileInteractionTitle =
    profileActiveInteraction === 'matches'
      ? 'Matches'
      : profileActiveInteraction === 'likes_sent'
        ? 'Likes Sent'
        : 'Likes Received';
  const limitAdjustButtonClass =
    'w-6 h-6 rounded-md border border-[rgba(123,97,255,0.25)] text-[#CFC6FF] hover:bg-[rgba(123,97,255,0.18)] disabled:opacity-0 disabled:pointer-events-none';

  const handleProfileEditLimits = () => {
    if (!profileViewer || !canChangeLimits) {
      return;
    }

    if (profileLimitsEditMode) {
      logAdminAction(
        `save_limits_${profileViewer.user.id}`,
        { id: 'current-admin', role },
        '127.0.0.1',
        getClientDevice(),
      );
      setProfileLimitsEditMode(false);
      return;
    }

    logAdminAction(
      `edit_limits_${profileViewer.user.id}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
    setProfileLimitsEditMode(true);
  };

  const adjustProfileLimit = (kind: ProfileLimitKind, delta: number) => {
    if (!profileViewer || !profileLimitsEditMode) {
      return;
    }

    const userId = profileViewer.user.id;
    setProfileLimitsByUserId((prev) => {
      const current = prev[userId] ?? createInitialProfileLimits(userId);
      const next = { ...current };

      if (kind === 'daily_swipes') {
        next.dailySwipesRemaining = Math.max(0, next.dailySwipesRemaining + delta);
      }
      if (kind === 'super_likes') {
        next.superLikesRemaining = Math.max(0, next.superLikesRemaining + delta);
      }
      if (kind === 'boosts') {
        next.boostsRemaining = Math.max(0, next.boostsRemaining + delta);
      }

      return { ...prev, [userId]: next };
    });
  };

  const templateConfirmLabel =
    templateSheet?.action === 'request_changes'
      ? 'Send request changes'
      : templateSheet?.action === 'dismiss'
        ? 'Dismiss and send'
        : 'Reject and send';

  return (
    <div className="p-6 animate-fade-in flex flex-col">
      <div className="glass-panel flex-1 min-h-[calc(100vh-64px-48px)] max-h-[min(calc(100vh-64px-48px),720px)] overflow-hidden flex">
        <div className="w-[380px] border-r border-[rgba(123,97,255,0.12)] flex flex-col">
          <FiltersBar
            counts={countsByType}
            selectedType={selectedType}
            onSelectType={handleSelectType}
            selectedSubType={selectedSubType}
            onSelectSubType={setSelectedSubType}
            query={query}
            onChangeQuery={setQuery}
          />
          <QueueList
            cases={filteredCases}
            selectedCaseId={selectedCaseId}
            onSelectCase={setSelectedCaseId}
          />
        </div>

        <DetailPanel
          caseItem={selectedCase}
          onAction={handleAction}
          canAction={canAction}
          reportDecision={selectedReportDecision}
          onUpdateSupportStatus={handleSupportStatusUpdate}
          onSendSupportReply={handleSendSupportReply}
          onOpenViewer={openViewer}
          onOpenUserProfile={openProfileViewer}
        />
      </div>

      {typeof document !== 'undefined' && templateSheet && createPortal(
        <div
          className="fixed inset-0 z-[97] flex items-end justify-center p-4"
          onClick={closeTemplateSheet}
        >
          <div className="absolute inset-0 bg-black/60 backdrop-blur-[2px]" />
          <div
            className="relative w-full max-w-3xl glass-panel border border-[rgba(123,97,255,0.22)] rounded-2xl overflow-hidden animate-slide-up"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-start justify-between gap-3">
              <div>
                <p className="text-base font-semibold text-[#F5F7FF]">{templateSheet.title}</p>
                <p className="text-xs text-[#A7B1C8] mt-1">{templateSheet.subtitle}</p>
              </div>
              <button
                onClick={closeTemplateSheet}
                className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-4 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                {templateSheet.templates.map((template) => {
                  const isSelected = template.id === templateSheet.selectedTemplateId;

                  return (
                    <button
                      key={template.id}
                      onClick={() => handleTemplateSelect(template.id)}
                      className={cn(
                        'text-left p-3 rounded-xl border transition-colors',
                        isSelected
                          ? 'bg-[rgba(123,97,255,0.14)] border-[rgba(123,97,255,0.45)]'
                          : 'bg-[rgba(14,19,32,0.45)] border-[rgba(123,97,255,0.14)] hover:border-[rgba(123,97,255,0.3)]',
                      )}
                    >
                      <p className="text-sm font-medium text-[#F5F7FF]">{template.title}</p>
                      <p className="text-xs text-[#A7B1C8] mt-1 leading-relaxed">{template.message}</p>
                    </button>
                  );
                })}
              </div>

              <div className="space-y-2">
                <p className="text-xs text-[#A7B1C8] uppercase tracking-wide">Message</p>
                <textarea
                  value={templateSheet.message}
                  onChange={(event) => handleTemplateMessageChange(event.target.value)}
                  rows={4}
                  className="w-full px-3 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.55)] border border-[rgba(123,97,255,0.22)] text-[#F5F7FF] placeholder:text-[#A7B1C8] focus:outline-none focus:border-[rgba(123,97,255,0.45)] resize-none"
                  placeholder="Type message for Telegram user..."
                />
              </div>

              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-xs text-[#A7B1C8]">
                  This message will be sent to the user after confirmation.
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={closeTemplateSheet}
                    className="px-4 py-2 rounded-lg text-sm border border-[rgba(123,97,255,0.22)] text-[#A7B1C8] hover:text-[#F5F7FF] hover:border-[rgba(123,97,255,0.4)] transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => void handleConfirmTemplateAction()}
                    disabled={!templateSheet.message.trim()}
                    className="px-4 py-2 rounded-lg text-sm font-medium btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {templateConfirmLabel}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      , document.body)}

      {typeof document !== 'undefined' && profileViewer && createPortal(
        <div
          className="fixed inset-0 z-[95] flex items-center justify-center p-4"
        >
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={closeProfileViewer}
          />
          <div
            className="relative w-full max-w-2xl glass-panel max-h-[90vh] overflow-hidden flex flex-col animate-slide-up"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="p-6 border-b border-[rgba(123,97,255,0.12)]">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4">
                  <button
                    onClick={() => profilePhotos.length > 0 && openViewer(profilePhotos, 0)}
                    className="relative rounded-2xl transition hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.3)]"
                    aria-label="Open profile photos"
                  >
                    {profilePrimaryPhoto ? (
                      <img
                        src={profilePrimaryPhoto}
                        alt={profileViewer.user.name}
                        className="w-20 h-20 rounded-2xl border-2 border-[rgba(123,97,255,0.25)] cursor-zoom-in object-cover"
                      />
                    ) : (
                      <div className="w-20 h-20 rounded-2xl border-2 border-[rgba(123,97,255,0.25)] bg-[rgba(123,97,255,0.12)] flex items-center justify-center text-2xl text-[#B7A9FF]">
                        {profileViewer.user.name.slice(0, 1).toUpperCase()}
                      </div>
                    )}
                    <span
                      className={cn(
                        'absolute -bottom-1 -right-1 w-5 h-5 rounded-full border-2 border-[#0E1320]',
                        profileStatus.dot,
                      )}
                    />
                  </button>

                  <div className="min-w-0">
                    <h3 className="text-xl font-bold text-[#F5F7FF]">
                      {profileViewer.user.name}, {profileAge}
                    </h3>
                    <div className="mt-0.5 flex flex-wrap items-center gap-2 text-sm text-[#A7B1C8]">
                      <span>{profileViewer.user.username ?? 'No username'}</span>
                      <span className="text-[rgba(167,177,200,0.5)]">•</span>
                      <span className="inline-flex items-center gap-1.5">
                        <Phone className="w-3.5 h-3.5" />
                        {profileViewer.user.phone ?? 'No phone'}
                      </span>
                    </div>
                    <p className="mt-0.5 text-sm text-[#A7B1C8]">Telegram ID: {profileTelegramId}</p>

                    <div className="flex items-center gap-3 mt-2 flex-wrap">
                      <span
                        className={cn(
                          'px-2 py-0.5 rounded-full text-xs font-medium',
                          profileStatus.bg,
                          profileStatus.text,
                        )}
                      >
                        {profileStatus.label}
                      </span>
                      {profileViewer.user.premium && (
                        <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-[rgba(123,97,255,0.15)] text-[#7B61FF] flex items-center gap-1">
                          <Star className="w-3 h-3" />
                          Gold
                        </span>
                      )}
                      <span className="text-xs text-[#A7B1C8]">
                        {profileHeight} cm • {profileEyeColor} eyes
                      </span>
                    </div>
                  </div>
                </div>

                <button
                  onClick={closeProfileViewer}
                  className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            <div className="flex border-b border-[rgba(123,97,255,0.12)]">
              {(['activity', 'limits', 'moderation'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setProfileActiveTab(tab)}
                  className={cn(
                    'flex-1 py-3 text-sm font-medium capitalize transition-colors relative',
                    profileActiveTab === tab
                      ? 'text-[#F5F7FF]'
                      : 'text-[#A7B1C8] hover:text-[#F5F7FF]',
                  )}
                >
                  {tab}
                  {profileActiveTab === tab && (
                    <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-[#7B61FF]" />
                  )}
                </button>
              ))}
            </div>

            <div className="p-6 overflow-y-auto scrollbar-thin flex-1 space-y-4">
              {profileActiveTab === 'activity' && (
                <>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <button
                      onClick={() => setProfileActiveInteraction('matches')}
                      className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                    >
                      <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                        <Heart className="w-4 h-4" />
                        <span className="text-sm">Matches</span>
                      </div>
                      <p className="text-2xl font-bold text-[#F5F7FF]">{profileMatches}</p>
                    </button>
                    <button
                      onClick={() => setProfileActiveInteraction('likes_sent')}
                      className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                    >
                      <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                        <Heart className="w-4 h-4" />
                        <span className="text-sm">Likes Sent</span>
                      </div>
                      <p className="text-2xl font-bold text-[#F5F7FF]">{profileLikesSent}</p>
                    </button>
                    <button
                      onClick={() => setProfileActiveInteraction('likes_received')}
                      className="text-left p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                    >
                      <div className="flex items-center gap-2 text-[#A7B1C8] mb-1">
                        <Star className="w-4 h-4" />
                        <span className="text-sm">Likes Received</span>
                      </div>
                      <p className="text-2xl font-bold text-[#F5F7FF]">{profileLikesReceived}</p>
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                      <Clock className="w-5 h-5 text-[#A7B1C8]" />
                      <div>
                        <p className="text-sm text-[#A7B1C8]">Last Active</p>
                        <p className="text-sm text-[#F5F7FF]">{profileLastActive}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                      <Calendar className="w-5 h-5 text-[#A7B1C8]" />
                      <div>
                        <p className="text-sm text-[#A7B1C8]">Joined</p>
                        <p className="text-sm text-[#F5F7FF]">{profileJoined}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                      <MapPin className="w-5 h-5 text-[#A7B1C8]" />
                      <div>
                        <p className="text-sm text-[#A7B1C8]">Location</p>
                        <p className="text-sm text-[#F5F7FF]">{profileLocation}</p>
                      </div>
                    </div>
                  </div>

                  <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <p className="text-sm text-[#A7B1C8] mb-2">Profile</p>
                    <div className="grid grid-cols-2 gap-3 text-sm mb-3">
                      <div>
                        <p className="text-[#A7B1C8] text-xs">Gender</p>
                        <p className="text-[#F5F7FF]">{profileGender}</p>
                      </div>
                      <div>
                        <p className="text-[#A7B1C8] text-xs">Age</p>
                        <p className="text-[#F5F7FF]">{profileAge}</p>
                      </div>
                      <div>
                        <p className="text-[#A7B1C8] text-xs">Height</p>
                        <p className="text-[#F5F7FF]">{profileHeight} cm</p>
                      </div>
                      <div>
                        <p className="text-[#A7B1C8] text-xs">Eyes</p>
                        <p className="text-[#F5F7FF]">{profileEyeColor}</p>
                      </div>
                    </div>
                    <p className="text-sm text-[#A7B1C8] mb-1">Bio</p>
                    <p className="text-sm text-[#F5F7FF]">{profileBio || 'No bio provided.'}</p>
                  </div>

                  <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <p className="text-sm text-[#A7B1C8] mb-3">Interests</p>
                    <div className="flex flex-wrap gap-2">
                      {profileInterests.map((interest) => (
                        <span
                          key={`${profileViewer.user.id}_interest_${interest}`}
                          className="px-2.5 py-1 rounded-full text-xs bg-[rgba(123,97,255,0.14)] border border-[rgba(123,97,255,0.25)] text-[#CFC6FF]"
                        >
                          {interest}
                        </span>
                      ))}
                    </div>
                  </div>

                  {profileFields.length > 0 && (
                    <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                      <p className="text-sm text-[#A7B1C8] mb-3">Registration Data</p>
                      <div className="grid grid-cols-2 gap-3 text-sm">
                        {profileFields.map(([label, value]) => (
                          <div key={`${profileViewer.user.id}_field_${label}`}>
                            <p className="text-[#A7B1C8] text-xs">{label}</p>
                            <p className="text-[#F5F7FF]">{value}</p>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}

              {profileActiveTab === 'limits' && (
                <>
                  <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm text-[#A7B1C8]">Daily Swipes</span>
                      <div className="flex items-center justify-end gap-2 min-w-[240px]">
                        <div className="flex items-center justify-end gap-2 w-[60px] translate-x-2">
                          <button
                            onClick={() => adjustProfileLimit('daily_swipes', -1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            -
                          </button>
                          <button
                            onClick={() => adjustProfileLimit('daily_swipes', 1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            +
                          </button>
                        </div>
                        <span className="text-sm text-[#F5F7FF] min-w-[170px] text-right tabular-nums">
                          {profileLimits?.dailySwipesRemaining ?? 0} / {profileLimits?.dailySwipesTotal ?? 0} remaining
                        </span>
                      </div>
                    </div>
                    <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                      <div
                        className="h-full rounded-full bg-[#7B61FF]"
                        style={{
                          width: profileLimits
                            ? `${Math.max(
                                0,
                                Math.min(100, (profileLimits.dailySwipesRemaining / profileLimits.dailySwipesTotal) * 100),
                              )}%`
                            : '0%',
                        }}
                      />
                    </div>
                  </div>
                  <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm text-[#A7B1C8]">Super Likes</span>
                      <div className="flex items-center justify-end gap-2 min-w-[240px]">
                        <div className="flex items-center justify-end gap-2 w-[60px] translate-x-2">
                          <button
                            onClick={() => adjustProfileLimit('super_likes', -1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            -
                          </button>
                          <button
                            onClick={() => adjustProfileLimit('super_likes', 1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            +
                          </button>
                        </div>
                        <span className="text-sm text-[#F5F7FF] min-w-[170px] text-right tabular-nums">
                          {profileLimits?.superLikesRemaining ?? 0} / {profileLimits?.superLikesTotal ?? 0} remaining
                        </span>
                      </div>
                    </div>
                    <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                      <div
                        className="h-full rounded-full bg-[#4CC9F0]"
                        style={{
                          width: profileLimits
                            ? `${Math.max(
                                0,
                                Math.min(100, (profileLimits.superLikesRemaining / profileLimits.superLikesTotal) * 100),
                              )}%`
                            : '0%',
                        }}
                      />
                    </div>
                  </div>
                  <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm text-[#A7B1C8]">Boosts</span>
                      <div className="flex items-center justify-end gap-2 min-w-[240px]">
                        <div className="flex items-center justify-end gap-2 w-[60px] translate-x-2">
                          <button
                            onClick={() => adjustProfileLimit('boosts', -1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            -
                          </button>
                          <button
                            onClick={() => adjustProfileLimit('boosts', 1)}
                            disabled={!profileLimitsEditMode}
                            className={limitAdjustButtonClass}
                          >
                            +
                          </button>
                        </div>
                        <span className="text-sm text-[#F5F7FF] min-w-[170px] text-right tabular-nums">
                          {profileLimits?.boostsRemaining ?? 0} / {profileLimits?.boostsTotal ?? 0} remaining
                        </span>
                      </div>
                    </div>
                    <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                      <div
                        className="h-full rounded-full bg-[#2DD4A8]"
                        style={{
                          width: profileLimits
                            ? `${Math.max(
                                0,
                                Math.min(100, (profileLimits.boostsRemaining / profileLimits.boostsTotal) * 100),
                              )}%`
                            : '0%',
                        }}
                      />
                    </div>
                  </div>
                </>
              )}

              {profileActiveTab === 'moderation' && (
                <>
                  <div className="flex items-center justify-between p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-[rgba(45,212,168,0.15)] flex items-center justify-center">
                        <Shield className="w-5 h-5 text-[#2DD4A8]" />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-[#F5F7FF]">Trust Score</p>
                        <p className="text-xs text-[#A7B1C8]">Based on profile history</p>
                      </div>
                    </div>
                    <span
                      className={cn(
                        'text-2xl font-bold',
                        profileTrustScore >= 90
                          ? 'text-[#2DD4A8]'
                          : profileTrustScore >= 80
                            ? 'text-[#FFD166]'
                            : 'text-[#FF6B6B]',
                      )}
                    >
                      {profileTrustScore}
                    </span>
                  </div>

                  <div className="p-4 rounded-xl bg-[rgba(255,107,107,0.05)] border border-[rgba(255,107,107,0.2)]">
                    <div className="flex items-center gap-2 mb-3">
                      <AlertTriangle className="w-4 h-4 text-[#FF6B6B]" />
                      <span className="text-sm font-medium text-[#FF6B6B]">Reports</span>
                    </div>
                    <p className="text-sm text-[#A7B1C8]">No active violations in this profile snapshot.</p>
                  </div>
                </>
              )}
            </div>

            {profileActiveTab === 'limits' && (
              <div className="p-4 border-t border-[rgba(123,97,255,0.12)]">
                <button
                  onClick={handleProfileEditLimits}
                  disabled={!canChangeLimits}
                  className="w-full btn-secondary flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Edit className="w-4 h-4" />
                  {profileLimitsEditMode ? 'Save' : 'Edit Limits'}
                </button>
              </div>
            )}
          </div>
        </div>
      , document.body)}

      {typeof document !== 'undefined' && profileViewer && profileActiveInteraction && createPortal(
        <div
          className="fixed inset-0 z-[96] flex items-center justify-center p-4"
          onClick={() => setProfileActiveInteraction(null)}
        >
          <div className="absolute inset-0 bg-black/55 backdrop-blur-[2px]" />
          <div
            className="relative w-full max-w-md glass-panel max-h-[72vh] overflow-hidden flex flex-col"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
              <div>
                <p className="text-sm text-[#A7B1C8]">{profileInteractionTitle}</p>
                <p className="text-xs text-[#A7B1C8]">{profileInteractionProfiles.length} profiles</p>
              </div>
              <button
                onClick={() => setProfileActiveInteraction(null)}
                className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="p-3 space-y-2 overflow-y-auto scrollbar-thin">
              {profileInteractionProfiles.map((profile) => (
                <button
                  key={`${profileActiveInteraction}_${profile.id}`}
                  onClick={() => {
                    if (!profileViewer) {
                      return;
                    }
                    setProfileActiveInteraction(null);
                    openProfileViewer(profile, profileViewer.contextCase);
                  }}
                  className="w-full text-left flex items-center gap-3 p-2 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)] hover:border-[rgba(123,97,255,0.35)] transition-colors"
                >
                  {profile.avatar ? (
                    <img
                      src={profile.avatar}
                      alt={profile.name}
                      className="w-11 h-11 rounded-xl border border-[rgba(123,97,255,0.25)] object-cover"
                    />
                  ) : (
                    <div className="w-11 h-11 rounded-xl border border-[rgba(123,97,255,0.25)] bg-[rgba(123,97,255,0.12)] flex items-center justify-center text-[#B7A9FF]">
                      {profile.name.slice(0, 1).toUpperCase()}
                    </div>
                  )}
                  <div>
                    <p className="text-sm font-medium text-[#F5F7FF]">
                      {profile.name}, {profile.age ?? 'N/A'}
                    </p>
                    <p className="text-xs text-[#A7B1C8]">{profile.username ?? 'No username'}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      , document.body)}

      {typeof document !== 'undefined' && viewerOpen && viewerPhotos.length > 0 && createPortal(
        <div
          className="fixed inset-0 z-[100] bg-[rgba(6,8,14,0.82)] backdrop-blur-sm flex items-center justify-center p-4"
          onClick={closeViewer}
        >
          <div
            className="relative w-full max-w-4xl"
            onClick={(event) => event.stopPropagation()}
          >
            <button
              className="absolute -top-10 right-0 px-3 py-2 rounded-lg bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.2)] text-[#F5F7FF] text-sm hover:bg-[rgba(123,97,255,0.12)]"
              onClick={closeViewer}
            >
              Close (Esc)
            </button>

            <div className="relative rounded-2xl overflow-hidden border border-[rgba(123,97,255,0.2)] bg-[rgba(14,19,32,0.6)]">
              <img
                src={viewerPhotos[viewerIndex]}
                alt="Preview"
                className="w-full max-h-[78vh] object-contain"
              />

              {viewerPhotos.length > 1 && (
                <button
                  onClick={prevViewer}
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-[rgba(14,19,32,0.75)] border border-[rgba(123,97,255,0.2)] flex items-center justify-center text-[#F5F7FF] hover:bg-[rgba(123,97,255,0.14)]"
                >
                  <ChevronLeft className="w-5 h-5" />
                </button>
              )}

              {viewerPhotos.length > 1 && (
                <button
                  onClick={nextViewer}
                  className="absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-[rgba(14,19,32,0.75)] border border-[rgba(123,97,255,0.2)] flex items-center justify-center text-[#F5F7FF] hover:bg-[rgba(123,97,255,0.14)]"
                >
                  <ChevronRight className="w-5 h-5" />
                </button>
              )}

              {viewerPhotos.length > 1 && (
                <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex items-center gap-2">
                  {viewerPhotos.map((_, index) => (
                    <button
                      key={`dot_${index}`}
                      onClick={() => setViewerIndex(index)}
                      className={cn(
                        'w-2.5 h-2.5 rounded-full border border-[rgba(123,97,255,0.35)]',
                        index === viewerIndex ? 'bg-[rgba(123,97,255,0.9)]' : 'bg-[rgba(245,247,255,0.12)]'
                      )}
                    />
                  ))}
                </div>
              )}
            </div>

            <div className="mt-3 flex items-center justify-between text-xs text-[#A7B1C8]">
              <span>
                {viewerIndex + 1} / {viewerPhotos.length}
              </span>
              <span>Use ← → to navigate</span>
            </div>
          </div>
        </div>
      , document.body)}
    </div>
  );
}
