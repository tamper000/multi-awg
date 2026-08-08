import { useEffect, useState } from 'preact/hooks'
import { api, jsonBody } from '../api.js'
import { Empty, formatBytes, formatDate, Icon, Loader, Modal, Notice } from '../ui.jsx'
import Layout from './Layout.jsx'

export default function UserPanel({ user, onLogout }) {
  const [info, setInfo] = useState(null)
  const [configs, setConfigs] = useState(null)
  const [error, setError] = useState('')
  const [modal, setModal] = useState('')
  const [pwDone, setPwDone] = useState(false)
  const [copied, setCopied] = useState(false)
  const [subscriptionCopied, setSubscriptionCopied] = useState(false)

  async function load() {
    setError('')
    try {
      const [nextInfo, nextConfigs] = await Promise.all([api('/api/user/info'), api('/api/user/configs')])
      setInfo(nextInfo); setConfigs(nextConfigs)
    } catch (err) { setError(err.message) }
  }
  useEffect(() => { load() }, [])

  async function create(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    try {
      await api('/api/user/configs', { method: 'POST', body: jsonBody({ name: form.get('name') }) })
      setModal(''); load()
    } catch {}
  }

  async function remove(name) {
    if (!confirm(`Удалить конфиг «${name}»? Ссылка подписки перестанет работать.`)) return
    try { await api(`/api/user/configs/${encodeURIComponent(name)}`, { method: 'DELETE' }); load() } catch {}
  }

  async function changePassword(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    setPwDone(false)
    try {
      await api('/api/user/password', { method: 'POST', body: jsonBody({ old: form.get('old'), new: form.get('new') }) })
      setPwDone(true)
    } catch {}
  }

  async function copySubscription(value) {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value)
        return true
      }
    } catch {}

    try {
      const textarea = document.createElement('textarea')
      textarea.value = value
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      const copied = document.execCommand('copy')
      textarea.remove()
      return copied
    } catch {
      return false
    }
  }

  async function openSubscriptionHelp() {
    const copied = await copySubscription(subscriptionUrl)
    setCopied(copied)
    if (copied) setTimeout(() => setCopied(false), 1600)
    setSubscriptionCopied(false)
    setModal('subscription')
  }

  async function copySubscriptionAgain() {
    setSubscriptionCopied(await copySubscription(subscriptionUrl))
  }

  const expired = info?.days_left === 0
  const subscriptionUrl = info?.subscription_url && `${location.origin}${info.subscription_url}`
  return <Layout user={user} onLogout={onLogout} title={`Привет, ${user.username}`} subtitle="Ваши защищённые подключения в одном месте" action={<div class="user-actions"><button class="button button-primary" onClick={() => setModal('create')} disabled={expired}><Icon name="plus" /> Новый конфиг</button>{subscriptionUrl && <button class="button button-ghost" onClick={openSubscriptionHelp}><Icon name="link" /> {copied ? 'Скопировано' : 'Ссылка для подключения'}</button>}</div>}>
    <Notice>{error}</Notice>
    {expired && <Notice kind="warning">Подписка истекла: существующие конфиги заморожены, новые создать нельзя.</Notice>}
    {!info || !configs ? <Loader /> : <>
      <section class="stats-grid">
        <article class="stat-card accent-lime"><small>Подписка</small><strong>{info.days_left < 0 ? 'Без срока' : `${info.days_left} дн.`}</strong><span>{formatDate(info.expires_at)}</span></article>
        <article class="stat-card accent-purple"><small>Устройства</small><strong>{configs.length}</strong><span>Активных конфигов</span></article>
        <article class="stat-card"><small>Общий трафик</small><strong>{formatBytes(configs.reduce((sum, item) => sum + (item.received || 0) + (item.sent || 0), 0))}</strong><span>Получено и отправлено</span></article>
      </section>
      <section class="section-block">
        <div class="section-title"><div><h2>Мои устройства</h2><p>Нажмите на конфиг, чтобы открыть способы подключения</p></div><button class="button button-ghost" onClick={() => setModal('password')}><Icon name="key" /> Сменить пароль</button></div>
        <div class="device-rule"><Icon name="bolt" /><strong>1 конфиг = 1 устройство</strong><span>Для другого устройства создайте отдельный конфиг</span></div>
        {!configs.length ? <Empty title="Устройств пока нет" text="Создайте первый конфиг и подключитесь за пару минут." /> : <div class="config-grid">{configs.map((config) => <article class="config-card" key={config.name}>
          <a class="config-main" href={`${subscriptionUrl?.replace(/\/mihomo$/, '')}/conf?name=${encodeURIComponent(config.name)}`}><span class="device-icon"><Icon name="bolt" /></span><span><strong>{config.name}</strong><small>Получено {formatBytes(config.received)} · Отдано {formatBytes(config.sent)}</small></span><Icon name="download" /></a>
          <button class="delete-button" onClick={() => remove(config.name)} aria-label={`Удалить ${config.name}`}><Icon name="trash" /></button>
        </article>)}</div>}
      </section>
    </>}
    {modal === 'create' && <Modal title="Новый конфиг" onClose={() => setModal('')}>
      <p>Назовите устройство, чтобы потом легко его узнать.</p><form class="form-stack" onSubmit={create}><label>Название<input name="name" required pattern="[A-Za-zА-Яа-яЁё0-9]+" placeholder="Например, Телефон" autoFocus /></label><button class="button button-primary button-wide"><Icon name="plus" /> Создать подключение</button></form>
    </Modal>}
    {modal === 'password' && <Modal title="Смена пароля" onClose={() => { setModal(''); setPwDone(false) }}>{pwDone ? <p class="pw-done">Пароль изменён.</p> : <form class="form-stack" onSubmit={changePassword}><label>Текущий пароль<input type="password" name="old" required autoFocus /></label><label>Новый пароль<input type="password" name="new" required /></label><button class="button button-primary button-wide">Сохранить пароль</button></form>}</Modal>}
    {modal === 'subscription' && <Modal title="Как подключиться" onClose={() => setModal('')}>
      <div class="tutorial">
        <p class="tutorial-lead"><strong>Ссылка уже скопирована.</strong> Осталось добавить её в приложение и выбрать нужный конфиг.</p>

        <div class="tutorial-note"><strong>Не передавайте ссылку другим</strong><span>Она открывает доступ ко всем конфигам вашей подписки.</span></div>

        <section class="tutorial-section">
          <h3>Приложение</h3>
          <p><strong class="tutorial-platform">Android, Windows, Linux и macOS:</strong> <a href="https://github.com/chen08209/FlClash" target="_blank" rel="noreferrer">FlClash</a> · <a href="https://github.com/chen08209/FlClash/releases/latest" target="_blank" rel="noreferrer">скачать последнюю версию</a></p>
          <p><strong class="tutorial-platform">iPhone и iPad:</strong> <a href="https://apps.apple.com/us/app/nextin/id6754002454" target="_blank" rel="noreferrer">Nextin в App Store</a></p>
        </section>

        <ol class="tutorial-steps">
          <li><strong>Добавьте подписку.</strong><span class="tutorial-line">В Nextin вставьте ссылку в поле подписки.</span><span class="tutorial-line">В FlClash откройте <strong>«Профили»</strong>, выберите добавление по URL и вставьте ссылку.</span></li>
          <li><strong>Обновляйте подписку.</strong> После создания нового конфига на сайте нажмите кнопку обновления профиля в приложении. При наличии включите автообновление.</li>
          <li><strong>Выберите конфиг.</strong> Откройте <strong>«Прокси»</strong>, <strong>«Узлы»</strong> или <strong>«Выбрать узел»</strong> и выберите нужное устройство. <strong>1 конфиг = 1 устройство.</strong></li>
          <li><strong>Включите режим «Правило».</strong> В главном меню выберите <strong>«Правило»</strong>, <strong>Rules</strong> или аналогичный режим автоматической маршрутизации.</li>
          <li><strong>Подключитесь.</strong> Включите VPN и проверьте, что соединение установлено.</li>
        </ol>

        <button class="button button-primary button-wide" type="button" onClick={copySubscriptionAgain}><Icon name="copy" /> {subscriptionCopied ? 'Скопировано' : 'Скопировать ссылку ещё раз'}</button>
      </div>
    </Modal>}
  </Layout>
}
