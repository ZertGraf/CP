import { useMemo, useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import { classNames, shortOperatorId, badgeMeta, BADGE_META } from '../lib/format'

// agronomist control to manually award points / badges to one operator (chapter 2.4)
function AwardControl({ userId, onAwarded, onToast }) {
    const [points, setPoints] = useState(10)
    const [sel, setSel] = useState([])
    const [busy, setBusy] = useState(false)

    const toggle = code => setSel(s => s.includes(code) ? s.filter(x => x !== code) : [...s, code])

    async function apply() {
        const p = Number(points) || 0
        if (p === 0 && sel.length === 0) {
            onToast?.('Укажите баллы или выберите бейдж', 'danger')
            return
        }
        setBusy(true)
        try {
            await api.awardScore(userId, { points: p, add_badges: sel })
            onToast?.(`Начислено: ${p > 0 ? '+' : ''}${p}${sel.length ? ` · ${sel.length} бейдж(а)` : ''}`, 'success')
            setSel([])
            setPoints(10)
            onAwarded?.()
        } catch (err) {
            onToast?.(err.message || 'Ошибка начисления', 'danger')
        } finally {
            setBusy(false)
        }
    }

    return (
        <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px dashed var(--line)' }}>
            <div className="field-label" style={{ marginBottom: 6 }}>Поощрение наставника</div>
            <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
                <input className="input" type="number" value={points}
                       onChange={e => setPoints(e.target.value)}
                       style={{ width: 76 }} title="Баллы (можно отрицательные)" />
                {Object.entries(BADGE_META).map(([code, m]) => (
                    <button key={code} type="button"
                            className={classNames('btn btn-sm', sel.includes(code) && 'btn-primary')}
                            title={m.hint} onClick={() => toggle(code)}>
                        {m.icon} {m.ru}
                    </button>
                ))}
                <button className="btn btn-sm btn-primary" disabled={busy} onClick={apply}>
                    {busy ? '...' : 'Начислить'}
                </button>
            </div>
        </div>
    )
}

export default function Operators({ sectors, onToast }) {
    const [board, setBoard] = useState([])
    const [loading, setLoading] = useState(true)

    const loadBoard = useCallback(() => {
        return api.getLeaderboard()
            .then(rows => setBoard(rows || []))
            .catch(() => {})
            .finally(() => setLoading(false))
    }, [])

    useEffect(() => { loadBoard() }, [loadBoard])

    const groups = useMemo(() => {
        const byOp = new Map()
        for (const s of sectors) {
            const k = s.operator_id || '__unassigned'
            if (!byOp.has(k)) byOp.set(k, [])
            byOp.get(k).push(s)
        }
        return [...byOp.entries()]
    }, [sectors])

    const maxScore = Math.max(1, ...board.map(b => b.total_score || 0))

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Операторы</h1>
                    <p className="page-subtitle">
                        Таблица лидеров, ручное поощрение и группировка секторов по оператору.
                    </p>
                </div>
            </div>

            {/* leaderboard (chapter 2.4.4) */}
            <div className="card" style={{ marginBottom: 16 }}>
                <div className="section-head" style={{ margin: 0 }}>
                    <h2 className="section-title">Таблица лидеров</h2>
                    <span className="section-meta">обучающая аналитика · сессия</span>
                </div>
                {loading && <div className="empty" style={{ marginTop: 12 }}>Загрузка...</div>}
                {!loading && board.length === 0 && (
                    <div className="empty" style={{ marginTop: 12 }}>
                        Пока нет накопленных баллов — движок начислит их по мере работы операторов.
                    </div>
                )}
                {board.length > 0 && (
                    <div style={{ overflow: 'auto', marginTop: 12 }}>
                        <table className="table">
                            <thead>
                            <tr>
                                <th style={{ width: 40 }}>#</th>
                                <th>Оператор</th>
                                <th style={{ width: 220 }}>Баллы</th>
                                <th style={{ textAlign: 'right' }}>Ср. здоровье</th>
                                <th style={{ textAlign: 'right' }}>Эффект. воды</th>
                                <th>Бейджи</th>
                            </tr>
                            </thead>
                            <tbody>
                            {board.map((b, i) => (
                                <tr key={b.user_id}>
                                    <td className="mono" style={{ fontWeight: 700, color: i === 0 ? 'var(--brand)' : 'var(--ink-3)' }}>
                                        {i === 0 ? '🏆' : i + 1}
                                    </td>
                                    <td style={{ fontWeight: 600 }}>{b.name || `Оператор #${shortOperatorId(b.user_id)}`}</td>
                                    <td>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                            <div className="progress" style={{ flex: 1 }}>
                                                <span style={{ width: `${Math.max(2, (b.total_score / maxScore) * 100)}%`, background: 'var(--brand)' }} />
                                            </div>
                                            <span className="mono tnum" style={{ fontWeight: 700, minWidth: 48, textAlign: 'right' }}>
                                                {Math.round(b.total_score)}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="num" style={{ textAlign: 'right' }}>{Math.round((b.avg_health || 0) * 100)}</td>
                                    <td className="num" style={{ textAlign: 'right' }}>{Math.round((b.water_efficiency || 0) * 100)}%</td>
                                    <td>
                                        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                                            {(b.badges || []).length === 0 && <span style={{ color: 'var(--ink-4)' }}>—</span>}
                                            {(b.badges || []).map(code => {
                                                const m = badgeMeta(code)
                                                return <span key={code} className="chip" title={m.hint}>{m.icon} {m.ru}</span>
                                            })}
                                        </div>
                                    </td>
                                </tr>
                            ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            <div className="sectors-grid">
                {groups.map(([opId, list]) => {
                    const isEmpty = opId === '__unassigned'
                    const avgHealth = list.reduce((a, s) => a + (s.health_index || 0), 0) / list.length * 100
                    const entry = board.find(b => b.user_id === opId)
                    return (
                        <div key={opId} className="card">
                            <div style={{ display: 'flex', gap: 14, alignItems: 'center', marginBottom: 14 }}>
                                <div className="avatar" style={{ width: 48, height: 48, fontSize: 16, background: isEmpty ? 'var(--ink-4)' : 'var(--brand)' }}>
                                    {isEmpty ? '—' : shortOperatorId(opId)}
                                </div>
                                <div>
                                    <div style={{ fontWeight: 600 }}>
                                        {isEmpty ? 'Без оператора' : (entry?.name || `Оператор #${shortOperatorId(opId)}`)}
                                    </div>
                                    <div style={{ fontSize: 12, color: 'var(--ink-3)', fontFamily: 'var(--mono)' }}>
                                        {isEmpty ? 'требуется назначение' : opId}
                                    </div>
                                </div>
                            </div>
                            <div className="metric-row" style={{ borderTop: '1px dashed var(--line)' }}>
                                <div className="metric"><div className="metric-label">секторов</div><div className="metric-val">{list.length}</div></div>
                                <div className="metric"><div className="metric-label">площадь</div><div className="metric-val">{Math.round(list.reduce((a, s) => a + (s.area_sqm || 0), 0))}</div></div>
                                <div className="metric"><div className="metric-label">ср. здор.</div><div className="metric-val">{Math.round(avgHealth)}</div></div>
                                <div className="metric"><div className="metric-label">баллы</div><div className="metric-val">{entry ? Math.round(entry.total_score) : '—'}</div></div>
                            </div>
                            <div style={{ marginTop: 12, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                                {list.map(s => <span key={s.id} className="chip">{s.name}</span>)}
                            </div>

                            {/* mentor reinforcement: manual points & badges */}
                            {!isEmpty && (
                                <AwardControl userId={opId} onAwarded={loadBoard} onToast={onToast} />
                            )}
                        </div>
                    )
                })}
            </div>
        </div>
    )
}
