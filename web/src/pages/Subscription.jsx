import { useEffect, useState } from 'preact/hooks'
import { api } from '../api.js'
import { Brand, copyText, Icon, Loader, Notice } from '../ui.jsx'
import { navigate } from '../App.jsx'

export default function Subscription({ token, user }) {
  const [data, setData] = useState(null)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState('')
  useEffect(() => { api(`/api/sub/${token}`).then(setData).catch((err) => setError(err.message)) }, [token])
  const copy = (kind, value) => copyText(value, (state) => setCopied(state ? kind : ''))
  const mihomoUrl = `${location.origin}/api/sub/${token}/mihomo`

  return <div class="subscription-page">
    <header class="sub-header"><Brand compact /><button class="button button-ghost" onClick={() => navigate(user?.role === 'admin' ? '/admin' : user ? '/dashboard' : '/login')}><Icon name="back" /> {user ? 'В кабинет' : 'Войти'}</button></header>
    <main class="sub-main">
      <div class="sub-intro"><span class="eyebrow">Готово к подключению</span><h1>{data ? data.name : 'Подписка'}</h1><p>Выберите удобный способ. Настройки обновляются по этой ссылке автоматически.</p></div>
      <Notice>{error}</Notice>
      {!data && !error ? <Loader /> : data && <>
        <section class="method-grid">
          <button class="method-card method-primary" onClick={() => copy('mihomo', mihomoUrl)}><span class="method-number">01</span><span class="method-icon"><Icon name="link" size={28} /></span><h2>Mihomo / Clash</h2><p>Рекомендуем. Скопируйте URL подписки в совместимый клиент — автоматически обходит российские приложения (не проксирует их).</p><strong>{copied === 'mihomo' ? 'Скопировано' : 'Копировать ссылку'} <Icon name="copy" /></strong></button>
          <a class="method-card" href={data.vpn_link}><span class="method-number">02</span><span class="method-icon purple"><Icon name="bolt" size={28} /></span><h2>Открыть в Amnezia</h2><p>Быстрый импорт в приложение одним нажатием.</p><strong>Открыть приложение <Icon name="arrow" /></strong></a>
          <a class="method-card" href={`/api/sub/${token}/conf`}><span class="method-number">03</span><span class="method-icon purple"><Icon name="download" size={28} /></span><h2>Скачать .conf</h2><p>Файл для ручного импорта в AmneziaWG.</p><strong>Скачать файл <Icon name="download" /></strong></a>
        </section>
        <details class="config-details"><summary><span><Icon name="eye" /> Показать конфигурацию</span><Icon name="arrow" /></summary><div><button class="button button-ghost" onClick={() => copy('conf', data.conf)}><Icon name="copy" /> {copied === 'conf' ? 'Скопировано' : 'Копировать конфиг'}</button><pre>{data.conf}</pre></div></details>
      </>}
    </main>
  </div>
}
