import { useState, useEffect } from 'react'
import { api } from '../api'
import { ICONS } from '../components/icons'
import { LineChart } from '../components/visuals'
import { resolveStatus, STATUS_LABEL } from '../lib/format'

export default function Reports({ sectors }) {
    const [selId, setSelId] = useState(sectors[0]?.id || '')
    const [report, setReport] = useState(null)
    const [tel, setTel] = useState([])
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        if (!sectors.length) return
        if (!selId || !sectors.find(s => s.id === selId)) setSelId(sectors[0].id)
    }, [sectors])

    useEffect(() => {
        if (!selId) return
        setLoading(true)
        Promise.all([api.getReport(selId), api.getTelemetry(selId, 200)])
            .then(([r, t]) => { setReport(r); setTel(t || []) })
            .finally(() => setLoading(false))
    }, [selId])

    const sec = sectors.find(s => s.id === selId)
    const telSorted = tel.slice().sort((a, b) => new Date(a.recorded_at) - new Date(b.recorded_at))
    const pts = telSorted.map(p => ({ t: new Date(p.recorded_at).getTime() }))

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Отчёты и телеметрия</h1>
                    <p className="page-subtitle">Агрегированные данные по сектору.</p>
                </div>
                <div className="page-head-actions">
                    <select className="select" value={selId} onChange={e => setSelId(e.target.value)} style={{ width: 260 }}>
                        {sectors.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                    </select>
                </div>
            </div>

            {loading && <div className="empty">Загрузка...</div>}

            {report && (
                <>
                    <div className="kpi-row" style={{ gridTemplateColumns: 'repeat(6,1fr)' }}>
                        <div className="kpi"><div className="kpi-label">точек</div><div className="kpi-value tnum">{report.telemetry_points}</div></div>
                        <div className="kpi"><div className="kpi-label">ср. влажн.</div><div className="kpi-value tnum">{(report.avg_soil_moisture || 0).toFixed(1)}<span className="kpi-unit">%</span></div></div>
                        <div className="kpi"><div className="kpi-label">ср. темп.</div><div className="kpi-value tnum">{(report.avg_temperature || 0).toFixed(1)}<span className="kpi-unit">°</span></div></div>
                        <div className="kpi"><div className="kpi-label">мин. здоровье</div><div className="kpi-value tnum">{Math.round((report.min_health_index || 0) * 100)}</div></div>
                        <div className="kpi"><div className="kpi-label">всего воды</div><div className="kpi-value tnum">{Math.round(report.total_water_liters || 0)}<span className="kpi-unit">л</span></div></div>
                        <div className="kpi"><div className="kpi-label">сеансов</div><div className="kpi-value tnum">{report.watering_events}</div></div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
                        <div className="card">
                            <div className="section-head" style={{ margin: 0 }}>
                                <h2 className="section-title">Влажность и здоровье</h2>
                            </div>
                            <div style={{ marginTop: 12 }}>
                                <LineChart height={220} series={[
                                    { label: 'Влажность, %', color: 'var(--data-water)',  values: telSorted.map((p, i) => ({ t: pts[i].t, v: p.soil_moisture })) },
                                    { label: 'Здоровье',     color: 'var(--data-health)', values: telSorted.map((p, i) => ({ t: pts[i].t, v: (p.health_index || 0) * 100 })) },
                                ]} />
                            </div>
                        </div>
                        <div className="card">
                            <div className="section-head" style={{ margin: 0 }}>
                                <h2 className="section-title">Температура</h2>
                            </div>
                            <div style={{ marginTop: 12 }}>
                                <LineChart height={220} unit="°" series={[
                                    { label: 'Температура, °C', color: 'var(--data-temp)', values: telSorted.map((p, i) => ({ t: pts[i].t, v: p.temperature })) },
                                ]} />
                            </div>
                        </div>
                    </div>

                    <div className="card" style={{ padding: 0 }}>
                        <div style={{ maxHeight: 340, overflow: 'auto' }}>
                            <table className="table">
                                <thead>
                                <tr>
                                    <th>Время</th>
                                    <th style={{ textAlign: 'right' }}>Влажность</th>
                                    <th style={{ textAlign: 'right' }}>Температура</th>
                                    <th style={{ textAlign: 'right' }}>Здоровье</th>
                                    <th>Статус</th>
                                </tr>
                                </thead>
                                <tbody>
                                {tel.slice(0, 100).map(p => {
                                    const st = resolveStatus({ soil_moisture: p.soil_moisture })
                                    const sl = STATUS_LABEL[st]
                                    return (
                                        <tr key={p.id}>
                                            <td className="mono">{new Date(p.recorded_at).toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit', day: '2-digit', month: '2-digit' })}</td>
                                            <td className="num" style={{ textAlign: 'right', color: p.soil_moisture < 20 ? 'var(--danger)' : p.soil_moisture > 90 ? 'var(--info)' : 'var(--ink)' }}>
                                                {p.soil_moisture.toFixed(1)}%
                                            </td>
                                            <td className="num" style={{ textAlign: 'right' }}>{p.temperature.toFixed(1)}°</td>
                                            <td className="num" style={{ textAlign: 'right', color: p.health_index < 0.4 ? 'var(--danger)' : p.health_index < 0.6 ? 'var(--warn)' : 'var(--good)' }}>
                                                {Math.round((p.health_index || 0) * 100)}
                                            </td>
                                            <td><span className={`chip ${sl.chip}`}><span className="dot" />{sl.ru}</span></td>
                                        </tr>
                                    )
                                })}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </>
            )}
        </div>
    )
}