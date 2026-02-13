import type { 
  KPIData, 
  User, 
  Alert, 
  ChartDataPoint, 
  RetentionCohort, 
  Experiment, 
  Role, 
  Permission,
  ModerationItem,
  AdCampaign,
  SystemMetric,
  HeatmapCell
} from '@/types';

// Overview KPIs (DAU, MAU, Registrations, Matches, Revenue, Active Subscriptions — 6 blocks, same size)
export const overviewKPIs: KPIData[] = [
  { title: 'DAU', value: '84.2K', trend: 6.4, sparklineData: [65, 68, 72, 70, 75, 78, 82, 80, 84] },
  { title: 'MAU', value: '1.24M', trend: 6.4, sparklineData: [1100, 1120, 1150, 1140, 1180, 1200, 1220, 1210, 1240] },
  { title: 'Registrations Today', value: '+3,847', trend: 12.0, sparklineData: [120, 135, 128, 145, 152, 148, 165, 172, 180] },
  { title: 'Matches Today', value: '28,491', trend: 4.2, sparklineData: [22000, 23500, 24100, 23800, 25200, 26100, 25800, 27200, 28491] },
  { title: 'Revenue Today', value: '$42,180', trend: 8.1, sparklineData: [32000, 33500, 34100, 33800, 36200, 37100, 38800, 40200, 42180] },
  { title: 'Active Subscriptions', value: '182,400', trend: 2.3, sparklineData: [168000, 171000, 173000, 172000, 175000, 177000, 179000, 180000, 182400] },
];

// Growth chart data
export const growthChartData: ChartDataPoint[] = [
  { name: 'Jan', value: 42000, value2: 38000 },
  { name: 'Feb', value: 48000, value2: 42000 },
  { name: 'Mar', value: 55000, value2: 48000 },
  { name: 'Apr', value: 62000, value2: 54000 },
  { name: 'May', value: 68000, value2: 59000 },
  { name: 'Jun', value: 75000, value2: 65000 },
  { name: 'Jul', value: 82000, value2: 71000 },
  { name: 'Aug', value: 79000, value2: 68000 },
  { name: 'Sep', value: 86000, value2: 74000 },
  { name: 'Oct', value: 92000, value2: 78000 },
  { name: 'Nov', value: 98000, value2: 82000 },
  { name: 'Dec', value: 105000, value2: 88000 },
];

// Revenue trend chart (weekly + monthly for period filter)
export const revenueTrendChartDataWeekly: ChartDataPoint[] = [
  { name: 'Mon', value: 38500 },
  { name: 'Tue', value: 39200 },
  { name: 'Wed', value: 40100 },
  { name: 'Thu', value: 39800 },
  { name: 'Fri', value: 41500 },
  { name: 'Sat', value: 43800 },
  { name: 'Sun', value: 42180 },
];

export const revenueTrendChartDataMonthly: ChartDataPoint[] = [
  { name: 'Jan', value: 118000 },
  { name: 'Feb', value: 125000 },
  { name: 'Mar', value: 132000 },
  { name: 'Apr', value: 128000 },
  { name: 'May', value: 138000 },
  { name: 'Jun', value: 142000 },
  { name: 'Jul', value: 148000 },
  { name: 'Aug', value: 145000 },
  { name: 'Sep', value: 152000 },
  { name: 'Oct', value: 158000 },
  { name: 'Nov', value: 162000 },
  { name: 'Dec', value: 168000 },
];

// Alerts data
export const alerts: Alert[] = [
  { id: '1', type: 'warning', title: 'High Report Volume', message: 'Photo reports are 23% above normal in the last hour', timestamp: '5 min ago', isRead: false },
  { id: '2', type: 'error', title: 'API Latency Spike', message: 'p99 latency exceeded 500ms on match service', timestamp: '12 min ago', isRead: false },
  { id: '3', type: 'info', title: 'New Feature Deployed', message: 'Smart Recommendations v2.3 is now live', timestamp: '1 hour ago', isRead: true },
  { id: '4', type: 'success', title: 'Revenue Milestone', message: 'Daily revenue target achieved 6 hours early', timestamp: '2 hours ago', isRead: true },
  { id: '5', type: 'warning', title: 'Trust Score Drop', message: 'Average trust score decreased by 2.3 points', timestamp: '3 hours ago', isRead: false },
];

