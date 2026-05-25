import { ICONS } from '../components/icons'
import { classNames, fmtTimeAgo, notifKindLabel } from '../lib/format'

export default function NotifDrawer({ open, onClose, notifs, onMarkRead, onMarkAllRead }) {
    if (!open) return null
    const unread = notifs.filter(n => !n.is_read).length

    return (
        <>
            <div className="drawer-backdrop" onClick={onClose} />
            <div className="drawer">
                <div className="drawer-head">
                    <div>
                        <div style={{ fontSize: 11, letterSpacing: '0.08em', textTransform: 'uppercase', color: 'var(--ink-3)', fontWeight: 500 }}>
                            Уведомления
                        </div>
                        <h2 style={{ margin: '4px 0 0', fontSize: 18, letterSpacing: '-0.01em' }}>
                            Центр событий {unread > 0 && <span style={{ fontSize: 12, color: 'var(--ink-3)', fontWeight: 400 }}>· {unread} новых</span>}
                        </h2>
                    </div>
                    <div style={{ display: 'flex', gap: 6 }}>
                        {unread > 0 && <button className="btn btn-sm" onClick={onMarkAllRead}>Прочитать все</button>}
                        <button className="icon-btn" onClick={onClose}>{ICONS.x}</button>
                    </div>
                </div>
                <div className="drawer-body">
                    {notifs.length === 0 && <div className="empty">Нет уведомлений.</div>}
                    {notifs.map(n => (
                        <div key={n.id} className={classNames('notif', !n.is_read && 'unread')}
                             onClick={() => !n.is_read && onMarkRead(n.id)}>
                            <div className="notif-head">
                                <div className="notif-tag">{notifKindLabel(n.kind)}</div>
                                <div className="notif-time">{fmtTimeAgo(n.created_at)}</div>
                            </div>
                            <div className="notif-body">{n.message}</div>
                        </div>
                    ))}
                </div>
            </div>
        </>
    )
}