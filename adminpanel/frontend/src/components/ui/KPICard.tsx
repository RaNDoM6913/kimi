import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { KPIData } from '@/types';
import { LineChart, Line, ResponsiveContainer } from 'recharts';

interface KPICardProps {
  data: KPIData;
  className?: string;
}

export function KPICard({ data, className }: KPICardProps) {
  const isPositive = data.trend > 0;
  const isNeutral = data.trend === 0;
  
  const sparklineData = data.sparklineData?.map((value, index) => ({ 
    name: index, 
    value 
  })) || [];

  return (
    <div className={cn("kpi-card", className)}>
      {/* Title */}
      <p className="text-sm text-[#A7B1C8] font-medium">{data.title}</p>
      
      {/* Value */}
      <div className="mt-2">
        <span className="text-4xl font-bold text-[#F5F7FF] tracking-tight">
          {data.value}
        </span>
      </div>
      
      {/* Bottom Row */}
      <div className="mt-2 flex items-end justify-between">
        {/* Trend */}
        <div className={cn(
          "trend-pill",
          isPositive && "trend-up",
          !isPositive && !isNeutral && "trend-down",
          isNeutral && "bg-[rgba(167,177,200,0.15)] text-[#A7B1C8]"
        )}>
          {isPositive && <TrendingUp className="w-3 h-3" />}
          {!isPositive && !isNeutral && <TrendingDown className="w-3 h-3" />}
          {isNeutral && <Minus className="w-3 h-3" />}
          <span>
            {isNeutral ? data.trendLabel : `${isPositive ? '+' : ''}${data.trend}%`}
          </span>
        </div>
        
        {/* Sparkline */}
        {sparklineData.length > 0 && (
          <div className="w-24 h-10">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={sparklineData}>
                <Line 
                  type="monotone" 
                  dataKey="value" 
                  stroke={isPositive ? '#2DD4A8' : isNeutral ? '#A7B1C8' : '#FF6B6B'}
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
}
