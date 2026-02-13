import { LineChartCard, BarChartCard } from '@/components/ui/charts';
import { sessionDurationData, timePerTabData, retentionCohorts, matchRateData } from '@/data/mockData';
import { TrendingUp, Clock, Users, Heart, ArrowRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

function RetentionCohortTable() {
  const getCellColor = (value: number) => {
    if (value >= 60) return 'bg-[rgba(45,212,168,0.3)] text-[#2DD4A8]';
    if (value >= 40) return 'bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]';
    if (value >= 25) return 'bg-[rgba(123,97,255,0.15)] text-[#7B61FF]';
    if (value >= 15) return 'bg-[rgba(255,209,102,0.15)] text-[#FFD166]';
    return 'bg-[rgba(255,107,107,0.15)] text-[#FF6B6B]';
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-[rgba(123,97,255,0.12)]">
            <th className="p-3 text-left text-xs font-medium text-[#A7B1C8]">Cohort</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 0</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 1</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 3</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 7</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 14</th>
            <th className="p-3 text-center text-xs font-medium text-[#A7B1C8]">Day 30</th>
          </tr>
        </thead>
        <tbody>
          {retentionCohorts.map((cohort) => (
            <tr 
              key={cohort.week} 
              className="border-b border-[rgba(123,97,255,0.08)] hover:bg-[rgba(123,97,255,0.04)] transition-colors"
            >
              <td className="p-3">
                <span className="text-sm font-medium text-[#F5F7FF]">{cohort.week}</span>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day0))}>
                  {cohort.day0}%
                </div>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day1))}>
                  {cohort.day1}%
                </div>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day3))}>
                  {cohort.day3}%
                </div>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day7))}>
                  {cohort.day7}%
                </div>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day14))}>
                  {cohort.day14}%
                </div>
              </td>
              <td className="p-3">
                <div className={cn("px-2 py-1 rounded text-sm font-medium text-center", getCellColor(cohort.day30))}>
                  {cohort.day30}%
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function EngagementPage() {
  const { hasPermission, role } = usePermissions();
  const canExportMetrics = hasPermission(ADMIN_PERMISSIONS.export_metrics);

  const handleExportReport = () => {
    if (!canExportMetrics) {
      return;
    }

    logAdminAction('export_engagement_metrics', { id: 'current-admin', role }, '127.0.0.1', getClientDevice());
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-[#F5F7FF]">Engagement</h2>
          <p className="text-sm text-[#A7B1C8]">User behavior and retention analytics</p>
        </div>
        <button
          onClick={handleExportReport}
          disabled={!canExportMetrics}
          className="btn-secondary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Export Report
          <ArrowRight className="w-4 h-4" />
        </button>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { 
            label: 'Avg Session Duration', 
            value: '8m 42s', 
            change: '+12%', 
            icon: Clock,
            color: '#7B61FF'
          },
          { 
            label: 'Daily Active Users', 
            value: '84.2K', 
            change: '+6.4%', 
            icon: Users,
            color: '#2DD4A8'
          },
          { 
            label: 'Match Rate', 
            value: '14.8%', 
            change: '+1.2%', 
            icon: Heart,
            color: '#FF6B6B'
          },
          { 
            label: 'Day 7 Retention', 
            value: '62%', 
            change: '+2.4%', 
            icon: TrendingUp,
            color: '#4CC9F0'
          },
        ].map((stat, i) => (
          <div key={i} className="glass-panel p-4">
            <div className="flex items-center justify-between mb-3">
              <div 
                className="w-10 h-10 rounded-lg flex items-center justify-center"
                style={{ background: `${stat.color}20` }}
              >
                <stat.icon className="w-5 h-5" style={{ color: stat.color }} />
              </div>
              <span className="text-xs font-medium text-[#2DD4A8]">{stat.change}</span>
            </div>
            <p className="text-sm text-[#A7B1C8]">{stat.label}</p>
            <p className="text-2xl font-bold text-[#F5F7FF]">{stat.value}</p>
          </div>
        ))}
      </div>

      {/* Session Duration Chart */}
      <div className="glass-panel p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Session Duration</h3>
            <p className="text-sm text-[#A7B1C8]">Average session length by hour of day</p>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#7B61FF]" />
              <span className="text-sm text-[#A7B1C8]">Today</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[rgba(123,97,255,0.3)]" />
              <span className="text-sm text-[#A7B1C8]">Yesterday</span>
            </div>
          </div>
        </div>
        <LineChartCard 
          data={sessionDurationData}
          dataKeys={['value']}
          colors={['#7B61FF']}
          showArea
          height={280}
        />
      </div>

      {/* Time Per Tab & Match Rate */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Time per Tab</h3>
            <p className="text-sm text-[#A7B1C8]">Average time spent in each section</p>
          </div>
          <BarChartCard 
            data={timePerTabData}
            dataKey="value"
            color="#7B61FF"
            height={260}
          />
          <div className="mt-4 grid grid-cols-2 gap-3">
            {[
              { label: 'Feed', value: '4m 10s', percent: 48 },
              { label: 'Likes', value: '2m 20s', percent: 27 },
              { label: 'Ideas', value: '1m 40s', percent: 19 },
              { label: 'Profile', value: '0m 32s', percent: 6 },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between p-2 rounded-lg bg-[rgba(14,19,32,0.5)]">
                <span className="text-sm text-[#A7B1C8]">{item.label}</span>
                <span className="text-sm font-medium text-[#F5F7FF]">{item.value}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Match Rate Analytics</h3>
            <p className="text-sm text-[#A7B1C8]">Daily match rate percentage</p>
          </div>
          <LineChartCard 
            data={matchRateData}
            dataKeys={['value']}
            colors={['#FF6B6B']}
            height={260}
          />
          <div className="mt-4 p-4 rounded-xl bg-[rgba(255,107,107,0.08)] border border-[rgba(255,107,107,0.2)]">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-[#A7B1C8]">Weekly Average</p>
                <p className="text-2xl font-bold text-[#FF6B6B]">14.8%</p>
              </div>
              <div className="text-right">
                <p className="text-sm text-[#2DD4A8]">+1.2%</p>
                <p className="text-xs text-[#A7B1C8]">vs last week</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Retention Cohorts */}
      <div className="glass-panel p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Retention Cohorts</h3>
            <p className="text-sm text-[#A7B1C8]">User retention by signup week</p>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded bg-[rgba(45,212,168,0.3)]" />
              <span className="text-xs text-[#A7B1C8]">High (60%+)</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded bg-[rgba(45,212,168,0.15)]" />
              <span className="text-xs text-[#A7B1C8]">Good (40-60%)</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded bg-[rgba(123,97,255,0.15)]" />
              <span className="text-xs text-[#A7B1C8]">Average (25-40%)</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded bg-[rgba(255,209,102,0.15)]" />
              <span className="text-xs text-[#A7B1C8]">Low (15-25%)</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded bg-[rgba(255,107,107,0.15)]" />
              <span className="text-xs text-[#A7B1C8]">Poor (&lt;15%)</span>
            </div>
          </div>
        </div>
        <RetentionCohortTable />
      </div>
    </div>
  );
}