// Users data
export const users: User[] = [
  { id: 'USR-7842', name: 'Emma Wilson', handle: '@emma_w', avatar: 'https://i.pravatar.cc/150?u=emma', email: 'emma@example.com', joined: '2024-01-15', lastActive: '2 min ago', location: 'New York, USA', gender: 'Female', age: 26, trustScore: 94, status: 'online', isPremium: true, subscriptionTier: 'Gold', bio: 'Coffee lover & travel enthusiast', matches: 42, likes: 128 },
  { id: 'USR-7841', name: 'James Chen', handle: '@jamesc', avatar: 'https://i.pravatar.cc/150?u=james', email: 'james@example.com', joined: '2024-02-03', lastActive: '5 min ago', location: 'San Francisco, USA', gender: 'Male', age: 29, trustScore: 88, status: 'online', isPremium: true, subscriptionTier: 'Plus', bio: 'Tech entrepreneur, hiking addict', matches: 38, likes: 95 },
  { id: 'USR-7840', name: 'Sofia Rodriguez', handle: '@sofia_r', avatar: 'https://i.pravatar.cc/150?u=sofia', email: 'sofia@example.com', joined: '2024-01-28', lastActive: '15 min ago', location: 'Miami, USA', gender: 'Female', age: 24, trustScore: 91, status: 'away', isPremium: false, bio: 'Dancer & art lover', matches: 31, likes: 87 },
  { id: 'USR-7839', name: 'Michael Park', handle: '@mpark', avatar: 'https://i.pravatar.cc/150?u=michael', email: 'michael@example.com', joined: '2023-11-20', lastActive: '1 hour ago', location: 'Seattle, USA', gender: 'Male', age: 31, trustScore: 96, status: 'offline', isPremium: true, subscriptionTier: 'Gold', bio: 'Software engineer, dog dad', matches: 56, likes: 203 },
  { id: 'USR-7838', name: 'Isabella Martinez', handle: '@bella_m', avatar: 'https://i.pravatar.cc/150?u=isabella', email: 'isabella@example.com', joined: '2024-02-10', lastActive: '30 min ago', location: 'Austin, USA', gender: 'Female', age: 27, trustScore: 85, status: 'online', isPremium: false, bio: 'Foodie & yoga instructor', matches: 22, likes: 64 },
  { id: 'USR-7837', name: 'William Thompson', handle: '@will_t', avatar: 'https://i.pravatar.cc/150?u=william', email: 'william@example.com', joined: '2023-12-05', lastActive: '2 hours ago', location: 'Chicago, USA', gender: 'Male', age: 33, trustScore: 78, status: 'offline', isPremium: true, subscriptionTier: 'Plus', bio: 'Musician & coffee snob', matches: 29, likes: 71 },
  { id: 'USR-7836', name: 'Olivia Kim', handle: '@oliviak', avatar: 'https://i.pravatar.cc/150?u=olivia', email: 'olivia@example.com', joined: '2024-01-08', lastActive: '45 min ago', location: 'Los Angeles, USA', gender: 'Female', age: 25, trustScore: 92, status: 'online', isPremium: true, subscriptionTier: 'Gold', bio: 'Actress & fitness enthusiast', matches: 48, likes: 156 },
  { id: 'USR-7835', name: 'Daniel Brown', handle: '@dan_b', avatar: 'https://i.pravatar.cc/150?u=daniel', email: 'daniel@example.com', joined: '2024-02-14', lastActive: '10 min ago', location: 'Denver, USA', gender: 'Male', age: 28, trustScore: 82, status: 'away', isPremium: false, bio: 'Photographer & traveler', matches: 18, likes: 52 },
  { id: 'USR-7834', name: 'Ava Johnson', handle: '@ava_j', avatar: 'https://i.pravatar.cc/150?u=ava', email: 'ava@example.com', joined: '2023-10-18', lastActive: '3 hours ago', location: 'Boston, USA', gender: 'Female', age: 30, trustScore: 95, status: 'offline', isPremium: true, subscriptionTier: 'Gold', bio: 'Doctor & book lover', matches: 61, likes: 234 },
  { id: 'USR-7833', name: 'Lucas Garcia', handle: '@lucas_g', avatar: 'https://i.pravatar.cc/150?u=lucas', email: 'lucas@example.com', joined: '2024-01-22', lastActive: '20 min ago', location: 'Phoenix, USA', gender: 'Male', age: 26, trustScore: 87, status: 'online', isPremium: false, bio: 'Entrepreneur & gym rat', matches: 35, likes: 89 },
];

