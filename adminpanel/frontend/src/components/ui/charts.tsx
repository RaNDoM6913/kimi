import { useState } from 'react';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from 'recharts';
import { cn } from '@/lib/utils';
import type { ChartDataPoint } from '@/types';

interface LineChartProps {
  data: ChartDataPoint[];
  dataKeys: string[];
  colors?: string[];
  showArea?: boolean;
  height?: number;
  className?: string;
}

export function LineChartCard({ 
  data, 
  dataKeys, 
  colors = ['#7B61FF', '#4CC9F0'], 
  showArea = false,
  height = 280,
  className 
}: LineChartProps) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  return (
    <div className={cn("chart-card", className)}>
      <ResponsiveContainer width="100%" height={height}>
        {showArea ? (
          <AreaChart data={data} onMouseMove={(e: any) => setHoveredIndex(e?.activeTooltipIndex ?? null)}>
            <defs>
              {colors.map((color, i) => (
                <linearGradient key={i} id={`gradient-${i}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              ))}
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
              tickFormatter={(value) => value >= 1000 ? `${(value/1000).toFixed(0)}K` : value}
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
            {dataKeys.map((key, i) => (
              <Area 
                key={key}
                type="monotone" 
                dataKey={key} 
                stroke={colors[i % colors.length]} 
                strokeWidth={2}
                fill={`url(#gradient-${i})`}
              />
            ))}
          </AreaChart>
        ) : (
          <LineChart data={data} onMouseMove={(e: any) => setHoveredIndex(e?.activeTooltipIndex ?? null)}>
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
              tickFormatter={(value) => value >= 1000 ? `${(value/1000).toFixed(0)}K` : value}
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
            {dataKeys.map((key, i) => (
              <Line 
                key={key}
                type="monotone" 
                dataKey={key} 
                stroke={colors[i % colors.length]} 
                strokeWidth={2}
                dot={hoveredIndex !== null}
                activeDot={{ r: 6, strokeWidth: 0, fill: colors[i % colors.length] }}
              />
            ))}
          </LineChart>
        )}
      </ResponsiveContainer>
    </div>
  );
}

interface BarChartProps {
  data: ChartDataPoint[];
  dataKey?: string;
  color?: string;
  height?: number;
  horizontal?: boolean;
  className?: string;
}

export function BarChartCard({ 
  data, 
  dataKey = 'value',
  color = '#7B61FF',
  height = 280,
  horizontal = false,
  className 
}: BarChartProps) {
  return (
    <div className={cn("chart-card", className)}>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={data} layout={horizontal ? 'vertical' : 'horizontal'}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(123,97,255,0.08)" />
          <XAxis 
            type={horizontal ? 'number' : 'category'}
            dataKey={horizontal ? undefined : 'name'}
            stroke="#A7B1C8" 
            fontSize={12}
            tickLine={false}
            axisLine={false}
          />
          <YAxis 
            type={horizontal ? 'category' : 'number'}
            dataKey={horizontal ? 'name' : undefined}
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
            cursor={{ fill: 'rgba(123, 97, 255, 0.05)' }}
          />
          <Bar 
            dataKey={dataKey} 
            fill={color}
            radius={horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

interface DonutChartProps {
  data: { name: string; value: number; fill: string }[];
  height?: number;
  className?: string;
  showLegend?: boolean;
}

export function DonutChartCard({ 
  data, 
  height = 200,
  className,
  showLegend = true
}: DonutChartProps) {
  return (
    <div className={cn("chart-card", className)}>
      <ResponsiveContainer width="100%" height={height}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={80}
            paddingAngle={4}
            dataKey="value"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.fill} stroke="none" />
            ))}
          </Pie>
          <Tooltip 
            contentStyle={{ 
              background: 'rgba(14, 19, 32, 0.95)', 
              border: '1px solid rgba(123, 97, 255, 0.25)',
              borderRadius: '12px',
              boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)'
            }}
            labelStyle={{ color: '#F5F7FF', fontWeight: 600 }}
            itemStyle={{ color: '#A7B1C8' }}
            formatter={(value: number) => [`${value}%`, '']}
          />
          {showLegend && (
            <Legend 
              verticalAlign="bottom" 
              height={36}
              iconType="circle"
              formatter={(value) => <span className="text-[#A7B1C8] text-xs">{value}</span>}
            />
          )}
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}

interface HeatmapProps {
  data: { hour: number; day: number; value: number }[];
  className?: string;
}

const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const hours = ['12am', '6am', '12pm', '6pm', '10pm'];

export function Heatmap({ data, className }: HeatmapProps) {
  const maxValue = Math.max(...data.map(d => d.value));
  
  const getColor = (value: number) => {
    const intensity = value / maxValue;
    return `rgba(123, 97, 255, ${0.1 + intensity * 0.8})`;
  };

  return (
    <div className={cn("chart-card", className)}>
      <div className="flex">
        {/* Day labels */}
        <div className="flex flex-col justify-around pr-2 py-4">
          {days.map(day => (
            <span key={day} className="text-xs text-[#A7B1C8] h-8 flex items-center">{day}</span>
          ))}
        </div>
        
        {/* Heatmap grid */}
        <div className="flex-1">
          {/* Hour labels */}
          <div className="flex justify-between px-1 mb-2">
            {hours.map(hour => (
              <span key={hour} className="text-xs text-[#A7B1C8]">{hour}</span>
            ))}
          </div>
          
          {/* Grid */}
          <div className="space-y-1">
            {days.map((day, dayIndex) => (
              <div key={day} className="flex gap-1">
                {[0, 6, 12, 18, 22].map(hour => {
                  const cell = data.find(d => d.day === dayIndex && d.hour === hour);
                  const value = cell?.value || 0;
                  return (
                    <div
                      key={`${day}-${hour}`}
                      className="flex-1 h-8 rounded-md transition-all duration-200 hover:ring-2 hover:ring-[#7B61FF] cursor-pointer"
                      style={{ backgroundColor: getColor(value) }}
                      title={`${day} ${hour}:00 - ${value} purchases`}
                    />
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
