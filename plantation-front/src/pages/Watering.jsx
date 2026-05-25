import { useState, useEffect } from 'react'
import { api } from '../api'

export default function Watering({ wsEvents }) {
    const [sectors, setSectors] = useState([])
    const [selected, setSelected] = useState(null)
    const [volume, setVolume] = useState(10)
    const [stats, setStats] = useState(null)
    const [pouring, setPouring] = useState(false)
    const [error, setError] = useState('')

    const role = localStorage.getItem('role')

    useEffect(() => {
        const loader = role === 'operator' ? api.getMySectors : api.getSectors
        loader().then(data => setSectors(data || [])).catch(() => {})
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

    const waterRemaining = selected ? (selected.daily_water_limit || 500) - (selected.water_consumed || 0) : 0

    return (
        <div className="animate-fade-in">
            <h1>💧 Пульт полива</h1>
            {error && <p className="error">{error}</p>}

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
                        <div className={`health-display ${pouring ? 'watering' : ''}`}>
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

                        <div className="math-model-stats" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '24px', padding: '16px', background: '#f8fafc', borderRadius: '12px', fontSize: '13px' }}>
                            <div>
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Коэф. водного стресса (Ks)</span>
                                <strong>{(selected.ks_water || 1).toFixed(2)}</strong>
                            </div>
                            <div>
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Аэрационный стресс (Ks,aer)</span>
                                <strong>{(selected.ks_aeration || 1).toFixed(2)}</strong>
                            </div>
                            <div>
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Дефицит влаги (Dr)</span>
                                <strong>{(selected.deficit_dr || 0).toFixed(1)} мм</strong>
                            </div>
                            <div>
                                <span style={{ color: 'var(--ink-3)', display: 'block', marginBottom: '4px' }}>Сумма темп. (GDD) / Фаза</span>
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
                                disabled={pouring || waterRemaining <= 0}
                            >
                                {pouring ? '💦 Идет полив...' : waterRemaining <= 0 ? '🚫 Лимит исчерпан' : '🚿 Полить сектор'}
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