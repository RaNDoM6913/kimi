import { useState } from 'react';
import { experiments } from '@/data/mockData';
import { 
  Play, 
  Pause, 
  CheckCircle, 
  Clock, 
  TrendingUp, 
  Users, 
  Plus,
  MoreHorizontal,
  ChevronRight
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Experiment } from '@/types';

function ExperimentDetail({ experiment }: { experiment: Experiment }) {
  const totalConversions = experiment.variants.reduce((sum, v) => sum + v.conversions, 0);
  
  return (
    <div className="p-6">
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <h3 className="text-xl font-bold text-[#F5F7FF]">{experiment.name}</h3>
            <span className={cn(
              "px-2 py-0.5 rounded-full text-xs font-medium capitalize",
              experiment.status === 'running' && "bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]",
              experiment.status === 'paused' && "bg-[rgba(255,209,102,0.15)] text-[#FFD166]",
              experiment.status === 'completed' && "bg-[rgba(123,97,255,0.15)] text-[#7B61FF]"
            )}>
              {experiment.status}
            </span>
          </div>
          <div className="flex items-center gap-4 text-sm text-[#A7B1C8]">
            <span className="flex items-center gap-1">
              <Clock className="w-4 h-4" />
              Started {experiment.startDate}
            </span>
            {experiment.endDate && (
              <span className="flex items-center gap-1">
                <CheckCircle className="w-4 h-4" />
                Ended {experiment.endDate}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {experiment.status === 'running' ? (
            <button className="px-4 py-2 rounded-lg text-sm font-medium bg-[rgba(255,209,102,0.15)] text-[#FFD166] border border-[rgba(255,209,102,0.25)] hover:bg-[rgba(255,209,102,0.25)] transition-colors flex items-center gap-2">
              <Pause className="w-4 h-4" />
              Pause
            </button>
          ) : experiment.status === 'paused' ? (
            <button className="px-4 py-2 rounded-lg text-sm font-medium bg-[rgba(45,212,168,0.15)] text-[#2DD4A8] border border-[rgba(45,212,168,0.25)] hover:bg-[rgba(45,212,168,0.25)] transition-colors flex items-center gap-2">
              <Play className="w-4 h-4" />
              Resume
            </button>
          ) : null}
          <button className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors">
            <MoreHorizontal className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Metric */}
      <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] mb-6">
        <p className="text-sm text-[#A7B1C8] mb-1">Primary Metric</p>
        <p className="text-lg font-medium text-[#F5F7FF]">{experiment.metric}</p>
      </div>

      {/* Variants */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium text-[#A7B1C8] uppercase tracking-wider">Variants</h4>
        {experiment.variants.map((variant, index) => (
          <div 
            key={variant.id}
            className={cn(
              "p-4 rounded-xl border transition-colors",
              experiment.winner === variant.id
                ? "bg-[rgba(45,212,168,0.08)] border-[rgba(45,212,168,0.3)]"
                : "bg-[rgba(14,19,32,0.5)] border-[rgba(123,97,255,0.1)]"
            )}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <span className={cn(
                  "w-8 h-8 rounded-lg flex items-center justify-center text-sm font-bold",
                  experiment.winner === variant.id
                    ? "bg-[rgba(45,212,168,0.2)] text-[#2DD4A8]"
                    : "bg-[rgba(123,97,255,0.15)] text-[#7B61FF]"
                )}>
                  {String.fromCharCode(65 + index)}
                </span>
                <div>
                  <p className="text-sm font-medium text-[#F5F7FF]">{variant.name}</p>
                  {experiment.winner === variant.id && (
                    <span className="text-xs text-[#2DD4A8] flex items-center gap-1">
                      <CheckCircle className="w-3 h-3" />
                      Winner
                    </span>
                  )}
                </div>
              </div>
              <div className="text-right">
                <p className="text-lg font-bold text-[#F5F7FF]">{variant.conversionRate}%</p>
                <p className="text-xs text-[#A7B1C8]">{variant.conversions.toLocaleString()} conversions</p>
              </div>
            </div>
            
            {/* Allocation Bar */}
            <div className="flex items-center gap-3">
              <div className="flex-1 h-2 rounded-full bg-[rgba(123,97,255,0.1)]">
                <div 
                  className={cn(
                    "h-full rounded-full transition-all",
                    experiment.winner === variant.id ? "bg-[#2DD4A8]" : "bg-[#7B61FF]"
                  )}
                  style={{ width: `${variant.allocation}%` }}
                />
              </div>
              <span className="text-sm text-[#A7B1C8] w-12 text-right">{variant.allocation}%</span>
            </div>
          </div>
        ))}
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-3 gap-4 mt-6">
        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] text-center">
          <p className="text-2xl font-bold text-[#F5F7FF]">{totalConversions.toLocaleString()}</p>
          <p className="text-xs text-[#A7B1C8]">Total Conversions</p>
        </div>
        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] text-center">
          <p className="text-2xl font-bold text-[#F5F7FF]">{experiment.variants.length}</p>
          <p className="text-xs text-[#A7B1C8]">Variants</p>
        </div>
        <div className="p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)] text-center">
          <p className="text-2xl font-bold text-[#F5F7FF]">
            {Math.max(...experiment.variants.map(v => v.conversionRate))}%
          </p>
          <p className="text-xs text-[#A7B1C8]">Best Rate</p>
        </div>
      </div>
    </div>
  );
}

