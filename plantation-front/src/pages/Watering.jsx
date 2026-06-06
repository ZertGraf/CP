import { useState, useEffect } from 'react'
import { api } from '../api'
import { badgeMeta } from '../lib/format'

// tree tint by health index H (green → yellow → brown), chapter 2.7.3
function treeColor(h) {
    if (h >= 0.8) return '#16a34a'
    if (h >= 0.6) return '#65a30d'
    if (h >= 0.4) return '#ca8a04'
    if (h >= 0.2) return '#b45309'
    return '#78350f'
}

function Plantation({ health }) {
    // schematic plantation: a grid of trees whose colour follows H
    const trees = Array.from({ length: 24 })
    return (
        <div style={{
            display: 'grid', gridTemplateColumns: 'repeat(8, 1fr)', gap: 6,
            padding: 16, background: 'linear-gradient(180deg,#f0fdf4,#ecfccb)', borderRadius: 12,
        }}>
            {trees.map((_, i) => {
                // slight per-tree variation so the field looks alive
                const h = Math.max(0, Math.min(1, health + (Math.sin(i * 12.9898) * 0.06)))
                return (
                    <div key={i} title={`H ≈ ${(h).toFixed(2)}`} style={{
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        aspectRatio: '1', borderRadius: 8,
                        background: `${treeColor(h)}22`,
                    }}>
                        <span style={{ fontSize: 20, filter: h < 0.4 ? 'grayscale(0.4)' : 'none', color: treeColor(h) }}>
                            {h >= 0.6 ? '🌳' : h >= 0.3 ? '🪴' : '🥀'}
                        </span>
                    </div>
                )
            })}
        </div>
    )
}

