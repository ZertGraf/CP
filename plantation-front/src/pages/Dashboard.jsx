import { useMemo, useState, useEffect } from 'react'
import { api } from '../api'
import { ICONS } from '../components/icons'
import { Sparkline, LineChart } from '../components/visuals'
import { classNames, fmtTimeAgo, resolveStatus, STATUS_LABEL, badgeMeta, shortOperatorId } from '../lib/format'

export default function Dashboard({ sectors, telemetry, plants, onSelectSector, onOpenNotifs }) {
    const [board, setBoard] = useState([])
    useEffect(() => {
        let alive = true
        api.getLeaderboard().then(rows => { if (alive) setBoard(rows || []) }).catch(() => {})
        return () => { alive = false }
    }, [])
    const maxScore = Math.max(1, ...board.map(b => b.total_score || 0))
    const totalPlants = plants.length
    const totalArea = Math.round(sectors.reduce((a, s) => a + (s.area_sqm || 0), 0))
    const avgMoist = sectors.length ? sectors.reduce((a, s) => a + (s.soil_moisture || 0), 0) / sectors.length : 0
    const avgTemp = sectors.length ? sectors.reduce((a, s) => a + (s.temperature || 0), 0) / sectors.length : 0
    const avgHealth = sectors.length ? sectors.reduce((a, s) => a + (s.health_index || 0), 0) / sectors.length * 100 : 0
    const totalWater = sectors.reduce((a, s) => a + (s.water_consumed || 0), 0)
    const limit = Math.max(1, sectors.reduce((a, s) => a + (s.daily_water_limit || 0), 0))

    const alerts = useMemo(() => {
        return sectors
            .map(s => ({ ...s, status: resolveStatus(s) }))
            .filter(s => ['drought', 'overwatered', 'warn', 'dead', 'critical', 'heat_stress', 'recovering'].includes(s.status))
    }, [sectors])

    const firstId = sectors[0]?.id
    const tel = telemetry[firstId] || []
    const chartSeries = [
        { label: 'Влажность, %',    color: 'var(--data-water)',  values: tel.map(p => ({ t: new Date(p.recorded_at).getTime(), v: p.soil_moisture })) },
        { label: 'Температура, °C', color: 'var(--data-temp)',   values: tel.map(p => ({ t: new Date(p.recorded_at).getTime(), v: p.temperature })) },
        { label: 'Здоровье',        color: 'var(--data-health)', values: tel.map(p => ({ t: new Date(p.recorded_at).getTime(), v: (p.health_index || 0) * 100 })) },
    ]

    const sparkMoist = tel.map(p => p.soil_moisture)
    const sparkTemp = tel.map(p => p.temperature)
    const sparkHealth = tel.map(p => (p.health_index || 0) * 100)

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Обзор плантации.</h1>
                    <p className="page-subtitle">
                        {alerts.length > 0
                            ? <>Требуют внимания: <strong style={{ color: 'var(--danger)' }}>{alerts.length} сектор(ов)</strong>. Проверьте уведомления.</>
                            : 'Все секторы в пределах нормы. Модель работает штатно.'}
                    </p>
                </div>
                <div className="page-head-actions">
                    <button className="btn" onClick={onOpenNotifs}>{ICONS.bell} Уведомления</button>
                </div>
            </div>

            <div className="kpi-row">
                <div className="kpi">
                    <div className="kpi-label">{ICONS.droplet} средняя влажность</div>
                    <div className="kpi-value tnum">{avgMoist.toFixed(1)}<span className="kpi-unit">%</span></div>
                    <div className="kpi-spark"><Sparkline data={sparkMoist} color="var(--data-water)" /></div>
                </div>
                <div className="kpi">
                    <div className="kpi-label">{ICONS.thermo} температура</div>
                    <div className="kpi-value tnum">{avgTemp.toFixed(1)}<span className="kpi-unit">°C</span></div>
                    <div className="kpi-spark"><Sparkline data={sparkTemp} color="var(--data-temp)" /></div>
                </div>
                <div className="kpi">
                    <div className="kpi-label">{ICONS.heart} индекс здоровья</div>
                    <div className="kpi-value tnum">{Math.round(avgHealth)}<span className="kpi-unit">/100</span></div>
                    <div className="kpi-spark"><Sparkline data={sparkHealth} color="var(--data-health)" /></div>
                </div>
                <div className="kpi">
                    <div className="kpi-label">{ICONS.droplet} воды за сутки</div>
                    <div className="kpi-value tnum">{Math.round(totalWater)}<span className="kpi-unit">/{Math.round(limit)} л</span></div>
                    <div className="progress water" style={{ marginTop: 8 }}>
                        <span style={{ width: `${(totalWater / limit) * 100}%` }} />
                    </div>
                </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 16 }}>
                <div className="card">
                    <div className="section-head" style={{ margin: 0 }}>
                        <h2 className="section-title">Карта плантации</h2>
                        <span className="section-meta">{sectors.length} секторов · {totalPlants} растений · {totalArea} м²</span>
                    </div>
                    <div className="map-svg-wrap" style={{ marginTop: 12 }}>
                        {sectors.map(s => {
                            const st = resolveStatus(s)
                            return (
                                <div key={s.id}
                                     className={classNames('map-cell', st)}
                                     onClick={() => onSelectSector(s.id)}>
                                    <p className="map-cell-name">{s.name}</p>
                                    <div className="map-cell-num">{Math.round(s.area_sqm || 0)} м²</div>
                                    <div className="map-cell-moist tnum">
                                        {(s.soil_moisture || 0).toFixed(0)}%
                                        <span style={{ fontSize: 10, color: 'var(--ink-3)', marginLeft: 4 }}>влажн.</span>
                                    </div>
                                </div>
                            )
                        })}
                    </div>
                    <div className="map-legend">
                        <span><i style={{ background: 'color-mix(in srgb, var(--good) 55%, transparent)' }} />в норме</span>
                        <span><i style={{ background: 'color-mix(in srgb, var(--warn) 55%, transparent)' }} />внимание</span>
                        <span><i style={{ background: 'color-mix(in srgb, var(--danger) 55%, transparent)' }} />засуха</span>
                        <span><i style={{ background: 'color-mix(in srgb, var(--info) 55%, transparent)' }} />переувлажнено</span>
                    </div>
                </div>

                <div className="card">
                    <div className="section-head" style={{ margin: 0 }}>
                        <h2 className="section-title">Требуют внимания</h2>
                        <span className="section-meta">{alerts.length}</span>
                    </div>
                    <div className="stack" style={{ marginTop: 12 }}>
                        {alerts.length === 0 && <div className="empty">Ни один сектор не требует вмешательства.</div>}
                        {alerts.map(s => {
                            const sl = STATUS_LABEL[s.status]
                            return (
                                <div key={s.id} className="card card-tight" style={{ cursor: 'pointer' }}
                                     onClick={() => onSelectSector(s.id)}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', gap: 10 }}>
                                        <div>
                                            <div style={{ fontWeight: 600, fontSize: 13 }}>{s.name}</div>
                                            <div style={{ fontSize: 11, color: 'var(--ink-3)', fontFamily: 'var(--mono)', marginTop: 2 }}>
                                                влажн. {(s.soil_moisture || 0).toFixed(0)}% · {fmtTimeAgo(s.last_watered_at)}
                                            </div>
                                        </div>
                                        <span className={`chip ${sl.chip}`}><span className="dot" />{sl.ru}</span>
                                    </div>
                                </div>
                            )
                        })}
                    </div>
                </div>
            </div>

            {/* leaderboard (chapter 2.4.4 / agronomist dashboard wireframe 2.7.2) */}
            <div className="section-head">
                <h2 className="section-title">Таблица лидеров</h2>
                <span className="section-meta">обучающая аналитика</span>
            </div>
            <div className="card">
                {board.length === 0 ? (
                    <div className="empty">Баллы ещё не накоплены — движок начисляет их по мере работы операторов.</div>
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                        {board.slice(0, 5).map((b, i) => (
                            <div key={b.user_id} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                                <span className="mono" style={{ width: 24, fontWeight: 700, color: i === 0 ? 'var(--brand)' : 'var(--ink-3)' }}>
                                    {i === 0 ? '🏆' : i + 1}
                                </span>
                                <span style={{ width: 200, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                    {b.name || `Оператор #${shortOperatorId(b.user_id)}`}
                                </span>
                                <div className="progress" style={{ flex: 1 }}>
                                    <span style={{ width: `${Math.max(2, (b.total_score / maxScore) * 100)}%`, background: 'var(--brand)' }} />
                                </div>
                                <span className="mono tnum" style={{ width: 48, textAlign: 'right', fontWeight: 700 }}>{Math.round(b.total_score)}</span>
                                <span style={{ display: 'flex', gap: 4 }}>
                                    {(b.badges || []).map(code => {
                                        const m = badgeMeta(code)
                                        return <span key={code} title={`${m.ru} — ${m.hint}`}>{m.icon}</span>
                                    })}
                                </span>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            <div className="section-head">
                <h2 className="section-title">Активность за последние 24 часа</h2>
                <span className="section-meta">{sectors[0]?.name || ''}</span>
            </div>
            <div className="card">
                <LineChart series={chartSeries} />
            </div>
        </div>
    )
}