export function ExperimentsPage() {
  const [selectedExperiment, setSelectedExperiment] = useState<Experiment>(experiments[0]);

  return (
    <div className="p-6 h-[calc(100vh-64px)] animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-[#F5F7FF]">Experiments</h2>
          <p className="text-sm text-[#A7B1C8]">A/B testing and feature experimentation</p>
        </div>
        <button className="btn-primary flex items-center gap-2">
          <Plus className="w-4 h-4" />
          New Experiment
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
        {[
          { label: 'Running', value: '2', icon: Play, color: '#2DD4A8' },
          { label: 'Paused', value: '1', icon: Pause, color: '#FFD166' },
          { label: 'Completed', value: '1', icon: CheckCircle, color: '#7B61FF' },
          { label: 'Total Users', value: '48.2K', icon: Users, color: '#4CC9F0' },
        ].map((stat, i) => (
          <div key={i} className="glass-panel p-4 flex items-center gap-4">
            <div 
              className="w-10 h-10 rounded-lg flex items-center justify-center"
              style={{ background: `${stat.color}20` }}
            >
              <stat.icon className="w-5 h-5" style={{ color: stat.color }} />
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">{stat.label}</p>
              <p className="text-xl font-bold text-[#F5F7FF]">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Main Content */}
      <div className="glass-panel h-[calc(100%-140px)] overflow-hidden flex">
        {/* Experiments List */}
        <div className="w-96 border-r border-[rgba(123,97,255,0.12)] overflow-y-auto scrollbar-thin">
          {experiments.map((experiment) => (
            <button
              key={experiment.id}
              onClick={() => setSelectedExperiment(experiment)}
              className={cn(
                "w-full p-4 text-left border-b border-[rgba(123,97,255,0.08)] transition-colors",
                selectedExperiment?.id === experiment.id
                  ? "bg-[rgba(123,97,255,0.1)]"
                  : "hover:bg-[rgba(123,97,255,0.05)]"
              )}
            >
              <div className="flex items-start justify-between mb-2">
                <p className="text-sm font-medium text-[#F5F7FF] line-clamp-1">{experiment.name}</p>
                <ChevronRight className={cn(
                  "w-4 h-4 text-[#A7B1C8] flex-shrink-0",
                  selectedExperiment?.id === experiment.id && "text-[#7B61FF]"
                )} />
              </div>
              <div className="flex items-center gap-3">
                <span className={cn(
                  "px-2 py-0.5 rounded-full text-[10px] font-medium capitalize",
                  experiment.status === 'running' && "bg-[rgba(45,212,168,0.15)] text-[#2DD4A8]",
                  experiment.status === 'paused' && "bg-[rgba(255,209,102,0.15)] text-[#FFD166]",
                  experiment.status === 'completed' && "bg-[rgba(123,97,255,0.15)] text-[#7B61FF]"
                )}>
                  {experiment.status}
                </span>
                <span className="text-xs text-[#A7B1C8]">{experiment.metric}</span>
              </div>
              {experiment.winner && (
                <div className="mt-2 flex items-center gap-1 text-xs text-[#2DD4A8]">
                  <TrendingUp className="w-3 h-3" />
                  Winner: {experiment.variants.find(v => v.id === experiment.winner)?.name}
                </div>
              )}
            </button>
          ))}
        </div>

        {/* Experiment Detail */}
        <div className="flex-1 overflow-y-auto scrollbar-thin">
          {selectedExperiment && <ExperimentDetail experiment={selectedExperiment} />}
        </div>
      </div>
    </div>
  );
}
