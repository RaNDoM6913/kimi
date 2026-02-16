import { useEffect, useState, type ReactNode } from 'react';
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
  Shield
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

type ModerationCaseType = 'onboarding' | 'report' | 'support';
type ModerationPriority = 'low' | 'med' | 'high';
type ModerationStatus = 'new' | 'in_review' | 'waiting' | 'escalated' | 'done';

type ReportSubType = 'photo' | 'bio' | 'message';
type OnboardingSubType = 'profile' | 'photos' | 'bio' | 'age';
type SupportSubType = 'payments' | 'account' | 'bugs' | 'safety' | 'other';

type SupportMsgFrom = 'user' | 'admin' | 'system';

interface UserSummary {
  id: string;
  name: string;
  username?: string;
  avatar?: string;
  age?: number;
  zodiac?: string;
  city?: string;
  premium?: boolean;
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
  reportedBy?: string;
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
    subType: 'profile',
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
    subType: 'photos',
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
    subType: 'bio',
    status: 'waiting',
    priority: 'low',
    createdAtLabel: '33m ago',
    createdAtTs: 1760570820000,
    title: 'Onboarding bio verification: Elena P',
    preview: 'Short bio requires language policy check.',
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
    subType: 'age',
    status: 'escalated',
    priority: 'high',
    createdAtLabel: '48m ago',
    createdAtTs: 1760569920000,
    title: 'Age verification required: Lina S',
    preview: 'Age confidence check dropped below threshold.',
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
      { id: 'ob_004_h2', text: 'Escalated to senior moderation', timestampLabel: '50m ago' },
      { id: 'ob_004_h3', text: 'Waiting for final decision', timestampLabel: '48m ago' },
    ],
  },
  {
    id: 'ob_005',
    type: 'onboarding',
    subType: 'profile',
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
    subType: 'photos',
    status: 'new',
    priority: 'high',
    createdAtLabel: '1h ago',
    createdAtTs: 1760569199000,
    title: 'Onboarding photos check: Kate V',
    preview: 'One photo requires explicit-content revalidation.',
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
    subType: 'photo',
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
    reportReason: 'Explicit content',
    reportedBy: '@mila23',
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
    subType: 'bio',
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
    reportReason: 'External links in bio',
    reportedBy: '@roman_88',
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
    subType: 'message',
    status: 'escalated',
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
    reportReason: 'Harassment / abuse',
    reportedBy: '@irina_m',
    contentText: 'You are pathetic, reply now or I will spam your account.',
    history: [
      { id: 'rp_003_h1', text: 'Report received from chat screen', timestampLabel: '30m ago' },
      { id: 'rp_003_h2', text: 'Toxicity score above threshold', timestampLabel: '28m ago' },
      { id: 'rp_003_h3', text: 'Escalated to senior moderator', timestampLabel: '26m ago' },
    ],
  },
  {
    id: 'rp_004',
    type: 'report',
    subType: 'photo',
    status: 'waiting',
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
    reportReason: 'Possible underage person',
    reportedBy: '@wolf_17',
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
    subType: 'message',
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
    reportReason: 'Spam / scam invite',
    reportedBy: '@anya_t',
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
    subType: 'bio',
    status: 'done',
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
    reportReason: 'Off-topic advertising',
    reportedBy: '@mia_x',
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
    status: 'waiting',
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
    status: 'escalated',
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
      { id: 'sp_004_m3', from: 'admin', text: 'We can escalate this to the trust and safety team now.', timestampLabel: '06:49' },
      { id: 'sp_004_m4', from: 'user', text: 'Please do, I feel unsafe.', timestampLabel: '06:50' },
      { id: 'sp_004_m5', from: 'system', text: 'Emergency policy checklist attached.', timestampLabel: '06:52' },
      { id: 'sp_004_m6', from: 'admin', text: 'Escalation submitted, we will update you shortly.', timestampLabel: '06:55' },
    ],
    tags: ['Underage suspicion'],
    history: [
      { id: 'sp_004_h1', text: 'Safety ticket created', timestampLabel: '47m ago' },
      { id: 'sp_004_h2', text: 'Escalated to trust and safety', timestampLabel: '45m ago' },
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
    status: 'waiting',
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

const subTypeTabs: Record<Exclude<ModerationViewType, 'all'>, Array<{ value: string; label: string }>> = {
  report: [
    { value: 'photo', label: 'Photos' },
    { value: 'bio', label: 'Bios' },
    { value: 'message', label: 'Messages' },
  ],
  onboarding: [
    { value: 'profile', label: 'Profiles' },
    { value: 'photos', label: 'Photos' },
    { value: 'bio', label: 'Bio' },
    { value: 'age', label: 'Age' },
  ],
  support: [
    { value: 'payments', label: 'Payments' },
    { value: 'account', label: 'Account' },
    { value: 'bugs', label: 'Bugs' },
    { value: 'safety', label: 'Safety' },
    { value: 'other', label: 'Other' },
  ],
};

const priorityClass: Record<ModerationPriority, string> = {
  high: 'bg-[rgba(255,107,107,0.15)] text-[#FF6B6B]',
  med: 'bg-[rgba(255,209,102,0.15)] text-[#FFD166]',
  low: 'bg-[rgba(123,97,255,0.15)] text-[#B7A9FF]',
};

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
        (selectedType !== 'report' && selectedType !== 'support') ||
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
  | 'remove_content'
  | 'warn'
  | 'ban'
  | 'resolve'
  | 'request_info'
  | 'escalate';

const actionPermissions = {
  approve: ADMIN_PERMISSIONS.approve_profiles,
  reject: ADMIN_PERMISSIONS.reject_profiles,
  request_changes: ADMIN_PERMISSIONS.moderate_profiles,
  dismiss: ADMIN_PERMISSIONS.reject_profiles,
  remove_content: ADMIN_PERMISSIONS.reject_profiles,
  warn: ADMIN_PERMISSIONS.moderate_profiles,
  escalate: ADMIN_PERMISSIONS.moderate_profiles,
  ban: ADMIN_PERMISSIONS.ban_users,
  resolve: ADMIN_PERMISSIONS.moderate_profiles,
  request_info: ADMIN_PERMISSIONS.moderate_profiles,
} as const;

const actionMeta: Record<ModerationAction, { label: string; icon: ReactNode; tone: 'primary' | 'danger' | 'warn' | 'neutral' }> = {
  approve: { label: 'Approve', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  reject: { label: 'Reject', icon: <X className="w-4 h-4" />, tone: 'danger' },
  request_changes: { label: 'Request changes', icon: <MessageSquare className="w-4 h-4" />, tone: 'neutral' },
  dismiss: { label: 'Dismiss', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  remove_content: { label: 'Remove content', icon: <X className="w-4 h-4" />, tone: 'danger' },
  warn: { label: 'Warn', icon: <AlertTriangle className="w-4 h-4" />, tone: 'warn' },
  ban: { label: 'Ban', icon: <Ban className="w-4 h-4" />, tone: 'danger' },
  resolve: { label: 'Resolve', icon: <Check className="w-4 h-4" />, tone: 'primary' },
  request_info: { label: 'Request info', icon: <MessageSquare className="w-4 h-4" />, tone: 'neutral' },
  escalate: { label: 'Escalate', icon: <AlertTriangle className="w-4 h-4" />, tone: 'warn' },
};

const actionsByType: Record<ModerationCaseType, ModerationAction[]> = {
  onboarding: ['approve', 'reject', 'request_changes', 'escalate'],
  report: ['dismiss', 'remove_content', 'warn', 'ban', 'escalate'],
  support: ['resolve', 'request_info', 'escalate'],
};

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
  const activeSubTabs =
    selectedType === 'report' || selectedType === 'support'
      ? subTypeTabs[selectedType]
      : [];

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
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-[rgba(123,97,255,0.15)] text-[#B7A9FF] uppercase">
                          {caseItem.type}/{caseItem.subType}
                        </span>
                        <span className={cn('px-1.5 py-0.5 rounded text-[10px] uppercase', priorityClass[caseItem.priority])}>
                          {caseItem.priority}
                        </span>
                        {caseItem.type === 'support' && caseItem.slaLabel && (
                          <span className="px-1.5 py-0.5 rounded text-[10px] bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]">
                            {caseItem.slaLabel}
                          </span>
                        )}
                        {(caseItem.tags ?? []).slice(0, 2).map((tag) => (
                          <span
                            key={`${caseItem.id}_${tag}`}
                            className="px-1.5 py-0.5 rounded text-[10px] border border-[rgba(123,97,255,0.2)] text-[#A7B1C8]"
                          >
                            {tag}
                          </span>
                        ))}
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
}: {
  caseType: ModerationCaseType;
  onAction: (action: ModerationAction) => void;
  canAction: (action: ModerationAction) => boolean;
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
    <div className="p-4 border-t border-[rgba(123,97,255,0.12)] flex flex-wrap gap-2">
      {actions.map((action) => {
        const meta = actionMeta[action];
        const isPrimaryTone = meta.tone === 'primary' || meta.tone === 'danger';

        return (
          <button
            key={action}
            onClick={() => onAction(action)}
            disabled={!canAction(action)}
            className={cn(
              isPrimaryTone ? 'flex-1' : '',
              'px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed',
              buttonClassByTone[meta.tone]
            )}
          >
            {meta.icon}
            {meta.label}
          </button>
        );
      })}
    </div>
  );
}

function DetailPanel({
  caseItem,
  onAction,
  canAction,
  onSendSupportReply,
  onOpenViewer,
}: {
  caseItem: ModerationCase | null;
  onAction: (action: ModerationAction) => void;
  canAction: (action: ModerationAction) => boolean;
  onSendSupportReply: (caseId: string, text: string) => void;
  onOpenViewer: (photos: string[], startIndex: number) => void;
}) {
  type DetailTab = 'evidence' | 'user' | 'history' | 'notes';

  const [activeTab, setActiveTab] = useState<DetailTab>('evidence');
  const [notesByCaseId, setNotesByCaseId] = useState<Record<string, string>>({});
  const [noteTagsByCaseId, setNoteTagsByCaseId] = useState<Record<string, string[]>>({});
  const [replyDraftByCaseId, setReplyDraftByCaseId] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!caseItem) {
      return;
    }

    setActiveTab('evidence');
    setNoteTagsByCaseId((prev) => {
      if (prev[caseItem.id]) {
        return prev;
      }
      return { ...prev, [caseItem.id]: caseItem.tags ?? [] };
    });
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
  const selectedNoteTags = noteTagsByCaseId[caseItem.id] ?? caseItem.tags ?? [];
  const noteTagPool = Array.from(
    new Set([...(caseItem.tags ?? []), 'NSFW risk', 'Underage suspicion', 'Link detected'])
  );

  const toggleNoteTag = (tag: string) => {
    setNoteTagsByCaseId((prev) => {
      const currentTags = prev[caseItem.id] ?? caseItem.tags ?? [];
      const nextTags = currentTags.includes(tag)
        ? currentTags.filter((currentTag) => currentTag !== tag)
        : [...currentTags, tag];
      return { ...prev, [caseItem.id]: nextTags };
    });
  };

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
                ['Birthday', caseItem.onboarding.birthday],
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
      return (
        <div className="space-y-4">
          {caseItem.contentMediaUrl && (
            <img
              src={caseItem.contentMediaUrl}
              alt="Reported content"
              className="max-w-md rounded-xl border border-[rgba(123,97,255,0.2)]"
            />
          )}
          {caseItem.contentText && (
            <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
              <p className="text-sm text-[#F5F7FF]">{caseItem.contentText}</p>
            </div>
          )}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <span className="text-[#A7B1C8]">Reported by:</span>
              <span className="text-[#F5F7FF]">{caseItem.reportedBy ?? 'Unknown'}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <span className="text-[#A7B1C8]">Reason:</span>
              <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(255,107,107,0.15)] text-[#FF6B6B]">
                {caseItem.reportReason ?? 'N/A'}
              </span>
            </div>
          </div>
          <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)] space-y-1">
            <p className="text-xs uppercase tracking-wide text-[#A7B1C8]">User Summary</p>
            <p className="text-sm text-[#F5F7FF]">{caseItem.user.name} ({caseItem.user.id})</p>
            <p className="text-xs text-[#A7B1C8]">{caseItem.user.username ?? 'No username'} • {caseItem.user.city ?? 'Unknown city'}</p>
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
        <div className="grid grid-cols-2 gap-2">
          {[
            ['Category', caseItem.category ?? caseItem.subType],
            ['SLA', caseItem.slaLabel ?? 'N/A'],
            ['Priority', caseItem.priority],
            ['Status', caseItem.status],
          ].map(([label, value]) => (
            <div
              key={`${caseItem.id}_support_${label}`}
              className="p-3 rounded-lg bg-[rgba(14,19,32,0.45)] border border-[rgba(123,97,255,0.1)]"
            >
              <p className="text-[10px] uppercase tracking-wide text-[#A7B1C8]">{label}</p>
              <p className="text-sm text-[#F5F7FF] capitalize">{value}</p>
            </div>
          ))}
        </div>

        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
          <p className="text-xs uppercase tracking-wide text-[#A7B1C8] mb-1">Description</p>
          <p className="text-sm text-[#F5F7FF]">{supportDescription}</p>
        </div>

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
    <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
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
    </div>
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
      <div className="space-y-2">
        <p className="text-xs uppercase tracking-wide text-[#A7B1C8]">Tags</p>
        <div className="flex flex-wrap gap-2">
          {noteTagPool.map((tag) => {
            const active = selectedNoteTags.includes(tag);
            return (
              <button
                key={`${caseItem.id}_note_tag_${tag}`}
                onClick={() => toggleNoteTag(tag)}
                className={cn(
                  'px-2 py-0.5 rounded-full text-xs border transition-colors',
                  active
                    ? 'bg-[rgba(123,97,255,0.18)] border-[rgba(123,97,255,0.35)] text-[#B7A9FF]'
                    : 'border-[rgba(123,97,255,0.2)] text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)]'
                )}
              >
                {tag}
              </button>
            );
          })}
        </div>
      </div>
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
            <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(123,97,255,0.15)] text-[#B7A9FF] uppercase">
              {caseItem.type}/{caseItem.subType}
            </span>
            <span className={cn('px-2 py-0.5 rounded-full text-xs uppercase', priorityClass[caseItem.priority])}>
              {caseItem.priority}
            </span>
            {caseItem.slaLabel && (
              <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]">
                {caseItem.slaLabel}
              </span>
            )}
            {(caseItem.tags ?? []).map((tag) => (
              <span
                key={`${caseItem.id}_tag_${tag}`}
                className="px-2 py-0.5 rounded-full text-xs border border-[rgba(123,97,255,0.2)] text-[#A7B1C8]"
              >
                {tag}
              </span>
            ))}
          </div>

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

          {activeTab === 'evidence' && renderEvidence()}
          {activeTab === 'user' && renderUser()}
          {activeTab === 'history' && renderHistory()}
          {activeTab === 'notes' && renderNotes()}
        </div>

        <ActionBar caseType={caseItem.type} onAction={onAction} canAction={canAction} />
      </div>
    </div>
  );
}

