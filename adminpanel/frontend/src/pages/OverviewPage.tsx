import { useState, useMemo } from 'react';
import { KPICard } from '@/components/ui/KPICard';
import { LineChartCard } from '@/components/ui/charts';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { overviewKPIs, growthChartData, alerts } from '@/data/mockData';
import { TrendingUp, AlertTriangle, Check, Info, AlertCircle, ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Alert as AlertType } from '@/types';
import type { ChartDataPoint } from '@/types';

type GrowthPeriod = '1d' | '7d' | '1m' | '3m' | '12m';

const GROWTH_PERIOD_LABELS: Record<GrowthPeriod, string> = {
  '1d': '1 day',
  '7d': '7 days',
  '1m': '30 days',
  '3m': '3 months',
  '12m': '12 months',
};

function getGrowthChartDataByPeriod(data: ChartDataPoint[], period: GrowthPeriod): ChartDataPoint[] {
  switch (period) {
    case '1d':
    case '7d':
    case '1m':
      return data.slice(-1);
    case '3m':
      return data.slice(-3);
    case '12m':
      return data;
    default:
      return data.slice(-1);
  }
}

function getGrowthSubtitle(period: GrowthPeriod): string {
  switch (period) {
    case '1d':
      return 'DAU and MAU trends over the last day';
    case '7d':
      return 'DAU and MAU trends over the last 7 days';
    case '1m':
      return 'DAU and MAU trends over the last 30 days';
    case '3m':
      return 'DAU and MAU trends over the last 3 months';
    case '12m':
      return 'DAU and MAU trends over the last 12 months';
    default:
      return 'DAU and MAU trends over the last 30 days';
  }
}

function AlertItem({ alert }: { alert: AlertType }) {
  const getIcon = (type: AlertType['type']) => {
    switch (type) {
      case 'error': return <AlertCircle className="w-4 h-4 text-[#FF6B6B]" />;
      case 'warning': return <AlertTriangle className="w-4 h-4 text-[#FFD166]" />;
      case 'success': return <Check className="w-4 h-4 text-[#2DD4A8]" />;
      case 'info': return <Info className="w-4 h-4 text-[#4CC9F0]" />;
    }
  };

  const getBorderColor = (type: AlertType['type']) => {
    switch (type) {
      case 'error': return 'border-l-[#FF6B6B]';
      case 'warning': return 'border-l-[#FFD166]';
      case 'success': return 'border-l-[#2DD4A8]';
      case 'info': return 'border-l-[#4CC9F0]';
    }
  };

  return (
    <div className={cn(
      "p-4 rounded-lg border-l-4 bg-[rgba(14,19,32,0.5)] hover:bg-[rgba(123,97,255,0.05)] transition-colors cursor-pointer",
      getBorderColor(alert.type)
    )}>
      <div className="flex items-start gap-3">
        <div className="mt-0.5">{getIcon(alert.type)}</div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-[#F5F7FF]">{alert.title}</p>
            <span className="text-[10px] text-[#A7B1C8]">{alert.timestamp}</span>
          </div>
          <p className="text-xs text-[#A7B1C8] mt-0.5 line-clamp-2">{alert.message}</p>
        </div>
      </div>
    </div>
  );
}

export function OverviewPage() {
  const [growthPeriod, setGrowthPeriod] = useState<GrowthPeriod>('1m');

  const growthDataByPeriod = useMemo(
    () => getGrowthChartDataByPeriod(growthChartData, growthPeriod),
    [growthPeriod]
  );

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* KPI Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        {overviewKPIs.map((kpi, index) => (
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
        {/* Growth Chart */}
        <div 
          className="lg:col-span-2 glass-panel p-5 animate-slide-up"
          style={{ animationDelay: '480ms' }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Growth Overview</h3>
              <p className="text-sm text-[#A7B1C8]">{getGrowthSubtitle(growthPeriod)}</p>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="flex items-center gap-2 text-sm text-[#7B61FF] hover:underline outline-none"
                >
                  {GROWTH_PERIOD_LABELS[growthPeriod]}
                  <ChevronDown className="w-4 h-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="min-w-[8rem] bg-[#0E1320] border-[#1E2636]">
                {(['1d', '7d', '1m', '3m', '12m'] as const).map((period) => (
                  <DropdownMenuItem
                    key={period}
                    onClick={() => setGrowthPeriod(period)}
                    className="text-[#F5F7FF] focus:bg-[rgba(123,97,255,0.15)] focus:text-[#F5F7FF]"
                  >
                    {GROWTH_PERIOD_LABELS[period]}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <LineChartCard 
            data={growthDataByPeriod} 
            dataKeys={['value', 'value2']}
            colors={['#7B61FF', '#4CC9F0']}
            showArea
            height={320}
          />
        </div>

        {/* Alerts Panel */}
        <div 
          className="glass-panel p-5 animate-slide-up"
          style={{ animationDelay: '560ms' }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Alerts</h3>
              <p className="text-sm text-[#A7B1C8]">Recent system alerts</p>
            </div>
            <span className="px-2 py-1 bg-[rgba(255,107,107,0.15)] text-[#FF6B6B] text-xs rounded-full">
              {alerts.filter(a => !a.isRead).length} new
            </span>
          </div>
          <div className="space-y-3 max-h-[320px] overflow-y-auto scrollbar-thin pr-1">
            {alerts.map((alert) => (
              <AlertItem key={alert.id} alert={alert} />
            ))}
          </div>
        </div>
      </div>

      {/* Bottom Row - Supporting Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Retention Mini Chart */}
        <div 
          className="glass-panel p-5 animate-slide-up"
          style={{ animationDelay: '640ms' }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Retention Rate</h3>
              <p className="text-sm text-[#A7B1C8]">Day 7 retention by cohort</p>
            </div>
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-[#2DD4A8]" />
              <span className="text-sm text-[#2DD4A8]">+2.4%</span>
            </div>
          </div>
          <LineChartCard 
            data={[
              { name: 'W1', value: 62 },
              { name: 'W2', value: 58 },
              { name: 'W3', value: 64 },
              { name: 'W4', value: 55 },
              { name: 'W5', value: 66 },
              { name: 'W6', value: 61 },
            ]}
            dataKeys={['value']}
            colors={['#2DD4A8']}
            height={200}
          />
        </div>

        {/* Revenue Mini Chart */}
        <div 
          className="glass-panel p-5 animate-slide-up"
          style={{ animationDelay: '720ms' }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Revenue Trend</h3>
              <p className="text-sm text-[#A7B1C8]">Daily revenue this week</p>
            </div>
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-[#7B61FF]" />
              <span className="text-sm text-[#7B61FF]">+8.1%</span>
            </div>
          </div>
          <LineChartCard 
            data={[
              { name: 'Mon', value: 38500 },
              { name: 'Tue', value: 39200 },
              { name: 'Wed', value: 40100 },
              { name: 'Thu', value: 39800 },
              { name: 'Fri', value: 41500 },
              { name: 'Sat', value: 43800 },
              { name: 'Sun', value: 42180 },
            ]}
            dataKeys={['value']}
            colors={['#7B61FF']}
            showArea
            height={200}
          />
        </div>
      </div>
    </div>
  );
}