// Session duration data
export const sessionDurationData: ChartDataPoint[] = [
  { name: '00:00', value: 4.2 },
  { name: '03:00', value: 3.8 },
  { name: '06:00', value: 5.1 },
  { name: '09:00', value: 7.8 },
  { name: '12:00', value: 9.2 },
  { name: '15:00', value: 8.6 },
  { name: '18:00', value: 10.2 },
  { name: '21:00', value: 11.5 },
];

// Time per tab data
export const timePerTabData: ChartDataPoint[] = [
  { name: 'Feed', value: 250 },
  { name: 'Likes', value: 140 },
  { name: 'Ideas', value: 100 },
  { name: 'Profile', value: 32 },
];

// Retention cohorts data
export const retentionCohorts: RetentionCohort[] = [
  { week: 'Week 1', day0: 100, day1: 72, day3: 68, day7: 62, day14: 55, day30: 48 },
  { week: 'Week 2', day0: 100, day1: 70, day3: 65, day7: 58, day14: 51, day30: 44 },
  { week: 'Week 3', day0: 100, day1: 74, day3: 70, day7: 64, day14: 57, day30: 50 },
  { week: 'Week 4', day0: 100, day1: 68, day3: 62, day7: 55, day14: 48, day30: 42 },
  { week: 'Week 5', day0: 100, day1: 75, day3: 71, day7: 66, day14: 60, day30: 54 },
  { week: 'Week 6', day0: 100, day1: 71, day3: 67, day7: 61, day14: 54, day30: 47 },
];

// Match rate analytics data
export const matchRateData: ChartDataPoint[] = [
  { name: 'Mon', value: 12.5 },
  { name: 'Tue', value: 13.8 },
  { name: 'Wed', value: 14.2 },
  { name: 'Thu', value: 13.5 },
  { name: 'Fri', value: 15.1 },
  { name: 'Sat', value: 16.8 },
  { name: 'Sun', value: 15.4 },
];

// GMV chart data
export const gmvData: ChartDataPoint[] = [
  { name: 'Jan', value: 28000, value2: 32000 },
  { name: 'Feb', value: 32000, value2: 36000 },
  { name: 'Mar', value: 38000, value2: 42000 },
  { name: 'Apr', value: 42000, value2: 48000 },
  { name: 'May', value: 48000, value2: 54000 },
  { name: 'Jun', value: 52000, value2: 58000 },
  { name: 'Jul', value: 58000, value2: 64000 },
  { name: 'Aug', value: 62000, value2: 68000 },
  { name: 'Sep', value: 68000, value2: 74000 },
  { name: 'Oct', value: 72000, value2: 80000 },
  { name: 'Nov', value: 78000, value2: 86000 },
  { name: 'Dec', value: 84000, value2: 92000 },
];

// Subscriptions growth data
export const subscriptionData: ChartDataPoint[] = [
  { name: 'Jan', value: 12000, value2: 8000, value3: 4000 },
  { name: 'Feb', value: 13500, value2: 9000, value3: 4500 },
  { name: 'Mar', value: 15200, value2: 10200, value3: 5000 },
  { name: 'Apr', value: 16800, value2: 11400, value3: 5400 },
  { name: 'May', value: 18500, value2: 12600, value3: 5900 },
  { name: 'Jun', value: 20200, value2: 13800, value3: 6400 },
  { name: 'Jul', value: 22100, value2: 15100, value3: 7000 },
  { name: 'Aug', value: 24000, value2: 16400, value3: 7600 },
  { name: 'Sep', value: 26100, value2: 17800, value3: 8300 },
  { name: 'Oct', value: 28300, value2: 19300, value3: 9000 },
  { name: 'Nov', value: 30600, value2: 20900, value3: 9700 },
  { name: 'Dec', value: 33000, value2: 22600, value3: 10400 },
];

// Microtransactions data
export const microtransactionData: ChartDataPoint[] = [
  { name: 'Mon', value: 4200 },
  { name: 'Tue', value: 4800 },
  { name: 'Wed', value: 5100 },
  { name: 'Thu', value: 4900 },
  { name: 'Fri', value: 5800 },
  { name: 'Sat', value: 7200 },
  { name: 'Sun', value: 6500 },
];

