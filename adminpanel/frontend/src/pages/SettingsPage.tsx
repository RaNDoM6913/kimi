import { useState } from 'react';
import {
  Save, 
  Bell, 
  Shield, 
  Globe, 
  Database, 
  Webhook,
  Mail,
  Smartphone,
  Slack
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

export function SettingsPage() {
  const [saved, setSaved] = useState(false);
  const [activeTab, setActiveTab] = useState<'general' | 'notifications' | 'security' | 'integrations'>('general');
  const { hasPermission, role } = usePermissions();
  const canManageFlags = hasPermission(ADMIN_PERMISSIONS.manage_flags);
  const canChangeLimits = hasPermission(ADMIN_PERMISSIONS.change_limits);
  const canDisableAdmin2FA = hasPermission(ADMIN_PERMISSIONS.disable_admin_2fa);

  const handleSave = () => {
    if (!canChangeLimits) {
      return;
    }

    logAdminAction('save_settings', { id: 'current-admin', role }, '127.0.0.1', getClientDevice());
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div className="p-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-[#F5F7FF]">Settings</h2>
          <p className="text-sm text-[#A7B1C8]">Configure application settings</p>
        </div>
        <button 
          onClick={handleSave}
          disabled={!canChangeLimits}
          className={cn(
            "btn-primary flex items-center gap-2 transition-all disabled:opacity-50 disabled:cursor-not-allowed",
            saved && "bg-[#2DD4A8]"
          )}
        >
          <Save className="w-4 h-4" />
          {saved ? 'Saved!' : 'Save Changes'}
        </button>
      </div>

      <div className="flex gap-6">
        {/* Sidebar */}
        <div className="w-64 flex-shrink-0">
          <div className="glass-panel overflow-hidden">
            {[
              { id: 'general', label: 'General', icon: Globe },
              { id: 'notifications', label: 'Notifications', icon: Bell },
              { id: 'security', label: 'Security', icon: Shield },
              { id: 'integrations', label: 'Integrations', icon: Database },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as typeof activeTab)}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 text-left transition-colors",
                  activeTab === tab.id
                    ? "bg-[rgba(123,97,255,0.1)] text-[#F5F7FF]"
                    : "text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.05)] hover:text-[#F5F7FF]"
                )}
              >
                <tab.icon className="w-5 h-5" />
                <span className="text-sm font-medium">{tab.label}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1">
          {activeTab === 'general' && (
            <div className="glass-panel p-6 space-y-6 animate-fade-in">
              <h3 className="text-lg font-semibold text-[#F5F7FF]">General Settings</h3>
              
              <div className="grid grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm text-[#A7B1C8] mb-2">App Name</label>
                  <input
                    type="text"
                    defaultValue="Heartbeat"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)]"
                  />
                </div>
                <div>
                  <label className="block text-sm text-[#A7B1C8] mb-2">Support Email</label>
                  <input
                    type="email"
                    defaultValue="support@heartbeat.app"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)]"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm text-[#A7B1C8] mb-2">Default Trust Threshold</label>
                  <input
                    type="number"
                    defaultValue="70"
                    min="0"
                    max="100"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)]"
                  />
                  <p className="text-xs text-[#A7B1C8] mt-1">Minimum trust score for new users</p>
                </div>
                <div>
                  <label className="block text-sm text-[#A7B1C8] mb-2">Session Timeout (minutes)</label>
                  <input
                    type="number"
                    defaultValue="30"
                    min="5"
                    max="120"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)]"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-[#A7B1C8] mb-2">App Description</label>
                <textarea
                  rows={4}
                  defaultValue="A modern dating platform connecting people worldwide."
                  disabled={!canChangeLimits}
                  className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none focus:ring-2 focus:ring-[rgba(123,97,255,0.15)] resize-none"
                />
              </div>
            </div>
          )}

          {activeTab === 'notifications' && (
            <div className="glass-panel p-6 space-y-6 animate-fade-in">
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Notification Channels</h3>
              
              <div className="space-y-4">
                {[
                  { id: 'email', label: 'Email Notifications', description: 'Receive alerts via email', icon: Mail, enabled: true },
                  { id: 'push', label: 'Push Notifications', description: 'Browser push notifications', icon: Smartphone, enabled: true },
                  { id: 'slack', label: 'Slack Integration', description: 'Send alerts to Slack channel', icon: Slack, enabled: false },
                  { id: 'webhook', label: 'Webhook', description: 'POST alerts to custom endpoint', icon: Webhook, enabled: false },
                ].map((channel) => (
                  <div key={channel.id} className="flex items-center justify-between p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-lg bg-[rgba(123,97,255,0.15)] flex items-center justify-center">
                        <channel.icon className="w-5 h-5 text-[#7B61FF]" />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-[#F5F7FF]">{channel.label}</p>
                        <p className="text-xs text-[#A7B1C8]">{channel.description}</p>
                      </div>
                    </div>
                    <button
                      disabled={!canManageFlags}
                      className={cn(
                        "w-12 h-6 rounded-full transition-colors relative disabled:opacity-50 disabled:cursor-not-allowed",
                        channel.enabled ? "bg-[#7B61FF]" : "bg-[rgba(123,97,255,0.2)]"
                      )}
                    >
                      <span className={cn(
                        "absolute top-1 w-4 h-4 rounded-full bg-white transition-all",
                        channel.enabled ? "left-7" : "left-1"
                      )} />
                    </button>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-[rgba(123,97,255,0.12)]">
                <h4 className="text-sm font-medium text-[#F5F7FF] mb-4">Alert Types</h4>
                <div className="space-y-3">
                  {[
                    { label: 'System Errors', enabled: true },
                    { label: 'High Report Volume', enabled: true },
                    { label: 'Revenue Milestones', enabled: true },
                    { label: 'New User Signups', enabled: false },
                    { label: 'Security Alerts', enabled: true },
                  ].map((alert) => (
                    <div key={alert.label} className="flex items-center justify-between">
                      <span className="text-sm text-[#A7B1C8]">{alert.label}</span>
                      <input
                        type="checkbox"
                        defaultChecked={alert.enabled}
                        disabled={!canManageFlags}
                        className="w-4 h-4 rounded border-[rgba(123,97,255,0.3)] bg-transparent text-[#7B61FF] focus:ring-[#7B61FF]"
                      />
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'security' && (
            <div className="glass-panel p-6 space-y-6 animate-fade-in">
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Security Settings</h3>
              
              <div className="space-y-4">
                <div className="p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <p className="text-sm font-medium text-[#F5F7FF]">Two-Factor Authentication</p>
                      <p className="text-xs text-[#A7B1C8]">Require 2FA for all admin accounts</p>
                    </div>
                    <button
                      disabled={!canDisableAdmin2FA}
                      className="w-12 h-6 rounded-full bg-[#7B61FF] relative disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <span className="absolute top-1 left-7 w-4 h-4 rounded-full bg-white" />
                    </button>
                  </div>
                </div>

                <div className="p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <p className="text-sm font-medium text-[#F5F7FF]">IP Whitelist</p>
                      <p className="text-xs text-[#A7B1C8]">Restrict access to specific IP addresses</p>
                    </div>
                    <button
                      disabled={!canManageFlags}
                      className="w-12 h-6 rounded-full bg-[rgba(123,97,255,0.2)] relative disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <span className="absolute top-1 left-1 w-4 h-4 rounded-full bg-white" />
                    </button>
                  </div>
                </div>

                <div className="p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <p className="text-sm font-medium text-[#F5F7FF]">Password Policy</p>
                      <p className="text-xs text-[#A7B1C8]">Enforce strong password requirements</p>
                    </div>
                    <button
                      disabled={!canManageFlags}
                      className="w-12 h-6 rounded-full bg-[#7B61FF] relative disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <span className="absolute top-1 left-7 w-4 h-4 rounded-full bg-white" />
                    </button>
                  </div>
                </div>

                <div className="p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <label className="block text-sm text-[#A7B1C8] mb-2">Session Duration (hours)</label>
                  <input
                    type="number"
                    defaultValue="8"
                    min="1"
                    max="24"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] focus:border-[#7B61FF] focus:outline-none"
                  />
                </div>

                <div className="p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                  <label className="block text-sm text-[#A7B1C8] mb-2">Max Login Attempts</label>
                  <input
                    type="number"
                    defaultValue="5"
                    min="3"
                    max="10"
                    disabled={!canChangeLimits}
                    className="w-full px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] focus:border-[#7B61FF] focus:outline-none"
                  />
                </div>
              </div>
            </div>
          )}

          {activeTab === 'integrations' && (
            <div className="glass-panel p-6 space-y-6 animate-fade-in">
              <h3 className="text-lg font-semibold text-[#F5F7FF]">Integrations</h3>
              
              <div className="space-y-4">
                {[
                  { name: 'Stripe', description: 'Payment processing', connected: true, icon: '💳' },
                  { name: 'SendGrid', description: 'Email delivery', connected: true, icon: '📧' },
                  { name: 'AWS S3', description: 'File storage', connected: true, icon: '📦' },
                  { name: 'Google Analytics', description: 'Analytics tracking', connected: false, icon: '📊' },
                  { name: 'Slack', description: 'Team notifications', connected: false, icon: '💬' },
                ].map((integration) => (
                  <div key={integration.name} className="flex items-center justify-between p-4 rounded-lg bg-[rgba(14,19,32,0.5)]">
                    <div className="flex items-center gap-4">
                      <span className="text-2xl">{integration.icon}</span>
                      <div>
                        <p className="text-sm font-medium text-[#F5F7FF]">{integration.name}</p>
                        <p className="text-xs text-[#A7B1C8]">{integration.description}</p>
                      </div>
                    </div>
                    <button
                      disabled={!canManageFlags}
                      className={cn(
                        "px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed",
                        integration.connected
                          ? "bg-[rgba(255,107,107,0.15)] text-[#FF6B6B] border border-[rgba(255,107,107,0.25)] hover:bg-[rgba(255,107,107,0.25)]"
                          : "btn-primary"
                      )}
                    >
                      {integration.connected ? 'Disconnect' : 'Connect'}
                    </button>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-[rgba(123,97,255,0.12)]">
                <h4 className="text-sm font-medium text-[#F5F7FF] mb-4">Webhook URL</h4>
                <div className="flex gap-3">
                  <input
                    type="text"
                    placeholder="https://your-app.com/webhook"
                    disabled={!canManageFlags}
                    className="flex-1 px-4 py-2.5 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none"
                  />
                  <button
                    disabled={!canManageFlags}
                    className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Test
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
