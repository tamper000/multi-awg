import { useEffect, useState } from 'preact/hooks'
import { api, jsonBody } from '../api.js'
import { Empty, formatBytes, formatDate, Icon, Loader, Modal, Notice } from '../ui.jsx'
import { navigate } from '../App.jsx'
import Layout from './Layout.jsx'

export default function UserPanel({ user, onLogout }) {
  const [info, setInfo] = useState(null)
  const [configs, setConfigs] = useState(null)
  const [error, setError] = useState('')
  const [modal, setModal] = useState('')
  const [createError, setCreateError] = useState('')
  const [pwError, setPwError] = useState('')
  const [pwDone, setPwDone] = useState(false)

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
      const created = await api('/api/user/configs', { method: 'POST', body: jsonBody({ name: form.get('name') }) })
      setModal(''); navigate(`/subscription/${created.sub_token}`)
    } catch (err) { setCreateError(err.message) }
  }

  async function remove(name) {
    if (!confirm(`Удалить конфиг «${name}»? Ссылка подписки перестанет работать.`)) return
    try { await api(`/api/user/configs/${encodeURIComponent(name)}`, { method: 'DELETE' }); load() } catch (err) { setError(err.message) }
  }

  async function changePassword(event) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    setPwError(''); setPwDone(false)
    try {
      await api('/api/user/password', { method: 'POST', body: jsonBody({ old: form.get('old'), new: form.get('new') }) })
      setPwDone(true)
    } catch (err) { setPwError(err.message === 'invalid credentials' ? 'Неверный текущий пароль' : err.message) }
  }

  const expired = info?.days_left === 0
  return <Layout user={user} onLogout={onLogout} title={`Привет, ${user.username}`} subtitle="Ваши защищённые подключения в одном месте" action={<button class="button button-primary" onClick={() => setModal('create')} disabled={expired}><Icon name="plus" /> Новый конфиг</button>}>
    <Notice>{error}</Notice>
    {expired && <Notice kind="warning">Подписка закончилась. Существующие конфиги доступны, но создать новый пока нельзя.</Notice>}
    {!info || !configs ? <Loader /> : <>
      <section class="stats-grid">
        <article class="stat-card accent-lime"><small>Подписка</small><strong>{info.days_left < 0 ? 'Без срока' : `${info.days_left} дн.`}</strong><span>{formatDate(info.expires_at)}</span></article>
        <article class="stat-card accent-purple"><small>Устройства</small><strong>{configs.length}</strong><span>Активных конфигов</span></article>
        <article class="stat-card"><small>Общий трафик</small><strong>{formatBytes(configs.reduce((sum, item) => sum + (item.received || 0) + (item.sent || 0), 0))}</strong><span>Получено и отправлено</span></article>
      </section>
      <section class="section-block">
        <div class="section-title"><div><h2>Мои устройства</h2><p>Нажмите на конфиг, чтобы открыть способы подключения</p></div><button class="button button-ghost" onClick={() => setModal('password')}><Icon name="key" /> Сменить пароль</button></div>
        {!configs.length ? <Empty title="Устройств пока нет" text="Создайте первый конфиг и подключитесь за пару минут." /> : <div class="config-grid">{configs.map((config) => <article class="config-card" key={config.name}>
          <button class="config-main" onClick={() => navigate(`/subscription/${config.sub_token}`)}><span class="device-icon"><Icon name="bolt" /></span><span><strong>{config.name}</strong><small>Получено {formatBytes(config.received)} · Отдано {formatBytes(config.sent)}</small></span><Icon name="arrow" /></button>
          <button class="delete-button" onClick={() => remove(config.name)} aria-label={`Удалить ${config.name}`}><Icon name="trash" /></button>
        </article>)}</div>}
      </section>
    </>}
    {modal === 'create' && <Modal title="Новый конфиг" onClose={() => setModal('')}>{createError ? <>
      <Notice>{createError === 'config limit reached' ? 'Достигнут лимит конфигов' : createError}</Notice>
      <button class="button button-primary button-wide" onClick={() => setModal('')}>Закрыть</button>
    </> : <>
      <p>Назовите устройство, чтобы потом легко его узнать.</p><form class="form-stack" onSubmit={create}><label>Название<input name="name" required pattern="[A-Za-zА-Яа-яЁё0-9]+" placeholder="Например, Телефон" autoFocus /></label><button class="button button-primary button-wide"><Icon name="plus" /> Создать подключение</button></form>
    </>}</Modal>}
    {modal === 'password' && <Modal title="Смена пароля" onClose={() => { setModal(''); setPwError(''); setPwDone(false) }}><Notice>{pwError}</Notice>{pwDone ? <p class="pw-done">Пароль изменён.</p> : <form class="form-stack" onSubmit={changePassword}><label>Текущий пароль<input type="password" name="old" required autoFocus /></label><label>Новый пароль<input type="password" name="new" required /></label><button class="button button-primary button-wide">Сохранить пароль</button></form>}</Modal>}
  </Layout>
}