// Purchase heatmap data
export const purchaseHeatmapData: HeatmapCell[] = [
  // Sunday (0)
  { hour: 0, day: 0, value: 15 }, { hour: 6, day: 0, value: 25 }, { hour: 12, day: 0, value: 45 }, { hour: 18, day: 0, value: 65 }, { hour: 22, day: 0, value: 55 },
  // Monday (1)
  { hour: 0, day: 1, value: 10 }, { hour: 6, day: 1, value: 20 }, { hour: 12, day: 1, value: 35 }, { hour: 18, day: 1, value: 50 }, { hour: 22, day: 1, value: 40 },
  // Tuesday (2)
  { hour: 0, day: 2, value: 12 }, { hour: 6, day: 2, value: 22 }, { hour: 12, day: 2, value: 38 }, { hour: 18, day: 2, value: 55 }, { hour: 22, day: 2, value: 42 },
  // Wednesday (3)
  { hour: 0, day: 3, value: 14 }, { hour: 6, day: 3, value: 24 }, { hour: 12, day: 3, value: 42 }, { hour: 18, day: 3, value: 58 }, { hour: 22, day: 3, value: 48 },
  // Thursday (4)
  { hour: 0, day: 4, value: 16 }, { hour: 6, day: 4, value: 28 }, { hour: 12, day: 4, value: 48 }, { hour: 18, day: 4, value: 68 }, { hour: 22, day: 4, value: 58 },
  // Friday (5)
  { hour: 0, day: 5, value: 20 }, { hour: 6, day: 5, value: 32 }, { hour: 12, day: 5, value: 55 }, { hour: 18, day: 5, value: 78 }, { hour: 22, day: 5, value: 72 },
  // Saturday (6)
  { hour: 0, day: 6, value: 25 }, { hour: 6, day: 6, value: 38 }, { hour: 12, day: 6, value: 62 }, { hour: 18, day: 6, value: 85 }, { hour: 22, day: 6, value: 75 },
];

// Purchases by gender data
export const purchasesByGenderData = [
  { name: 'Male', value: 58, fill: '#7B61FF' },
  { name: 'Female', value: 40, fill: '#4CC9F0' },
  { name: 'Other', value: 2, fill: '#2DD4A8' },
];

// Purchases by region data
export const purchasesByRegionData = [
  { name: 'North America', value: 42, fill: '#7B61FF' },
  { name: 'Europe', value: 28, fill: '#4CC9F0' },
  { name: 'Asia', value: 18, fill: '#2DD4A8' },
  { name: 'Other', value: 12, fill: '#FFD166' },
];

// Ads KPIs
export const adsKPIs: KPIData[] = [
  { title: 'Impressions', value: '1.84M', trend: 8.5 },
  { title: 'Clicks', value: '46,200', trend: 12.3 },
  { title: 'CTR', value: '2.51%', trend: 3.2 },
  { title: 'Conversion', value: '4.8%', trend: 5.7 },
];

// Ad revenue data
export const adRevenueData: ChartDataPoint[] = [
  { name: 'Mon', value: 6200 },
  { name: 'Tue', value: 6800 },
  { name: 'Wed', value: 7100 },
  { name: 'Thu', value: 6900 },
  { name: 'Fri', value: 7800 },
  { name: 'Sat', value: 9200 },
  { name: 'Sun', value: 8540 },
];

// Audience segmentation data
export const audienceData: ChartDataPoint[] = [
  { name: '18-24 F', value: 18 },
  { name: '18-24 M', value: 22 },
  { name: '25-34 F', value: 24 },
  { name: '25-34 M', value: 28 },
  { name: '35-44 F', value: 12 },
  { name: '35-44 M', value: 15 },
  { name: '45+ F', value: 8 },
  { name: '45+ M', value: 10 },
];

// Ad campaigns
export const adCampaigns: AdCampaign[] = [
  { id: 'CAMP-001', name: 'Valentine\'s Special', status: 'active', impressions: 452000, clicks: 12400, ctr: 2.74, conversions: 892, revenue: 12500, spend: 4200, roas: 2.98 },
  { id: 'CAMP-002', name: 'Spring Dating', status: 'active', impressions: 328000, clicks: 8900, ctr: 2.71, conversions: 654, revenue: 9200, spend: 3100, roas: 2.97 },
  { id: 'CAMP-003', name: 'Premium Upgrade', status: 'paused', impressions: 215000, clicks: 5200, ctr: 2.42, conversions: 423, revenue: 6800, spend: 2800, roas: 2.43 },
  { id: 'CAMP-004', name: 'Weekend Boost', status: 'ended', impressions: 845000, clicks: 20100, ctr: 2.38, conversions: 1842, revenue: 28400, spend: 9500, roas: 2.99 },
];

