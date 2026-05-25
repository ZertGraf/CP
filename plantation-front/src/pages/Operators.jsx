import { useMemo } from 'react'
import { ICONS } from '../components/icons'
import { shortOperatorId } from '../lib/format'

export default function Operators({ sectors }) {
    const groups = useMemo(() => {
        const byOp = new Map()
        for (const s of sectors) {
            const k = s.operator_id || '__unassigned'
            if (!byOp.has(k)) byOp.set(k, [])
            byOp.get(k).push(s)
        }
        return [...byOp.entries()]
    }, [sectors])

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Операторы</h1>
                    <p className="page-subtitle">
                        Группировка секторов по назначенному оператору. Бэкенд пока не предоставляет реестр пользователей, потому имена отображаются по обрезанному UUID.
                    </p>
                </div>
            </div>

            <div className="sectors-grid">
                {groups.map(([opId, list]) => {
                    const isEmpty = opId === '__unassigned'
                    const avgHealth = list.reduce((a, s) => a + (s.health_index || 0), 0) / list.length * 100
                    const plantsTotal = list.reduce((a, s) => a + (s.area_sqm ? 1 : 0), 0)
                    return (
                        <div key={opId} className="card">
                            <div style={{ display: 'flex', gap: 14, alignItems: 'center', marginBottom: 14 }}>
                                <div className="avatar" style={{ width: 48, height: 48, fontSize: 16, background: isEmpty ? 'var(--ink-4)' : 'var(--brand)' }}>
                                    {isEmpty ? '—' : shortOperatorId(opId)}
                                </div>
                                <div>
                                    <div style={{ fontWeight: 600 }}>
                                        {isEmpty ? 'Без оператора' : `Оператор #${shortOperatorId(opId)}`}
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
                            </div>
                            <div style={{ marginTop: 12, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                                {list.map(s => <span key={s.id} className="chip">{s.name}</span>)}
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}