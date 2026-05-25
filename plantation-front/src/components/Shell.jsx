import { ICONS } from './icons'
import { classNames } from '../lib/format'

export default function Shell({
                                  user, page, setPage, onLogout,
                                  notifCount, onOpenNotifs, sectorsCount, plantsCount, wsConnected, children,
                              }) {
    const nav = [
        { id: 'dashboard', label: 'Обзор',    icon: ICONS.grid,   count: null },
        { id: 'sectors',   label: 'Секторы',  icon: ICONS.leaf,   count: sectorsCount ?? null },
        { id: 'plants',    label: 'Растения', icon: ICONS.plants, count: plantsCount ?? null },
        { id: 'reports',   label: 'Отчёты',   icon: ICONS.chart,  count: null },
    ]
    const adminNav = [
        { id: 'operators', label: 'Операторы', icon: ICONS.user, count: null },
    ]
    const titles = {
        dashboard: 'Обзор плантации',
        sectors: 'Секторы',
        plants: 'Растения',
        reports: 'Отчёты и телеметрия',
        operators: 'Операторы',
    }
    const pageTitle = titles[page] || ''
    const isAgro = user.role === 'agronomist'

    return (
        <div className="app">
            <div className="brand-cell">
                <div className="brand-mark">{ICONS.leaf}</div>
                <div>
                    <div className="brand-name">Plantation Control</div>
                    <div className="brand-sub">ops · v2.4</div>
                </div>
            </div>

            <div className="topbar">
                <div className="crumb">
                    <span>{pageTitle}</span>
                    <span className="crumb-sep">/</span>
                    <strong>Гватемала · Кобан</strong>
                </div>
                <div className="topbar-right">
          <span className="ws-status">
            <span className={classNames('ws-dot', !wsConnected && 'off')} />
              {wsConnected ? 'WS · live' : 'WS · offline'}
          </span>
                    <button className="icon-btn" onClick={onOpenNotifs} title="Уведомления">
                        {ICONS.bell}
                        {notifCount > 0 && <span className="badge-dot" />}
                    </button>
                    <div className="user-chip">
                        <div className="avatar">{(user.name || '?').slice(0, 1)}</div>
                        <div style={{ lineHeight: 1.1 }}>
                            <div style={{ fontSize: 13, fontWeight: 500 }}>{user.name}</div>
                            <div className="role">{isAgro ? 'Агроном' : 'Оператор'}</div>
                        </div>
                        <button className="icon-btn" onClick={onLogout} title="Выход" style={{ width: 28, height: 28 }}>
                            {ICONS.logout}
                        </button>
                    </div>
                </div>
            </div>

            <aside className="sidebar">
                <div className="nav-section">Рабочее пространство</div>
                {nav.map(n => (
                    <div key={n.id}
                         className={classNames('nav-item', page === n.id && 'active')}
                         onClick={() => setPage(n.id)}>
                        {n.icon}
                        <span>{n.label}</span>
                        {n.count != null && <span className="count">{n.count}</span>}
                    </div>
                ))}
                {isAgro && (
                    <>
                        <div className="nav-section">Администрирование</div>
                        {adminNav.map(n => (
                            <div key={n.id}
                                 className={classNames('nav-item', page === n.id && 'active')}
                                 onClick={() => setPage(n.id)}>
                                {n.icon}
                                <span>{n.label}</span>
                                {n.count != null && <span className="count">{n.count}</span>}
                            </div>
                        ))}
                    </>
                )}
                <div className="sidebar-foot">
                    <div>simulation · running</div>
                    <div>tick: 10s · sectors: {sectorsCount ?? '—'}</div>
                    <div style={{ color: wsConnected ? 'var(--good)' : 'var(--ink-4)' }}>
                        ● {wsConnected ? 'healthy' : 'waiting'}
                    </div>
                </div>
            </aside>

            <main className="main">
                <div className="main-inner">{children}</div>
            </main>
        </div>
    )
}