// Experiments data
export const experiments: Experiment[] = [
  {
    id: 'EXP-001',
    name: 'Onboarding Step 3 Copy',
    status: 'running',
    startDate: '2024-02-01',
    metric: 'Completion Rate',
    variants: [
      { id: 'control', name: 'Control', allocation: 50, conversions: 1240, conversionRate: 62 },
      { id: 'variant_a', name: 'Shorter Copy', allocation: 50, conversions: 1380, conversionRate: 69 },
    ],
  },
  {
    id: 'EXP-002',
    name: 'Like Button Animation',
    status: 'running',
    startDate: '2024-02-10',
    metric: 'Like Rate',
    variants: [
      { id: 'control', name: 'Control', allocation: 50, conversions: 8500, conversionRate: 24 },
      { id: 'variant_a', name: 'Heart Burst', allocation: 50, conversions: 9200, conversionRate: 26 },
    ],
  },
  {
    id: 'EXP-003',
    name: 'Paywall Timing',
    status: 'completed',
    startDate: '2024-01-15',
    endDate: '2024-02-05',
    metric: 'Conversion Rate',
    winner: 'variant_b',
    variants: [
      { id: 'control', name: 'Day 3', allocation: 33, conversions: 420, conversionRate: 4.2 },
      { id: 'variant_a', name: 'Day 5', allocation: 33, conversions: 380, conversionRate: 3.8 },
      { id: 'variant_b', name: 'Day 7', allocation: 34, conversions: 510, conversionRate: 5.1 },
    ],
  },
  {
    id: 'EXP-004',
    name: 'Profile Photo Prompt',
    status: 'paused',
    startDate: '2024-02-08',
    metric: 'Photo Upload Rate',
    variants: [
      { id: 'control', name: 'Control', allocation: 50, conversions: 2100, conversionRate: 42 },
      { id: 'variant_a', name: 'With Examples', allocation: 50, conversions: 2080, conversionRate: 41.6 },
    ],
  },
];

// System metrics
export const systemMetrics: SystemMetric[] = [
  { name: 'API Latency (p99)', value: '142', unit: 'ms', trend: -5.2, data: [
    { name: '00:00', value: 120 }, { name: '04:00', value: 115 }, { name: '08:00', value: 145 },
    { name: '12:00', value: 165 }, { name: '16:00', value: 155 }, { name: '20:00', value: 142 },
  ]},
  { name: 'Error Rate', value: '0.12', unit: '%', trend: -12.5, data: [
    { name: '00:00', value: 0.15 }, { name: '04:00', value: 0.12 }, { name: '08:00', value: 0.18 },
    { name: '12:00', value: 0.14 }, { name: '16:00', value: 0.11 }, { name: '20:00', value: 0.12 },
  ]},
  { name: 'Events/sec', value: '84', unit: 'K', trend: 8.4, data: [
    { name: '00:00', value: 62 }, { name: '04:00', value: 58 }, { name: '08:00', value: 78 },
    { name: '12:00', value: 92 }, { name: '16:00', value: 88 }, { name: '20:00', value: 84 },
  ]},
  { name: 'Queue Size', value: '1,240', unit: '', trend: 15.2, data: [
    { name: '00:00', value: 800 }, { name: '04:00', value: 650 }, { name: '08:00', value: 1200 },
    { name: '12:00', value: 1800 }, { name: '16:00', value: 1500 }, { name: '20:00', value: 1240 },
  ]},
  { name: 'Redis Memory', value: '78', unit: '%', trend: 2.1, data: [
    { name: '00:00', value: 72 }, { name: '04:00', value: 74 }, { name: '08:00', value: 76 },
    { name: '12:00', value: 78 }, { name: '16:00', value: 79 }, { name: '20:00', value: 78 },
  ]},
  { name: 'Rate Limit Blocks', value: '340', unit: '/h', trend: -8.5, data: [
    { name: '00:00', value: 280 }, { name: '04:00', value: 220 }, { name: '08:00', value: 420 },
    { name: '12:00', value: 480 }, { name: '16:00', value: 380 }, { name: '20:00', value: 340 },
  ]},
];

