import { LineChartCard, BarChartCard } from '@/components/ui/charts';
import { 
  adsKPIs, 
  adRevenueData, 
  audienceData, 
  adCampaigns 
} from '@/data/mockData';
import { KPICard } from '@/components/ui/KPICard';
import { TrendingUp, Eye, MousePointer, Target, ArrowRight, Play, Pause, MoreHorizontal } from 'lucide-react';
import { cn } from '@/lib/utils';

export function AdsPage() {
  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-[#F5F7FF]">Ads</h2>
          <p className="text-sm text-[#A7B1C8]">Advertising performance and campaign management</p>
        </div>
        <button className="btn-primary flex items-center gap-2">
          + New Campaign
        </button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {adsKPIs.map((kpi, index) => (
          <div 
            key={kpi.title}
            className="animate-slide-up"
            style={{ animationDelay: `${index * 80}ms` }}
          >
            <KPICard data={kpi} />
          </div>
        ))}
      </div>

      {/* Main Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Ad Revenue Chart */}
        <div className="lg:col-span-2 glass-panel p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Ad Revenue</h3>
              <p className="text-sm text-[#A7B1C8]">Daily ad revenue this week</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-2xl font-bold text-[#7B61FF]">$8,240</span>
              <span className="text-xs text-[#2DD4A8] flex items-center gap-1">
                <TrendingUp className="w-3 h-3" />
                +12.3%
              </span>
            </div>
          </div>
          <LineChartCard 
            data={adRevenueData}
            dataKeys={['value']}
            colors={['#7B61FF']}
            showArea
            height={280}
          />
        </div>

        {/* Audience Segmentation */}
        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Audience</h3>
            <p className="text-sm text-[#A7B1C8]">By age and gender</p>
          </div>
          <BarChartCard 
            data={audienceData}
            dataKey="value"
            color="#4CC9F0"
            horizontal
            height={280}
          />
        </div>
      </div>

      {/* Campaigns Table */}
      <div className="glass-panel overflow-hidden">
        <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Campaigns</h3>
            <p className="text-sm text-[#A7B1C8]">Active and past advertising campaigns</p>
          </div>
          <button className="flex items-center gap-2 text-sm text-[#7B61FF] hover:underline">
            View All
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[rgba(123,97,255,0.12)]">
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Campaign</th>
                <th className="p-4 text-left text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Status</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Impressions</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Clicks</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">CTR</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Conv.</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Revenue</th>
                <th className="p-4 text-right text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">ROAS</th>
                <th className="p-4 text-center text-xs font-medium text-[#A7B1C8] uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody>
              {adCampaigns.map((campaign) => (
                <tr 
                  key={campaign.id} 
                  className="border-b border-[rgba(123,97,255,0.08)] table-row"
                >
                  <td className="p-4">
                    <div>
                      <p className="text-sm font-medium text-[#F5F7FF]">{campaign.name}</p>
                      <p className="text-xs text-[#A7B1C8] font-mono">{campaign.id}</p>
                    </div>
                  </td>
                  <td className="p-4">
                    <span className={cn(
                      "px-2 py-1 rounded-full text-xs font-medium capitalize",
                      campaign.status === 'active' && "bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]",
                      campaign.status === 'paused' && "bg-[rgba(255,209,102,0.15)] text-[#FFD166]",
                      campaign.status === 'ended' && "bg-[rgba(167,177,200,0.15)] text-[#A7B1C8]"
                    )}>
                      {campaign.status}
                    </span>
                  </td>
                  <td className="p-4 text-right">
                    <p className="text-sm text-[#F5F7FF]">{(campaign.impressions / 1000).toFixed(0)}K</p>
                  </td>
                  <td className="p-4 text-right">
                    <p className="text-sm text-[#F5F7FF]">{(campaign.clicks / 1000).toFixed(1)}K</p>
                  </td>
                  <td className="p-4 text-right">
                    <p className="text-sm text-[#F5F7FF]">{campaign.ctr}%</p>
                  </td>
                  <td className="p-4 text-right">
                    <p className="text-sm text-[#F5F7FF]">{campaign.conversions}</p>
                  </td>
                  <td className="p-4 text-right">
                    <p className="text-sm text-[#F5F7FF]">${campaign.revenue.toLocaleString()}</p>
                  </td>
                  <td className="p-4 text-right">
                    <span className={cn(
                      "text-sm font-medium",
                      campaign.roas >= 2.5 ? "text-[#2DD4A8]" : "text-[#FFD166]"
                    )}>
                      {campaign.roas.toFixed(2)}x
                    </span>
                  </td>
                  <td className="p-4">
                    <div className="flex items-center justify-center gap-2">
                      <button 
                        className={cn(
                          "p-2 rounded-lg transition-colors",
                          campaign.status === 'active' 
                            ? "text-[#FFD166] hover:bg-[rgba(255,209,102,0.1)]" 
                            : "text-[#2DD4A8] hover:bg-[rgba(45,212,168,0.1)]"
                        )}
                        title={campaign.status === 'active' ? 'Pause' : 'Resume'}
                      >
                        {campaign.status === 'active' ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                      </button>
                      <button 
                        className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors"
                        title="More options"
                      >
                        <MoreHorizontal className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Performance Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="glass-panel p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-[rgba(123,97,255,0.15)] flex items-center justify-center">
              <Eye className="w-5 h-5 text-[#7B61FF]" />
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">Total Impressions</p>
              <p className="text-xl font-bold text-[#F5F7FF]">1.84M</p>
            </div>
          </div>
          <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
            <div className="h-full w-[78%] rounded-full bg-[#7B61FF]" />
          </div>
          <p className="text-xs text-[#A7B1C8] mt-2">78% of monthly target</p>
        </div>

        <div className="glass-panel p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-[rgba(76,201,240,0.15)] flex items-center justify-center">
              <MousePointer className="w-5 h-5 text-[#4CC9F0]" />
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">Click-Through Rate</p>
              <p className="text-xl font-bold text-[#F5F7FF]">2.51%</p>
            </div>
          </div>
          <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
            <div className="h-full w-[63%] rounded-full bg-[#4CC9F0]" />
          </div>
          <p className="text-xs text-[#A7B1C8] mt-2">+0.3% vs last week</p>
        </div>

        <div className="glass-panel p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-[rgba(45,212,168,0.15)] flex items-center justify-center">
              <Target className="w-5 h-5 text-[#2DD4A8]" />
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">Conversion Rate</p>
              <p className="text-xl font-bold text-[#F5F7FF]">4.8%</p>
            </div>
          </div>
          <div className="h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
            <div className="h-full w-[82%] rounded-full bg-[#2DD4A8]" />
          </div>
          <p className="text-xs text-[#A7B1C8] mt-2">+0.5% vs last week</p>
        </div>
      </div>
    </div>
  );
}
