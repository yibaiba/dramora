import { Film, KeyRound, LayoutPanelTop, ShieldCheck, Sparkles } from 'lucide-react'
import { useState } from 'react'
import type { FormEvent } from 'react'
import { useLogin, useRegister } from '../../api/hooks'
import { useAuthStore } from '../../state/authStore'

const LOCAL_ADMIN_EMAIL = 'admin@local.dev'
const LOCAL_ADMIN_PASSWORD = 'strongpass123'

export function AuthPage() {
  const setSession = useAuthStore((state) => state.setSession)
  const loginMutation = useLogin()
  const registerMutation = useRegister()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [invitationToken, setInvitationToken] = useState(() => {
    if (typeof window === 'undefined') return ''
    const params = new URLSearchParams(window.location.search)
    return params.get('invite') ?? ''
  })
  const [errorMessage, setErrorMessage] = useState('')

  const activeMutation = mode === 'login' ? loginMutation : registerMutation

  const useLocalAdminPreset = () => {
    setMode('login')
    setEmail(LOCAL_ADMIN_EMAIL)
    setPassword(LOCAL_ADMIN_PASSWORD)
    setErrorMessage('')
  }

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setErrorMessage('')
    if (mode === 'login') {
      loginMutation.mutate(
        { email, password },
        {
          onError: (error) => setErrorMessage(error.message),
          onSuccess: (session) => setSession(session),
        },
      )
      return
    }

    registerMutation.mutate(
      {
        display_name: displayName,
        email,
        password,
        ...(invitationToken.trim() ? { invitation_token: invitationToken.trim() } : {}),
      },
      {
        onError: (error) => setErrorMessage(error.message),
        onSuccess: (session) => setSession(session),
      },
    )
  }

  return (
    <main className="auth-shell auth-shell--revamp">
      <section className="auth-hero-card auth-hero-card--revamp">
        <div className="auth-brand-row">
          <span className="section-kicker">Dramora Studio</span>
          <span className="auth-brand-badge">创作控制中心</span>
        </div>
        <div className="auth-headline-stack">
          <h1>把导演台变回真正可用的 AI 漫剧工作流入口。</h1>
          <p>
            登录后直接进入统一工作台：项目、剧集、故事解析、分镜、资产、导出和通知都在一条可追踪的生产链里。
          </p>
        </div>
        <div className="auth-preview-panel" aria-label="工作台亮点预览">
          <div className="auth-preview-surface">
            <div className="auth-preview-topline">
              <span>今日</span>
              <strong>导演控制台</strong>
            </div>
            <div className="auth-preview-headline">
              <strong>从故事解析到分镜台的关键决策，都在首页可见。</strong>
              <small>更少的无效入口，更强的起手引导，更清晰的下一步。</small>
            </div>
            <div className="auth-preview-chip-row">
              <span>
                <LayoutPanelTop aria-hidden="true" />
                首页总览
              </span>
              <span>
                <Film aria-hidden="true" />
                分镜生产
              </span>
              <span>
                <Sparkles aria-hidden="true" />
                智能体协作
              </span>
            </div>
          </div>
          <div className="auth-preview-metrics">
            <div>
              <span>核心页面</span>
              <strong>6 个</strong>
              <small>围绕真实制作路径重新收束</small>
            </div>
            <div>
              <span>默认本地管理员</span>
              <strong>已就绪</strong>
              <small>启动即登录，方便快速验收 UI</small>
            </div>
          </div>
        </div>
        <div className="auth-feature-grid auth-feature-grid--revamp">
          <div>
            <Film aria-hidden="true" />
            <strong>生产控制台</strong>
            <small>把首页做成导演台而不是文档页，第一屏就给出明确下一步。</small>
          </div>
          <div>
            <KeyRound aria-hidden="true" />
            <strong>会话保持</strong>
            <small>本地持久化 token，刷新页面后自动恢复会话，不再反复重登。</small>
          </div>
          <div>
            <ShieldCheck aria-hidden="true" />
            <strong>本地管理员已就绪</strong>
            <small>本地环境默认带管理员账号，方便直接检查首页与主工作台。</small>
          </div>
        </div>
      </section>

      <section className="auth-form-card auth-form-card--revamp">
        <div className="auth-form-header">
          <div>
            <span className="section-kicker">访问入口</span>
            <h2>{mode === 'login' ? '进入导演台' : '创建导演账号'}</h2>
          </div>
          {mode === 'login' ? (
            <button className="auth-preset-button" onClick={useLocalAdminPreset} type="button">
              使用本地管理员
            </button>
          ) : null}
        </div>
        <div className="auth-mode-switch">
          <button
            className={mode === 'login' ? 'active' : ''}
            onClick={() => setMode('login')}
            type="button"
          >
            登录
          </button>
          <button
            className={mode === 'register' ? 'active' : ''}
            onClick={() => setMode('register')}
            type="button"
          >
            注册
          </button>
        </div>

        <form className="auth-form" onSubmit={submit}>
          {mode === 'register' ? (
            <label>
              <span>显示名</span>
              <input
                onChange={(event) => setDisplayName(event.target.value)}
                placeholder="Lin Yifei"
                required
                value={displayName}
              />
            </label>
          ) : null}

          {mode === 'register' ? (
            <label>
              <span>邀请码（可选）</span>
              <input
                onChange={(event) => setInvitationToken(event.target.value)}
                placeholder="留空则自动创建你的工作台"
                value={invitationToken}
              />
            </label>
          ) : null}

          <label>
            <span>Email</span>
            <input
              autoComplete="email"
              onChange={(event) => setEmail(event.target.value)}
              placeholder="director@dramora.ai"
              required
              type="email"
              value={email}
            />
          </label>

          <label>
            <span>Password</span>
            <input
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              minLength={8}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="至少 8 位"
              required
              type="password"
              value={password}
            />
          </label>

          {errorMessage ? <p className="auth-error">{errorMessage}</p> : null}

          <button className="auth-submit-button" disabled={activeMutation.isPending} type="submit">
            {mode === 'login' ? '进入导演台' : '创建导演账号'}
          </button>
        </form>
        <div className="auth-helper-note">
          <strong>本地默认账号</strong>
          <span>
            {LOCAL_ADMIN_EMAIL} / {LOCAL_ADMIN_PASSWORD}
          </span>
        </div>
      </section>
    </main>
  )
}
