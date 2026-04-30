import { Film, KeyRound, Sparkles } from 'lucide-react'
import { useState } from 'react'
import type { FormEvent } from 'react'
import { useLogin, useRegister } from '../../api/hooks'
import { useAuthStore } from '../../state/authStore'

export function AuthPage() {
  const setSession = useAuthStore((state) => state.setSession)
  const loginMutation = useLogin()
  const registerMutation = useRegister()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  const activeMutation = mode === 'login' ? loginMutation : registerMutation

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
      { display_name: displayName, email, password },
      {
        onError: (error) => setErrorMessage(error.message),
        onSuccess: (session) => setSession(session),
      },
    )
  }

  return (
    <main className="auth-shell">
      <section className="auth-hero-card">
        <span className="section-kicker">Dramora Studio</span>
        <h1>登录你的导演台，继续推进 AI 漫剧生产。</h1>
        <p>
          这一版先把 JWT 登录链路接通到 Studio：你可以注册导演账号、恢复会话，并用同一套 token
          继续后续的组织鉴权扩展。
        </p>
        <div className="auth-feature-grid">
          <div>
            <Film aria-hidden="true" />
            <strong>Production cockpit</strong>
            <small>从故事解析到 Storyboard 的导演台保持同一登录态。</small>
          </div>
          <div>
            <KeyRound aria-hidden="true" />
            <strong>JWT session</strong>
            <small>本地持久化 token，刷新页面后会自动恢复会话。</small>
          </div>
          <div>
            <Sparkles aria-hidden="true" />
            <strong>Ready for auth hardening</strong>
            <small>后续可以在这条链路上继续加 organization 级访问控制。</small>
          </div>
        </div>
      </section>

      <section className="auth-form-card">
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
      </section>
    </main>
  )
}
