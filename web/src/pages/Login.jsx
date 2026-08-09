import { useEffect, useState } from 'preact/hooks'
import { api, jsonBody, session } from '../api.js'
import { Brand, Icon } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function Login({ onLogin }) {
  const [loading, setLoading] = useState(false)

  async function login(username, password) {
    setLoading(true)
    try {
      const data = await api('/api/auth/login', { method: 'POST', body: jsonBody({ username, password }) })
      session.set(data.token, data.user)
      onLogin(data.user)
      navigate(data.user.role === 'admin' ? '/admin' : '/dashboard')
    } catch {
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const params = new URLSearchParams(location.hash.slice(1))
    const username = params.get('username')
    const password = params.get('password')
    if (!username || !password) return
    history.replaceState({}, '', location.pathname + location.search)
    login(username, password)
  }, [])

  function submit(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    login(form.get('username'), form.get('password'))
  }

  return <main class="login-page">
    <div class="glow glow-lime" /><div class="glow glow-purple" />
    <section class="login-card">
      <Brand />
      <div class="login-copy">
        <span class="eyebrow">Личный кабинет</span>
        <h1>Быстрый интернет.<br /><em>Без лишних границ.</em></h1>
        <p>Управляйте защищёнными подключениями на всех своих устройствах.</p>
      </div>
      <form onSubmit={submit} class="form-stack">
        <label>Логин<input name="username" autoComplete="username" required placeholder="Ваш логин" /></label>
        <label>Пароль<input name="password" type="password" autoComplete="current-password" required placeholder="Ваш пароль" /></label>
        <button class="button button-primary button-wide" disabled={loading}>{loading ? 'Входим…' : <>Войти <Icon name="arrow" /></>}</button>
      </form>
      <p class="login-foot">Защищено шифрованием AmneziaWG</p>
    </section>
  </main>
}
