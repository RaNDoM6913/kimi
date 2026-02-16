import { LineChartCard, BarChartCard, DonutChartCard, Heatmap } from '@/components/ui/charts';
import { 
  gmvData, 
  subscriptionData, 
  microtransactionData, 
  purchaseHeatmapData,
  purchasesByGenderData,
  purchasesByRegionData
} from '@/data/mockData';
import { TrendingUp, DollarSign, Users, CreditCard, ArrowRight } from 'lucide-react';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

/** "Export Report" button for TopBar when on Monetization page */
export function MonetizationExportButton() {
  const { hasPermission, role } = usePermissions();
  const canExportMetrics = hasPermission(ADMIN_PERMISSIONS.export_metrics);

  const handleExportReport = () => {
    if (!canExportMetrics) return;
    logAdminAction(
      'export_monetization_metrics',
      { id: 'current-admin', role },
      '127.0.0.1',
      getClientDevice(),
    );
  };

  return (
    <button
      onClick={handleExportReport}
      disabled={!canExportMetrics}
      className="btn-secondary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      Export
      <ArrowRight className="w-4 h-4" />
    </button>
  );
}

export function MonetizationPage() {
  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Key Revenue Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { 
            label: 'GMV Today', 
            value: '$42,180', 
            change: '+8.1%', 
            icon: DollarSign,
            color: '#7B61FF'
          },
          { 
            label: 'ARPU', 
            value: '$2.40', 
            change: '+3.1%', 
            icon: Users,
            color: '#2DD4A8'
          },
          { 
            label: 'ARPPU', 
            value: '$18.90', 
            change: '+5.4%', 
            icon: CreditCard,
            color: '#4CC9F0'
          },
          { 
            label: 'MRR', 
            value: '$1.24M', 
            change: '+12.3%', 
            icon: TrendingUp,
            color: '#FFD166'
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

      {/* GMV Chart */}
      <div className="glass-panel p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Gross Merchandise Value</h3>
            <p className="text-sm text-[#A7B1C8]">Total transaction volume over time</p>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#7B61FF]" />
              <span className="text-sm text-[#A7B1C8]">GMV</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#4CC9F0]" />
              <span className="text-sm text-[#A7B1C8]">Target</span>
            </div>
          </div>
        </div>
        <LineChartCard 
          data={gmvData}
          dataKeys={['value', 'value2']}
          colors={['#7B61FF', '#4CC9F0']}
          showArea
          height={320}
        />
      </div>

      {/* Subscriptions & Microtransactions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Subscription Growth</h3>
            <p className="text-sm text-[#A7B1C8]">Active subscriptions by tier</p>
          </div>
          <LineChartCard 
            data={subscriptionData}
            dataKeys={['value', 'value2', 'value3']}
            colors={['#7B61FF', '#4CC9F0', '#2DD4A8']}
            showArea
            height={260}
          />
          <div className="mt-4 flex items-center gap-6">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#7B61FF]" />
              <span className="text-sm text-[#A7B1C8]">Gold</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#4CC9F0]" />
              <span className="text-sm text-[#A7B1C8]">Plus</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#2DD4A8]" />
              <span className="text-sm text-[#A7B1C8]">Basic</span>
            </div>
          </div>
        </div>

        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Microtransactions</h3>
            <p className="text-sm text-[#A7B1C8]">In-app purchases by day</p>
          </div>
          <BarChartCard 
            data={microtransactionData}
            dataKey="value"
            color="#FFD166"
            height={260}
          />
          <div className="mt-4 grid grid-cols-3 gap-3">
            {[
              { label: 'Super Likes', value: '$12,420' },
              { label: 'Boosts', value: '$8,340' },
              { label: 'Other', value: '$3,580' },
            ].map((item) => (
              <div key={item.label} className="p-3 rounded-lg bg-[rgba(14,19,32,0.5)] text-center">
                <p className="text-xs text-[#A7B1C8]">{item.label}</p>
                <p className="text-sm font-medium text-[#F5F7FF]">{item.value}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ARPU/ARPPU & Purchase Heatmap */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">ARPU / ARPPU</h3>
            <p className="text-sm text-[#A7B1C8]">Average revenue per user</p>
          </div>
          <div className="space-y-4">
            <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-[#A7B1C8]">ARPU</span>
                <span className="text-xs text-[#2DD4A8]">+3.1%</span>
              </div>
              <p className="text-3xl font-bold text-[#F5F7FF]">$2.40</p>
              <div className="mt-2 h-1.5 rounded-full bg-[rgba(123,97,255,0.1)]">
                <div className="h-full w-[60%] rounded-full bg-[#7B61FF]" />
              </div>
            </div>
            <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-[#A7B1C8]">ARPPU</span>
                <span className="text-xs text-[#2DD4A8]">+5.4%</span>
              </div>
              <p className="text-3xl font-bold text-[#F5F7FF]">$18.90</p>
              <div className="mt-2 h-1.5 rounded-full bg-[rgba(123,97,255,0.1)]">
                <div className="h-full w-[85%] rounded-full bg-[#4CC9F0]" />
              </div>
            </div>
          </div>
        </div>

        <div className="lg:col-span-2 glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Purchase Heatmap</h3>
            <p className="text-sm text-[#A7B1C8]">Purchases by hour and day of week</p>
          </div>
          <Heatmap data={purchaseHeatmapData} />
        </div>
      </div>

      {/* Purchases by Gender & Region */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Purchases by Gender</h3>
            <p className="text-sm text-[#A7B1C8]">Revenue distribution by gender</p>
          </div>
          <DonutChartCard data={purchasesByGenderData} height={240} />
        </div>

        <div className="glass-panel p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-[#F5F7FF]">Purchases by Region</h3>
            <p className="text-sm text-[#A7B1C8]">Revenue distribution by geography</p>
          </div>
          <DonutChartCard data={purchasesByRegionData} height={240} />
        </div>
      </div>
    </div>
  );
}
