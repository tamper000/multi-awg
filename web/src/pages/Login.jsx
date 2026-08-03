import { useState } from 'preact/hooks'
import { api, jsonBody, session } from '../api.js'
import { Brand, Icon, Notice } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function Login({ onLogin }) {
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(event) {
    event.preventDefault()
    setLoading(true)
    setError('')
    const form = new FormData(event.currentTarget)
    try {
      const data = await api('/api/auth/login', { method: 'POST', body: jsonBody({ username: form.get('username'), password: form.get('password') }) })
      session.set(data.token, data.user)
      onLogin(data.user)
      navigate(data.user.role === 'admin' ? '/admin' : '/dashboard')
    } catch (err) {
      setError(err.message === 'invalid credentials' ? 'Неверный логин или пароль' : err.message)
    } finally {
      setLoading(false)
    }
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
        <Notice>{error}</Notice>
        <label>Логин<input name="username" autoComplete="username" required placeholder="Ваш логин" /></label>
        <label>Пароль<input name="password" type="password" autoComplete="current-password" required placeholder="Ваш пароль" /></label>
        <button class="button button-primary button-wide" disabled={loading}>{loading ? 'Входим…' : <>Войти <Icon name="arrow" /></>}</button>
      </form>
      <p class="login-foot">Защищено шифрованием AmneziaWG</p>
    </section>
  </main>
}