// Moderation queue
export const moderationQueue: ModerationItem[] = [
  { id: 'MOD-001', type: 'photo', content: 'User profile photo', thumbnail: 'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200&h=200&fit=crop', reportedBy: 'User #7821', reason: 'Inappropriate content', timestamp: '5 min ago', userId: 'USR-7820', userName: 'Jessica Miller', status: 'pending' },
  { id: 'MOD-002', type: 'bio', content: 'Looking for fun tonight, hit me up...', reportedBy: 'User #7819', reason: 'Spam', timestamp: '12 min ago', userId: 'USR-7818', userName: 'David Lee', status: 'pending' },
  { id: 'MOD-003', type: 'photo', content: 'Group photo at beach', thumbnail: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200&h=200&fit=crop', reportedBy: 'User #7817', reason: 'Misleading', timestamp: '18 min ago', userId: 'USR-7816', userName: 'Ryan Taylor', status: 'pending' },
  { id: 'MOD-004', type: 'message', content: 'Hey beautiful, want to meet up?', reportedBy: 'User #7815', reason: 'Harassment', timestamp: '25 min ago', userId: 'USR-7814', userName: 'Mark Wilson', status: 'pending' },
  { id: 'MOD-005', type: 'photo', content: 'Mirror selfie', thumbnail: 'https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=200&h=200&fit=crop', reportedBy: 'User #7813', reason: 'Underage suspicion', timestamp: '32 min ago', userId: 'USR-7812', userName: 'Amy Chen', status: 'pending' },
  { id: 'MOD-006', type: 'bio', content: 'Check my Instagram @...', reportedBy: 'User #7811', reason: 'External promotion', timestamp: '45 min ago', userId: 'USR-7810', userName: 'Chris Brown', status: 'pending' },
];

// Permissions data
export const permissions: Permission[] = [
  { id: 'perm_users_view', name: 'View Users', description: 'Access user list and profiles', category: 'Users' },
  { id: 'perm_users_edit', name: 'Edit Users', description: 'Modify user data and settings', category: 'Users' },
  { id: 'perm_users_ban', name: 'Ban Users', description: 'Suspend or ban user accounts', category: 'Users' },
  { id: 'perm_moderation_view', name: 'View Moderation', description: 'Access moderation queue', category: 'Moderation' },
  { id: 'perm_moderation_approve', name: 'Approve Content', description: 'Approve reported content', category: 'Moderation' },
  { id: 'perm_moderation_remove', name: 'Remove Content', description: 'Remove reported content', category: 'Moderation' },
  { id: 'perm_revenue_view', name: 'View Revenue', description: 'Access revenue dashboards', category: 'Finance' },
  { id: 'perm_pricing_edit', name: 'Edit Pricing', description: 'Modify subscription pricing', category: 'Finance' },
  { id: 'perm_experiments_manage', name: 'Manage Experiments', description: 'Create and manage A/B tests', category: 'Product' },
  { id: 'perm_roles_manage', name: 'Manage Roles', description: 'Create and assign roles', category: 'Admin' },
  { id: 'perm_system_view', name: 'View System', description: 'Access system metrics', category: 'System' },
  { id: 'perm_settings_edit', name: 'Edit Settings', description: 'Modify app settings', category: 'Admin' },
];

// Roles data
export const roles: Role[] = [
  { 
    id: 'role_super_admin', 
    name: 'Super Admin', 
    description: 'Full system access', 
    userCount: 3,
    permissions: permissions 
  },
  { 
    id: 'role_product_admin', 
    name: 'Product Admin', 
    description: 'Product and feature management', 
    userCount: 8,
    permissions: permissions.filter(p => ['Product', 'Users', 'Moderation'].includes(p.category)) 
  },
  { 
    id: 'role_support_lead', 
    name: 'Support Lead', 
    description: 'User support and moderation', 
    userCount: 12,
    permissions: permissions.filter(p => ['Users', 'Moderation'].includes(p.category)) 
  },
  { 
    id: 'role_support_agent', 
    name: 'Support Agent', 
    description: 'Basic user support', 
    userCount: 24,
    permissions: permissions.filter(p => p.category === 'Users' && p.name.includes('View')) 
  },
  { 
    id: 'role_finance_viewer', 
    name: 'Finance Viewer', 
    description: 'Read-only finance access', 
    userCount: 5,
    permissions: permissions.filter(p => p.category === 'Finance' && p.name.includes('View')) 
  },
  { 
    id: 'role_read_only', 
    name: 'Read Only', 
    description: 'View-only access', 
    userCount: 7,
    permissions: permissions.filter(p => p.name.includes('View')) 
  },
];
