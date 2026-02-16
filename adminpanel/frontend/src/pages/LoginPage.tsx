import { useState } from 'react';
import { 
  Shield, 
  Lock, 
  ArrowRight, 
  Check, 
  Loader2,
  Key,
  Eye,
  EyeOff,
  User
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface LoginPageProps {
  onLogin: () => void;
}

export function LoginPage({ onLogin }: LoginPageProps) {
  const [step, setStep] = useState<'telegram' | '2fa' | 'password'>('telegram');
  const [isLoading, setIsLoading] = useState(false);
  const [twoFaCode, setTwoFaCode] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');

  // Simulate Telegram login
  const handleTelegramLogin = () => {
    setIsLoading(true);
    
    setTimeout(() => {
      setIsLoading(false);
      setStep('2fa');
    }, 2000);
  };

  // Handle 2FA submit
  const handle2FASubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    if (twoFaCode.length < 6) {
      setError('Please enter a valid 6-digit code');
      return;
    }
    
    setIsLoading(true);
    
    setTimeout(() => {
      setIsLoading(false);
      setStep('password');
    }, 1500);
  };

  // Handle password submit
  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    
    setIsLoading(true);
    
    setTimeout(() => {
      setIsLoading(false);
      onLogin();
    }, 1500);
  };

  // Progress indicator
  const getStepProgress = () => {
    switch (step) {
      case 'telegram': return 33;
      case '2fa': return 66;
      case 'password': return 100;
    }
  };

  return (
    <div className="min-h-screen bg-[#070B14] flex items-center justify-center relative overflow-hidden">
      {/* Background Effects */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-[#7B61FF] rounded-full opacity-10 blur-[120px]" />
        <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-[#4CC9F0] rounded-full opacity-10 blur-[100px]" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-[#7B61FF] rounded-full opacity-5 blur-[150px]" />
        <div className="noise-overlay" />
        <div 
          className="absolute inset-0 opacity-[0.03]"
          style={{
            backgroundImage: `
              linear-gradient(rgba(123, 97, 255, 0.3) 1px, transparent 1px),
              linear-gradient(90deg, rgba(123, 97, 255, 0.3) 1px, transparent 1px)
            `,
            backgroundSize: '50px 50px'
          }}
        />
      </div>

      {/* Login Card */}
      <div className="relative z-10 w-full max-w-md mx-4">
        <div className="glass-panel p-8 animate-slide-up">
          {/* Logo */}
          <div className="text-center mb-6">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-[#7B61FF] to-[#4CC9F0] mb-4 shadow-lg shadow-[#7B61FF]/20">
              <Shield className="w-8 h-8 text-white" />
            </div>
            <h1 className="text-2xl font-bold text-[#F5F7FF] mb-1">Heartbeat Admin</h1>
            <p className="text-sm text-[#A7B1C8]">Secure Admin Console</p>
          </div>

          {/* Progress Bar */}
          <div className="mb-6">
            <div className="flex items-center justify-between text-xs text-[#A7B1C8] mb-2">
              <span className={cn(
                "transition-colors",
                step === 'telegram' ? "text-[#7B61FF]" : "text-[#2DD4A8]"
              )}>
                {step === 'telegram' ? '● Telegram' : '✓ Telegram'}
              </span>
              <span className={cn(
                "transition-colors",
                step === '2fa' ? "text-[#7B61FF]" : step === 'password' ? "text-[#2DD4A8]" : ""
              )}>
                {step === 'password' ? '✓ 2FA' : '● 2FA'}
              </span>
              <span className={cn(
                "transition-colors",
                step === 'password' ? "text-[#7B61FF]" : ""
              )}>
                ● Password
              </span>
            </div>
            <div className="h-1.5 rounded-full bg-[rgba(123,97,255,0.1)] overflow-hidden">
              <div 
                className="h-full rounded-full bg-gradient-to-r from-[#7B61FF] to-[#4CC9F0] transition-all duration-500"
                style={{ width: `${getStepProgress()}%` }}
              />
            </div>
          </div>

          {/* Step 1: Telegram Login */}
          {step === 'telegram' && (
            <div className="space-y-6 animate-fade-in">
              <div className="text-center">
                <h2 className="text-lg font-semibold text-[#F5F7FF] mb-2">Sign in with Telegram</h2>
                <p className="text-sm text-[#A7B1C8]">
                  Click the button below to authenticate via Telegram
                </p>
              </div>

              <button
                onClick={handleTelegramLogin}
                disabled={isLoading}
                className={cn(
                  "w-full relative overflow-hidden group",
                  "flex items-center justify-center gap-3",
                  "px-6 py-4 rounded-xl",
                  "bg-gradient-to-r from-[#0088cc] to-[#0099dd]",
                  "text-white font-medium",
                  "transition-all duration-300",
                  "hover:shadow-lg hover:shadow-[#0088cc]/25 hover:-translate-y-0.5",
                  "disabled:opacity-70 disabled:cursor-not-allowed disabled:hover:translate-y-0"
                )}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    <span>Connecting to Telegram...</span>
                  </>
                ) : (
                  <>
                    <svg 
                      className="w-5 h-5" 
                      viewBox="0 0 24 24" 
                      fill="currentColor"
                    >
                      <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/>
                    </svg>
                    <span>Continue with Telegram</span>
                    <ArrowRight className="w-5 h-5 transition-transform group-hover:translate-x-1" />
                  </>
                )}
              </button>

              {/* Security Note */}
              <div className="flex items-center gap-3 p-4 rounded-xl bg-[rgba(14,19,32,0.5)] border border-[rgba(123,97,255,0.1)]">
                <div className="w-10 h-10 rounded-lg bg-[rgba(123,97,255,0.15)] flex items-center justify-center flex-shrink-0">
                  <Lock className="w-5 h-5 text-[#7B61FF]" />
                </div>
                <div>
                  <p className="text-sm font-medium text-[#F5F7FF]">Secure Connection</p>
                  <p className="text-xs text-[#A7B1C8]">Your data is encrypted end-to-end</p>
                </div>
              </div>
            </div>
          )}

          {/* Step 2: 2FA Code */}
          {step === '2fa' && (
            <form onSubmit={handle2FASubmit} className="space-y-6 animate-fade-in">
              <div className="text-center">
                <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-[rgba(123,97,255,0.15)] mb-4">
                  <Key className="w-6 h-6 text-[#7B61FF]" />
                </div>
                <h2 className="text-lg font-semibold text-[#F5F7FF] mb-2">
                  Two-Factor Authentication
                </h2>
                <p className="text-sm text-[#A7B1C8]">
                  Enter the 6-digit code from your authenticator app
                </p>
              </div>

              {/* Success Badge */}
              <div className="flex items-center gap-3 p-3 rounded-xl bg-[rgba(45,212,168,0.08)] border border-[rgba(45,212,168,0.2)]">
                <div className="w-8 h-8 rounded-full bg-[#2DD4A8] flex items-center justify-center">
                  <Check className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className="text-sm font-medium text-[#2DD4A8]">Telegram Verified</p>
                  <p className="text-xs text-[#A7B1C8]">@admin_user</p>
                </div>
              </div>

              {/* 2FA Input */}
              <div className="space-y-2">
                <label className="block text-sm text-[#A7B1C8]">2FA Code</label>
                <div className="relative">
                  <div className="absolute left-4 top-1/2 -translate-y-1/2">
                    <Key className="w-5 h-5 text-[#A7B1C8]" />
                  </div>
                  <input
                    type="text"
                    value={twoFaCode}
                    onChange={(e) => setTwoFaCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                    placeholder="000000"
                    maxLength={6}
                    inputMode="numeric"
                    autoFocus
                    className={cn(
                      "w-full pl-12 pr-4 py-4 rounded-xl",
                      "text-center text-2xl tracking-[0.5em] font-mono",
                      "bg-[rgba(14,19,32,0.8)] border-2",
                      "text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.3)]",
                      "transition-all duration-200",
                      "focus:outline-none focus:border-[#7B61FF] focus:ring-4 focus:ring-[rgba(123,97,255,0.15)]",
                      error && "border-[#FF6B6B] focus:border-[#FF6B6B] focus:ring-[rgba(255,107,107,0.15)]"
                    )}
                  />
                </div>
                {error && (
                  <p className="text-sm text-[#FF6B6B] flex items-center gap-1">
                    <span className="inline-block w-1 h-1 rounded-full bg-[#FF6B6B]" />
                    {error}
                  </p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isLoading || twoFaCode.length < 6}
                className={cn(
                  "w-full flex items-center justify-center gap-2",
                  "px-6 py-4 rounded-xl",
                  "bg-gradient-to-r from-[#7B61FF] to-[#9B81FF]",
                  "text-white font-medium",
                  "transition-all duration-300",
                  "hover:shadow-lg hover:shadow-[#7B61FF]/25 hover:-translate-y-0.5",
                  "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
                )}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    <span>Verifying...</span>
                  </>
                ) : (
                  <>
                    <span>Continue</span>
                    <ArrowRight className="w-5 h-5" />
                  </>
                )}
              </button>

              {/* Back Button */}
              <button
                type="button"
                onClick={() => {
                  setStep('telegram');
                  setTwoFaCode('');
                  setError('');
                }}
                className="w-full text-sm text-[#A7B1C8] hover:text-[#F5F7FF] transition-colors"
              >
                Back to Telegram
              </button>
            </form>
          )}

          {/* Step 3: Password */}
          {step === 'password' && (
            <form onSubmit={handlePasswordSubmit} className="space-y-6 animate-fade-in">
              <div className="text-center">
                <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-[rgba(45,212,168,0.15)] mb-4">
                  <Lock className="w-6 h-6 text-[#2DD4A8]" />
                </div>
                <h2 className="text-lg font-semibold text-[#F5F7FF] mb-2">
                  Enter Your Password
                </h2>
                <p className="text-sm text-[#A7B1C8]">
                  Provide your admin password to access the dashboard
                </p>
              </div>

              {/* Success Badge */}
              <div className="flex items-center gap-3 p-3 rounded-xl bg-[rgba(45,212,168,0.08)] border border-[rgba(45,212,168,0.2)]">
                <div className="w-8 h-8 rounded-full bg-[#2DD4A8] flex items-center justify-center">
                  <Check className="w-5 h-5 text-white" />
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-[#2DD4A8]">2FA Verified</p>
                  <p className="text-xs text-[#A7B1C8]">Authentication successful</p>
                </div>
              </div>

              {/* Password Input */}
              <div className="space-y-2">
                <label className="block text-sm text-[#A7B1C8]">Password</label>
                <div className="relative">
                  <div className="absolute left-4 top-1/2 -translate-y-1/2">
                    <User className="w-5 h-5 text-[#A7B1C8]" />
                  </div>
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Enter your password"
                    autoFocus
                    className={cn(
                      "w-full pl-12 pr-12 py-4 rounded-xl",
                      "text-base",
                      "bg-[rgba(14,19,32,0.8)] border-2",
                      "text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.3)]",
                      "transition-all duration-200",
                      "focus:outline-none focus:border-[#7B61FF] focus:ring-4 focus:ring-[rgba(123,97,255,0.15)]",
                      error && "border-[#FF6B6B] focus:border-[#FF6B6B] focus:ring-[rgba(255,107,107,0.15)]"
                    )}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-4 top-1/2 -translate-y-1/2 text-[#A7B1C8] hover:text-[#F5F7FF] transition-colors"
                  >
                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                </div>
                {error && (
                  <p className="text-sm text-[#FF6B6B] flex items-center gap-1">
                    <span className="inline-block w-1 h-1 rounded-full bg-[#FF6B6B]" />
                    {error}
                  </p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isLoading || password.length < 8}
                className={cn(
                  "w-full flex items-center justify-center gap-2",
                  "px-6 py-4 rounded-xl",
                  "bg-gradient-to-r from-[#2DD4A8] to-[#3DE5B9]",
                  "text-white font-medium",
                  "transition-all duration-300",
                  "hover:shadow-lg hover:shadow-[#2DD4A8]/25 hover:-translate-y-0.5",
                  "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
                )}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    <span>Accessing Dashboard...</span>
                  </>
                ) : (
                  <>
                    <span>Access Dashboard</span>
                    <ArrowRight className="w-5 h-5" />
                  </>
                )}
              </button>

              {/* Back Button */}
              <button
                type="button"
                onClick={() => {
                  setStep('2fa');
                  setPassword('');
                  setError('');
                }}
                className="w-full text-sm text-[#A7B1C8] hover:text-[#F5F7FF] transition-colors"
              >
                Back to 2FA
              </button>

              {/* Forgot Password */}
              <div className="pt-2 text-center">
                <a href="#" className="text-xs text-[#7B61FF] hover:underline">
                  Forgot password?
                </a>
              </div>
            </form>
          )}
        </div>

        {/* Footer */}
        <div className="mt-6 text-center">
          <p className="text-xs text-[#A7B1C8]">
            Protected by enterprise-grade security
          </p>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="flex items-center gap-1 text-[10px] text-[#A7B1C8]">
              <div className="w-1.5 h-1.5 rounded-full bg-[#2DD4A8]" />
              SSL Secure
            </div>
            <div className="flex items-center gap-1 text-[10px] text-[#A7B1C8]">
              <div className="w-1.5 h-1.5 rounded-full bg-[#2DD4A8]" />
              End-to-End Encrypted
            </div>
            <div className="flex items-center gap-1 text-[10px] text-[#A7B1C8]">
              <div className="w-1.5 h-1.5 rounded-full bg-[#2DD4A8]" />
              SOC 2 Compliant
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
