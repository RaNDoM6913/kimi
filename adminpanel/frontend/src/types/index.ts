export interface KPIData {
  title: string;
  value: string;
  trend: number;
  trendLabel?: string;
  sparklineData?: number[];
}

export interface User {
  id: string;
  name: string;
  handle: string;
  avatar: string;
  email: string;
  joined: string;
  lastActive: string;
  location: string;
  gender: string;
  age: number;
  trustScore: number;
  status: 'online' | 'away' | 'offline';
  isPremium: boolean;
  subscriptionTier?: string;
  bio?: string;
  photos?: string[];
  matches: number;
  likes: number;
  reports?: number;
}

export interface Alert {
  id: string;
  type: 'warning' | 'error' | 'info' | 'success';
  title: string;
  message: string;
  timestamp: string;
  isRead: boolean;
}

export interface ChartDataPoint {
  name: string;
  value: number;
  value2?: number;
  value3?: number;
}

export interface RetentionCohort {
  week: string;
  day0: number;
  day1: number;
  day3: number;
  day7: number;
  day14: number;
  day30: number;
}

export interface Experiment {
  id: string;
  name: string;
  status: 'running' | 'paused' | 'completed';
  startDate: string;
  endDate?: string;
  variants: ExperimentVariant[];
  metric: string;
  winner?: string;
}

export interface ExperimentVariant {
  id: string;
  name: string;
  allocation: number;
  conversions: number;
  conversionRate: number;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  userCount: number;
  permissions: Permission[];
}

export interface Permission {
  id: string;
  name: string;
  description: string;
  category: string;
}

export interface ModerationItem {
  id: string;
  type: 'photo' | 'bio' | 'message';
  content: string;
  thumbnail?: string;
  reportedBy: string;
  reason: string;
  timestamp: string;
  userId: string;
  userName: string;
  status: 'pending' | 'approved' | 'removed' | 'escalated';
}

export interface AdCampaign {
  id: string;
  name: string;
  status: 'active' | 'paused' | 'ended';
  impressions: number;
  clicks: number;
  ctr: number;
  conversions: number;
  revenue: number;
  spend: number;
  roas: number;
}

export interface SystemMetric {
  name: string;
  value: string;
  unit?: string;
  trend: number;
  data: ChartDataPoint[];
}

export interface HeatmapCell {
  hour: number;
  day: number;
  value: number;
}

export interface NavItem {
  id: string;
  label: string;
  icon: string;
  badge?: number;
}
