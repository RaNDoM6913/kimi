import { useState } from 'react';
import { systemMetrics } from '@/data/mockData';
import { 
  AreaChart, 
  Area,
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer
} from 'recharts';
import { 
  Zap, 
  AlertTriangle, 
  Activity, 
  Database, 
  Server,
  Shield,
  TrendingUp,
  TrendingDown,
  Maximize2
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { SystemMetric } from '@/types';

function MetricCard({ 
  metric, 
  icon: Icon, 
  color,
  onClick 
}: { 
  metric: SystemMetric; 
  icon: React.ElementType;
  color: string;
  onClick: () => void;
}) {
  // Determine if trend is good or bad based on metric
  const isGoodTrend = metric.name.includes('Error') || metric.name.includes('Latency') || metric.name.includes('Blocks')
    ? metric.trend < 0 
    : metric.trend > 0;
  const gradientId = `gradient-${metric.name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;

  return (
    <div 
      onClick={onClick}
      className="glass-panel p-5 cursor-pointer hover:border-[rgba(123,97,255,0.35)] transition-all hover:-translate-y-0.5"
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div 
            className="w-10 h-10 rounded-lg flex items-center justify-center"
            style={{ background: `${color}20` }}
          >
            <Icon className="w-5 h-5" style={{ color }} />
          </div>
          <div>
            <p className="text-sm text-[#A7B1C8]">{metric.name}</p>
            <p className="text-2xl font-bold text-[#F5F7FF]">
              {metric.value}{metric.unit && <span className="text-lg text-[#A7B1C8]">{metric.unit}</span>}
            </p>
          </div>
        </div>
        <div className={cn(
          "flex items-center gap-1 text-xs font-medium",
          isGoodTrend ? "text-[#2DD4A8]" : "text-[#FF6B6B]"
        )}>
          {isGoodTrend ? <TrendingDown className="w-3 h-3" /> : <TrendingUp className="w-3 h-3" />}
          {Math.abs(metric.trend)}%
        </div>
      </div>
      
      {/* Mini Chart */}
      <div className="h-16">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={metric.data}>
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                <stop offset="95%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <Area 
              type="monotone" 
              dataKey="value" 
              stroke={color}
              strokeWidth={2}
              fill={`url(#${gradientId})`}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      
      <div className="mt-3 flex items-center justify-between">
        <span className="text-xs text-[#A7B1C8]">Last 24 hours</span>
        <Maximize2 className="w-4 h-4 text-[#A7B1C8]" />
      </div>
    </div>
  );
}

function MetricModal({ metric, onClose }: { metric: SystemMetric; onClose: () => void }) {
  const getColor = (name: string) => {
    if (name.includes('Latency')) return '#7B61FF';
    if (name.includes('Error')) return '#FF6B6B';
    if (name.includes('Events')) return '#2DD4A8';
    if (name.includes('Queue')) return '#FFD166';
    if (name.includes('Redis')) return '#4CC9F0';
    return '#A7B1C8';
  };

  const color = getColor(metric.name);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-4xl glass-panel max-h-[90vh] overflow-hidden flex flex-col animate-slide-up">
        {/* Header */}
        <div className="p-6 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
          <div>
            <h3 className="text-xl font-bold text-[#F5F7FF]">{metric.name}</h3>
            <p className="text-sm text-[#A7B1C8]">24-hour historical data</p>
          </div>
          <button 
            onClick={onClose}
            className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
          >
            <span className="text-2xl">&times;</span>
          </button>
        </div>

        {/* Content */}
        <div className="p-6 flex-1 overflow-y-auto">
          <div className="flex items-center gap-8 mb-6">
            <div>
              <p className="text-sm text-[#A7B1C8]">Current</p>
              <p className="text-4xl font-bold text-[#F5F7FF]">
                {metric.value}{metric.unit && <span className="text-2xl text-[#A7B1C8]">{metric.unit}</span>}
              </p>
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">Average</p>
              <p className="text-2xl font-bold text-[#F5F7FF]">
                {(metric.data.reduce((a, b) => a + b.value, 0) / metric.data.length).toFixed(1)}
                {metric.unit}
              </p>
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">Peak</p>
              <p className="text-2xl font-bold text-[#F5F7FF]">
                {Math.max(...metric.data.map(d => d.value))}
                {metric.unit}
              </p>
            </div>
          </div>

          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={metric.data}>
                <defs>
                  <linearGradient id={`modal-gradient`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                    <stop offset="95%" stopColor={color} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(123,97,255,0.08)" />
                <XAxis 
                  dataKey="name" 
                  stroke="#A7B1C8" 
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis 
                  stroke="#A7B1C8" 
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip 
                  contentStyle={{ 
                    background: 'rgba(14, 19, 32, 0.95)', 
                    border: '1px solid rgba(123, 97, 255, 0.25)',
                    borderRadius: '12px',
                    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)'
                  }}
                  labelStyle={{ color: '#F5F7FF', fontWeight: 600 }}
                  itemStyle={{ color: '#A7B1C8' }}
                />
                <Area 
                  type="monotone" 
                  dataKey="value" 
                  stroke={color}
                  strokeWidth={2}
                  fill="url(#modal-gradient)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  );
}

export function SystemPage() {
  const [selectedMetric, setSelectedMetric] = useState<SystemMetric | null>(null);

  const metricConfigs = [
    { metric: systemMetrics[0], icon: Zap, color: '#7B61FF' },
    { metric: systemMetrics[1], icon: AlertTriangle, color: '#FF6B6B' },
    { metric: systemMetrics[2], icon: Activity, color: '#2DD4A8' },
    { metric: systemMetrics[3], icon: Server, color: '#FFD166' },
    { metric: systemMetrics[4], icon: Database, color: '#4CC9F0' },
    { metric: systemMetrics[5], icon: Shield, color: '#A7B1C8' },
  ];

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Status Overview */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { label: 'Uptime', value: '99.97%', status: 'good' },
          { label: 'API Health', value: 'Healthy', status: 'good' },
          { label: 'Database', value: 'Connected', status: 'good' },
          { label: 'CDN', value: 'Active', status: 'good' },
        ].map((item, i) => (
          <div key={i} className="glass-panel p-4 flex items-center justify-between">
            <span className="text-sm text-[#A7B1C8]">{item.label}</span>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 bg-[#2DD4A8] rounded-full" />
              <span className="text-sm font-medium text-[#F5F7FF]">{item.value}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {metricConfigs.map(({ metric, icon, color }) => (
          <MetricCard
            key={metric.name}
            metric={metric}
            icon={icon}
            color={color}
            onClick={() => setSelectedMetric(metric)}
          />
        ))}
      </div>

      {/* Additional Info */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="glass-panel p-5">
          <h3 className="text-lg font-semibold text-[#F5F7FF] mb-4">Recent Incidents</h3>
          <div className="space-y-3">
            {[
              { time: '2 hours ago', title: 'API latency spike', status: 'resolved', duration: '5 min' },
              { time: '1 day ago', title: 'Database connection pool', status: 'resolved', duration: '12 min' },
              { time: '3 days ago', title: 'CDN cache invalidation', status: 'resolved', duration: '3 min' },
            ].map((incident, i) => (
              <div key={i} className="flex items-center justify-between p-3 rounded-lg bg-[rgba(14,19,32,0.5)]">
                <div>
                  <p className="text-sm text-[#F5F7FF]">{incident.title}</p>
                  <p className="text-xs text-[#A7B1C8]">{incident.time}</p>
                </div>
                <div className="text-right">
                  <span className="px-2 py-0.5 rounded-full text-xs bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]">
                    {incident.status}
                  </span>
                  <p className="text-xs text-[#A7B1C8] mt-1">{incident.duration}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="glass-panel p-5">
          <h3 className="text-lg font-semibold text-[#F5F7FF] mb-4">System Resources</h3>
          <div className="space-y-4">
            {[
              { label: 'CPU Usage', value: 42, color: '#7B61FF' },
              { label: 'Memory', value: 68, color: '#4CC9F0' },
              { label: 'Disk I/O', value: 23, color: '#2DD4A8' },
              { label: 'Network', value: 56, color: '#FFD166' },
            ].map((resource, i) => (
              <div key={i}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-[#A7B1C8]">{resource.label}</span>
                  <span className="text-sm text-[#F5F7FF]">{resource.value}%</span>
                </div>
                <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                  <div 
                    className="h-full rounded-full transition-all"
                    style={{ width: `${resource.value}%`, background: resource.color }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Metric Modal */}
      {selectedMetric && (
        <MetricModal 
          metric={selectedMetric} 
          onClose={() => setSelectedMetric(null)} 
        />
      )}
    </div>
  );
}
