import { useState, useEffect, useCallback, useRef } from 'react'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Sectors from './pages/Sectors'
import Watering from './pages/Watering'
import Plants from './pages/Plants'
import Reports from './pages/Reports'
import Operators from './pages/Operators'
import NotifDrawer from './pages/Notifications'
import Shell from './components/Shell'
import { ICONS } from './components/icons'
import { api, connectWS } from './api'

// fallback: pull the user UUID from the JWT `sub` claim for sessions
// created before the id was persisted to localStorage.
function userIdFromToken() {
  const t = localStorage.getItem('token')
  if (!t) return ''
  try {
    const b64 = t.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    return JSON.parse(atob(b64)).sub || ''
  } catch {
    return ''
  }
}

function Toast({ toast, onDismiss }) {
  useEffect(() => {
    if (!toast) return
    const t = setTimeout(onDismiss, 4000)
    return () => clearTimeout(t)
  }, [toast, onDismiss])
  if (!toast) return null
  return (
      <div className={`toast toast-${toast.kind || 'info'}`}>
        <div className="toast-dot" />
        <div style={{ flex: 1 }}>{toast.msg}</div>
        <button className="toast-close" onClick={onDismiss}>{ICONS.x}</button>
      </div>
  )
}

export default function App() {
  const [loggedIn, setLoggedIn] = useState(!!localStorage.getItem('token'))
  const [page, setPage] = useState(() => localStorage.getItem('pc-page') || 'dashboard')
  const [sectors, setSectors] = useState([])
  const [plants, setPlants] = useState([])
  const [telemetry, setTelemetry] = useState({})
  const [notifs, setNotifs] = useState([])
  const [notifsOpen, setNotifsOpen] = useState(false)
  const [selectedSectorId, setSelectedSectorId] = useState(null)
  const [toast, setToast] = useState(null)
  const [wsConnected, setWsConnected] = useState(false)
  const [wsEvents, setWsEvents] = useState([])
  const pendingTelemetry = useRef({})

  const user = {
    id: localStorage.getItem('id') || userIdFromToken(),
    role: localStorage.getItem('role') || 'operator',
    name: localStorage.getItem('name') || 'Гость',
  }

  useEffect(() => { localStorage.setItem('pc-page', page) }, [page])

  const showToast = useCallback((msg, kind = 'info') => {
    setToast({ msg, kind, id: Math.random() })
  }, [])

  // initial load
  useEffect(() => {
    if (!loggedIn) return
        ;(async () => {
      try {
        const [sec, pl] = await Promise.all([api.getSectors(), api.getPlants()])
        setSectors(sec || [])
        setPlants(pl || [])
        if (!selectedSectorId && sec?.length) setSelectedSectorId(sec[0].id)
        // load telemetry for each sector (last 48 points each)
        const telMap = {}
        await Promise.all((sec || []).map(async s => {
          try {
            const t = await api.getTelemetry(s.id, 48)
            telMap[s.id] = (t || []).slice().reverse()
          } catch {}
        }))
        setTelemetry(telMap)
        try {
          const n = await api.getNotifications(false)
          setNotifs(n || [])
        } catch {}
      } catch (err) {
        showToast('Ошибка загрузки данных: ' + err.message, 'danger')
      }
    })()
  }, [loggedIn])

  // websocket
  useEffect(() => {
    if (!loggedIn) return
    const disconnect = connectWS(msg => {
      setWsEvents(prev => [msg, ...prev].slice(0, 20))
      if (msg.event === 'sector:update' && msg.data) {
        setSectors(prev => prev.map(s => s.id === msg.data.id ? msg.data : s))
        // push to telemetry buffer
        setTelemetry(prev => {
          const arr = prev[msg.data.id] ? [...prev[msg.data.id]] : []
          arr.push({
            id: 't-' + Date.now(),
            sector_id: msg.data.id,
            soil_moisture: msg.data.soil_moisture,
            temperature: msg.data.temperature,
            health_index: msg.data.health_index,
            recorded_at: new Date().toISOString(),
          })
          if (arr.length > 72) arr.shift()
          return { ...prev, [msg.data.id]: arr }
        })
      }
      if (msg.event === 'sector:watered' && msg.data?.sector) {
        setSectors(prev => prev.map(s => s.id === msg.data.sector.id ? msg.data.sector : s))
      }
      if (msg.event === 'notification' && msg.data) {
        setNotifs(prev => [msg.data, ...prev].slice(0, 50))
        if (!notifsOpen) showToast(msg.data.message, 'warn')
      }
    })
    setWsConnected(true)
    return () => {
      setWsConnected(false)
      disconnect()
    }
  }, [loggedIn, showToast, notifsOpen])

  function handleLogout() {
    localStorage.removeItem('token')
    localStorage.removeItem('role')
    localStorage.removeItem('name')
    localStorage.removeItem('id')
    setLoggedIn(false)
  }

  async function handleMarkRead(id) {
    try {
      await api.markNotificationRead(id)
      setNotifs(prev => prev.map(n => n.id === id ? { ...n, is_read: true } : n))
    } catch (err) {
      showToast(err.message, 'danger')
    }
  }

  async function handleMarkAllRead() {
    const unread = notifs.filter(n => !n.is_read)
    await Promise.all(unread.map(n => api.markNotificationRead(n.id).catch(() => {})))
    setNotifs(prev => prev.map(n => ({ ...n, is_read: true })))
  }

  if (!loggedIn) return <Login onLogin={() => setLoggedIn(true)} />

  const notifCount = notifs.filter(n => !n.is_read).length

  return (
      <>
        <Shell
            user={user}
            page={page}
            setPage={setPage}
            onLogout={handleLogout}
            onOpenNotifs={() => setNotifsOpen(true)}
            notifCount={notifCount}
            sectorsCount={sectors.length}
            plantsCount={plants.length}
            wsConnected={wsConnected}>
          {page === 'dashboard' && (
              <Dashboard
                  sectors={sectors}
                  telemetry={telemetry}
                  plants={plants}
                  onSelectSector={id => { setSelectedSectorId(id); setPage('sectors') }}
                  onOpenNotifs={() => setNotifsOpen(true)} />
          )}
          {page === 'sectors' && (
              <Sectors
                  sectors={sectors}
                  setSectors={setSectors}
                  telemetry={telemetry}
                  user={user}
                  selectedId={selectedSectorId}
                  setSelectedId={setSelectedSectorId}
                  onToast={showToast} />
          )}
          {page === 'watering' && (
              <Watering wsEvents={wsEvents} />
          )}
          {page === 'plants' && (
              <Plants sectors={sectors} onToast={showToast} />
          )}
          {page === 'reports' && (
              <Reports sectors={sectors} />
          )}
          {page === 'operators' && user.role === 'agronomist' && (
              <Operators sectors={sectors} onToast={showToast} />
          )}
        </Shell>

        <NotifDrawer
            open={notifsOpen}
            onClose={() => setNotifsOpen(false)}
            notifs={notifs}
            onMarkRead={handleMarkRead}
            onMarkAllRead={handleMarkAllRead} />

        <Toast toast={toast} onDismiss={() => setToast(null)} />
      </>
  )
}