export function ModerationPage() {
  const [cases, setCases] = useState<ModerationCase[]>(unifiedCases);
  const [selectedType, setSelectedType] = useState<ModerationViewType>('all');
  const [selectedSubType, setSelectedSubType] = useState('');
  const [selectedCaseId, setSelectedCaseId] = useState<string | null>(unifiedCases[0]?.id ?? null);
  const [query, setQuery] = useState('');
  const [viewerOpen, setViewerOpen] = useState(false);
  const [viewerPhotos, setViewerPhotos] = useState<string[]>([]);
  const [viewerIndex, setViewerIndex] = useState(0);
  const { hasPermission, role } = usePermissions();
  const totalPending = cases.length;

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

  const canAction = (action: ModerationAction) => hasPermission(actionPermissions[action]);

  const handleAction = (action: ModerationAction) => {
    if (!selectedCase || !canAction(action)) {
      return;
    }

    logAdminAction(
      `moderation_${action}`,
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );

    setCases((prevCases) => {
      const nextCases = prevCases.filter((caseItem) => caseItem.id !== selectedCase.id);
      const nextFilteredCases = filterCases({
        cases: nextCases,
        selectedType,
        selectedSubType,
        query,
      });
      const nextSelectedCaseId = nextFilteredCases[0]?.id ?? null;

      queueMicrotask(() => {
        setSelectedCaseId(nextSelectedCaseId);
      });

      return nextCases;
    });
  };

  const handleSelectType = (type: ModerationViewType) => {
    setSelectedType(type);
    setSelectedSubType('');
  };

  const handleSendSupportReply = (caseId: string, text: string) => {
    const message = text.trim();
    if (!message) {
      return;
    }

    const targetCase = cases.find((caseItem) => caseItem.id === caseId);
    if (!targetCase) {
      return;
    }

    console.log({
      caseId: targetCase.id,
      userId: targetCase.user.id,
      username: targetCase.user.username,
      message,
    });

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
          onSendSupportReply={handleSendSupportReply}
          onOpenViewer={openViewer}
        />
      </div>

      {viewerOpen && viewerPhotos.length > 0 && (
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
      )}
    </div>
  );
}