export default function Watering({ wsEvents = [] }) {
    const [sectors, setSectors] = useState([])
    const [selected, setSelected] = useState(null)
    const [volume, setVolume] = useState(10)
    const [stats, setStats] = useState(null)
    const [pouring, setPouring] = useState(false)
    const [error, setError] = useState('')
    const [score, setScore] = useState(null)

    const role = localStorage.getItem('role')

    useEffect(() => {
        const loader = role === 'operator' ? api.getMySectors : api.getSectors
        loader().then(data => setSectors(data || [])).catch(() => {})
        api.getMyScore().then(setScore).catch(() => {})
    }, [role])

    useEffect(() => {
        if (selected) api.getWaterStats(selected.id).then(setStats).catch(() => {})
    }, [selected])

    // live update selected sector
    useEffect(() => {
        if (!wsEvents.length || !selected) return
        const last = wsEvents[0]
        if (last.event === 'sector:update' && last.data?.id === selected.id) {
            setSelected(last.data)
        }
    }, [wsEvents, selected])

    async function handleWater() {
        if (!selected) return
        setPouring(true)
        setError('')
        try {
            const res = await api.water(selected.id, volume)
            setSelected(res.sector)
            setStats(await api.getWaterStats(selected.id))
        } catch (err) {
            setError(err.message)
        }
        setTimeout(() => setPouring(false), 1500)
    }

    async function handleTreat() {
        if (!selected) return
        setError('')
        try {
            const res = await api.treat(selected.id)
            setSelected(res.sector)
        } catch (err) {
            setError(err.message)
        }
    }

    const waterRemaining = selected ? (selected.daily_water_limit || 500) - (selected.water_consumed || 0) : 0
    const equipmentLocked = selected ? (selected.equipment_locked_ticks || 0) > 0 : false
    const pestActive = selected ? !!selected.pest_active : false

    return (
        <div className="animate-fade-in">
            <h1>💧 Пульт полива</h1>
            {error && <p className="error">{error}</p>}

            {/* operator score + badges (chapter 2.4.1–2.4.2) */}
            {score && (
                <div className="card" style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16, flexWrap: 'wrap' }}>
                    <div>
                        <span style={{ color: 'var(--ink-3)', fontSize: 12, display: 'block' }}>Ваш счёт</span>
                        <strong style={{ fontSize: 22 }}>{Math.round(score.total_score || 0)}</strong>
                    </div>
                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                        {(score.badges || []).length === 0 && (
                            <span style={{ color: 'var(--ink-4)', fontSize: 13 }}>Бейджи пока не получены — поддерживайте здоровье и не переливайте.</span>
                        )}
                        {(score.badges || []).map(code => {
                            const m = badgeMeta(code)
                            return <span key={code} className="chip" title={m.hint}>{m.icon} {m.ru}</span>
                        })}
                    </div>
                </div>
            )}

            {role === 'operator' && sectors.length === 0 && (
                <div className="card" style={{ textAlign: 'center', padding: 40 }}>
                    <p style={{ fontSize: 18 }}>Вам пока не назначен ни один сектор.</p>
                    <p className="stats">Обратитесь к агроному для назначения.</p>
                </div>
            )}

            <div className="card" style={{ maxWidth: 600, margin: '0 auto' }}>
                <select
                    style={{ width: '100%', marginBottom: 24, fontSize: 16 }}
                    value={selected?.id || ''}
                    onChange={e => {
                        const s = sectors.find(s => s.id === e.target.value)
                        setSelected(s || null)
                        setStats(null)
                        setError('')
                    }}>
                    <option value="">Выберите сектор для полива...</option>
                    {sectors.map(s => (
                        <option key={s.id} value={s.id}>📍 {s.name} (Влага: {s.soil_moisture?.toFixed(1)}%)</option>
                    ))}
                </select>

                {selected && (
                    <div className="animate-fade-in">
                        <Plantation health={selected.health_index || 0} />

                        <div className={`health-display ${pouring ? 'watering' : ''}`} style={{ marginTop: 16 }}>
                            {pouring && (
                                <>
                                    <div className="watering-can-anim">
                                        <svg viewBox="0 0 24 24" width="64" height="64" fill="currentColor">
                                            <path d="M21 7l-2.6-1.5a1.5 1.5 0 0 0-2 .6l-1 1.7h-3.9a1.5 1.5 0 0 0-1.4 1H7v-.3a1.5 1.5 0 0 0-1.5-1.5h-1a1.5 1.5 0 0 0-1.5 1.5v6.5A1.5 1.5 0 0 0 4.5 17h1a1.5 1.5 0 0 0 1.5-1.5V15h8v.5a1.5 1.5 0 0 0 1.5 1.5h3.4a1.5 1.5 0 0 0 1.3-2.2l1.6-2.7a1.5 1.5 0 0 0-.6-2l-1-.6z" />
                                        </svg>
                                    </div>
                                    <div className="rain-container">
                                        <div className="drop"></div><div className="drop"></div>
                                        <div className="drop"></div><div className="drop"></div>
                                        <div className="drop"></div>
                                    </div>
                                </>
                            )}
                            <span className="plant-emoji">
                                {selected.health_index >= 0.8 ? '🌳' : selected.health_index >= 0.5 ? '🪴' : '🥀'}
                            </span>
                            <div className="health-text">
                                Здоровье: {Math.round((selected.health_index || 0) * 100)}/100
                            </div>
                        </div>

                        {/* math-model coefficients with formula tooltips (chapter 2.7.3) */}
                        <div className="math-model-stats" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '24px', padding: '16px', background: '#f8fafc', borderRadius: '12px', fontSize: '13px' }}>
                            <div title="Коэффициент водного стресса по FAO-56: Ks = (TAW − Dr) / ((1 − p)·TAW). 1 — стресса нет, 0 — критическая засуха.">
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Коэф. водного стресса (Ks) ⓘ</span>
                                <strong>{(selected.ks_water ?? 1).toFixed(2)}</strong>
                            </div>
                            <div title="Аэрационный стресс при переувлажнении: Ks,aer = (θsat − θ) / θaer. 1 — корни дышат, 0 — удушье от воды.">
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Аэрационный стресс (Ks,aer) ⓘ</span>
                                <strong>{(selected.ks_aeration ?? 1).toFixed(2)}</strong>
                            </div>
                            <div title="Индекс водного стресса растения CWSI = ((Tc−Ta) − LL) / (UL − LL). 0 — растение обводнено, 1 — критический стресс.">
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>CWSI ⓘ</span>
                                <strong>{(selected.cwsi ?? 0).toFixed(2)}</strong>
                            </div>
                            <div title="Дефицит влаги корневой зоны Dr по уравнению водного баланса: Dr(t+1) = Dr(t) − Peff − I + ETc + DP.">
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Дефицит влаги (Dr) ⓘ</span>
                                <strong>{(selected.deficit_dr || 0).toFixed(1)} мм</strong>
                            </div>
                            <div title="Сумма эффективных температур GDD = Σ max(0, (Tmax+Tmin)/2 − Tbase). Определяет фенофазу по шкале BBCH." style={{ gridColumn: '1 / -1' }}>
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Сумма темп. (GDD) / Фаза BBCH ⓘ</span>
                                <strong>{(selected.gdd_cumulative || 0).toFixed(0)} / {selected.phenophase || '00'}</strong>
                            </div>
                        </div>

                        <div style={{ marginBottom: 24 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                <strong>Влажность почвы</strong>
                                <span className={`badge ${selected.status}`}>
                  {selected.status === 'normal' ? 'Норма' : selected.status === 'drought' ? 'Засуха' : 'Перелив'}
                </span>
                            </div>
                            <div className="moisture-bar" style={{ height: 20 }}>
                                <div className="moisture-bar-fill" style={{ width: `${Math.min(selected.soil_moisture || 0, 100)}%` }} />
                            </div>
                            <span className="stats">{selected.soil_moisture?.toFixed(1)}% / 100%</span>
                        </div>

                        <div className="water-limit-bar" style={{ marginBottom: 24 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 4 }}>
                                <span>Суточный лимит воды</span>
                                <span style={{ fontWeight: 600 }}>{(selected.water_consumed || 0).toFixed(0)} / {(selected.daily_water_limit || 500).toFixed(0)} л</span>
                            </div>
                            <div className="moisture-bar" style={{ height: 10 }}>
                                <div style={{
                                    height: '100%',
                                    borderRadius: 12,
                                    transition: 'width 0.5s',
                                    background: waterRemaining < 50 ? '#ef4444' : '#f59e0b',
                                    width: `${Math.min(((selected.water_consumed || 0) / (selected.daily_water_limit || 500)) * 100, 100)}%`
                                }} />
                            </div>
                            <span className="stats">Осталось: {Math.max(waterRemaining, 0).toFixed(0)} л</span>
                        </div>

                        {pestActive && (
                            <div className="card" style={{ background: '#fffbeb', border: '1px solid #fde68a', marginBottom: 16, padding: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
                                <span>🐛 Нашествие вредителей — здоровье падает, пока сектор не обработан.</span>
                                <button className="primary" style={{ whiteSpace: 'nowrap' }} onClick={handleTreat}>Обработать</button>
                            </div>
                        )}

                        {equipmentLocked && (
                            <div className="card" style={{ background: '#fef2f2', border: '1px solid #fecaca', marginBottom: 16, padding: 12, textAlign: 'center' }}>
                                🔧 Поломка оборудования: полив недоступен ещё {selected.equipment_locked_ticks} тик(ов).
                            </div>
                        )}

                        <div style={{ background: '#f3f4f6', padding: 20, borderRadius: 12 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                                <strong>Объем воды:</strong>
                                <strong style={{ color: 'var(--primary-dark)', fontSize: 18 }}>{volume} л</strong>
                            </div>
                            <input
                                type="range" min="1" max={Math.min(50, Math.max(1, Math.floor(waterRemaining)))} step="1" value={volume}
                                onChange={e => setVolume(Number(e.target.value))}
                                style={{ width: '100%', marginBottom: 20, cursor: 'pointer' }}
                            />
                            <button
                                className="primary"
                                style={{ width: '100%', padding: '16px', fontSize: 18 }}
                                onClick={handleWater}
                                disabled={pouring || waterRemaining <= 0 || equipmentLocked}
                            >
                                {equipmentLocked ? '🔧 Оборудование неисправно' : pouring ? '💦 Идет полив...' : waterRemaining <= 0 ? '🚫 Лимит исчерпан' : '🚿 Полить сектор'}
                            </button>
                        </div>

                        {stats && (
                            <div className="stats" style={{ marginTop: 24, textAlign: 'center' }}>
                                📊 За всё время: <b>{stats.total_liters?.toFixed(1)} л</b> за {stats.total_events} сеансов
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    )
}
