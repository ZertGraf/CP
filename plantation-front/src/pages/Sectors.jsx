import { useState, useEffect, useMemo } from 'react'
import { api } from '../api'
import { ICONS } from '../components/icons'
import { HealthRing, LineChart, PlantVisual } from '../components/visuals'
import { classNames, fmtTimeAgo, resolveStatus, STATUS_LABEL, shortOperatorId } from '../lib/format'

export default function Sectors({
                                    sectors, setSectors, telemetry, user, selectedId, setSelectedId, onToast,
                                }) {
    const selected = useMemo(
        () => sectors.find(s => s.id === selectedId) || sectors[0] || null,
        [sectors, selectedId]
    )

    const isAgro = user.role === 'agronomist'
    const [volume, setVolume] = useState(20)
    const [pressed, setPressed] = useState(false)
    const [query, setQuery] = useState('')
    const [assignOpen, setAssignOpen] = useState(false)
    const [assignId, setAssignId] = useState('')
    const [importFile, setImportFile] = useState(null)

    const filtered = sectors.filter(s => s.name.toLowerCase().includes(query.toLowerCase()))
    const canWater = selected && (isAgro || selected.operator_id === localStorage.getItem('user_id'))

    async function handleWater() {
        if (!selected) return
        if ((selected.water_consumed || 0) + volume > (selected.daily_water_limit || 0)) {
            onToast('Суточный лимит воды превышен.', 'danger')
            return
        }
        setPressed(true)
        try {
            const res = await api.water(selected.id, volume)
            setSectors(prev => prev.map(s => s.id === res.sector.id ? res.sector : s))
            onToast(`Сектор полит: +${volume}л`, 'success')
        } catch (err) {
            onToast(err.message || 'Ошибка полива', 'danger')
        } finally {
            setTimeout(() => setPressed(false), 900)
        }
    }

    async function handleUnassign(sectorId) {
        try {
            await api.unassignOperator(sectorId)
            const list = await api.getSectors()
            setSectors(list || [])
            onToast('Оператор снят', 'info')
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    async function handleAssign() {
        if (!selected || !assignId.trim()) return
        try {
            await api.assignOperator(selected.id, assignId.trim())
            const list = await api.getSectors()
            setSectors(list || [])
            setAssignOpen(false)
            setAssignId('')
            onToast('Оператор назначен', 'success')
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    async function handleExport() {
        try {
            const blob = await api.exportSectors()
            const url = URL.createObjectURL(blob)
            const a = document.createElement('a')
            a.href = url; a.download = 'sectors.csv'; a.click()
            URL.revokeObjectURL(url)
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    async function handleImport() {
        if (!importFile) return
        try {
            const r = await api.importSectors(importFile)
            setImportFile(null)
            const list = await api.getSectors()
            setSectors(list || [])
            onToast(`Импортировано: ${r.imported}`, 'success')
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    const tel = selected ? (telemetry[selected.id] || []) : []

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Секторы</h1>
                    <p className="page-subtitle">Мониторинг состояния и управление ирригацией.</p>
                </div>
                <div className="page-head-actions">
                    <div style={{ position: 'relative' }}>
            <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--ink-4)', width: 14, height: 14 }}>
              {ICONS.search}
            </span>
                        <input className="input" placeholder="Поиск сектора..." value={query} onChange={e => setQuery(e.target.value)}
                               style={{ paddingLeft: 32, width: 220 }} />
                    </div>
                    {isAgro && (
                        <>
                            <label className="btn" style={{ cursor: 'pointer' }}>
                                {ICONS.upload} Импорт CSV
                                <input type="file" accept=".csv" hidden
                                       onChange={e => { setImportFile(e.target.files[0]); setTimeout(handleImport, 0) }} />
                            </label>
                            <button className="btn" onClick={handleExport}>{ICONS.download} Экспорт</button>
                        </>
                    )}
                </div>
            </div>

            <div className="sectors-grid">
                {filtered.map(s => {
                    const st = resolveStatus(s)
                    const sl = STATUS_LABEL[st]
                    const opShort = shortOperatorId(s.operator_id)
                    return (
                        <div key={s.id}
                             className={classNames('sector-tile', st, s.id === selected?.id && 'sel')}
                             onClick={() => setSelectedId(s.id)}>
                            <div className="status-bar" />
                            <div className="sector-tile-head">
                                <div>
                                    <h3 className="sector-name">{s.name}</h3>
                                    <div className="sector-meta">
                                        {Math.round(s.area_sqm || 0)} м² · {opShort ? `оп. #${opShort}` : 'без оператора'}
                                    </div>
                                </div>
                                <span className={`chip ${sl.chip}`}><span className="dot" />{sl.ru}</span>
                            </div>
                            <div className="metric-row">
                                <div className="metric"><div className="metric-label">Влажн.</div><div className="metric-val">{(s.soil_moisture || 0).toFixed(0)}%</div></div>
                                <div className="metric"><div className="metric-label">Темп.</div><div className="metric-val">{(s.temperature || 0).toFixed(1)}°</div></div>
                                <div className="metric"><div className="metric-label">Здор.</div><div className="metric-val">{Math.round((s.health_index || 0) * 100)}</div></div>
                            </div>
                            <div className="progress water" style={{ marginTop: 10 }}>
                                <span style={{ width: `${((s.water_consumed || 0) / (s.daily_water_limit || 1)) * 100}%` }} />
                            </div>
                            <div style={{
                                display: 'flex', justifyContent: 'space-between',
                                fontSize: 10, fontFamily: 'var(--mono)',
                                color: 'var(--ink-4)', marginTop: 4,
                                textTransform: 'uppercase', letterSpacing: '0.04em',
                            }}>
                                <span>{Math.round(s.water_consumed || 0)}/{Math.round(s.daily_water_limit || 0)} л</span>
                                <span>{fmtTimeAgo(s.last_watered_at)}</span>
                            </div>
                        </div>
                    )
                })}
            </div>

            {selected && (
                <div className="detail">
                    <div className="watering-panel">
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                            <div>
                                <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--ink-3)', fontWeight: 500 }}>
                                    Выбранный сектор
                                </div>
                                <h2 style={{ margin: '4px 0 0', fontSize: 18, letterSpacing: '-0.01em' }}>{selected.name}</h2>
                            </div>
                            <HealthRing value={(selected.health_index || 0) * 100} size={72} />
                        </div>

                        <div className="plant-vis">
                            <PlantVisual moisture={selected.soil_moisture} health={(selected.health_index || 0) * 100} watering={pressed} />
                        </div>

                        <div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                <span>Влажность почвы</span>
                                <span className="mono tnum"><strong style={{ color: 'var(--ink)' }}>{(selected.soil_moisture || 0).toFixed(1)}</strong>%</span>
                            </div>
                            <div className="progress water"><span style={{ width: `${selected.soil_moisture || 0}%` }} /></div>
                        </div>

                        <div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                <span>Лимит на сутки</span>
                                <span className="mono tnum">
                  <strong style={{ color: 'var(--ink)' }}>{Math.round(selected.water_consumed || 0)}</strong>/{Math.round(selected.daily_water_limit || 0)} л
                </span>
                            </div>
                            <div className={classNames('progress', (selected.water_consumed || 0) / (selected.daily_water_limit || 1) > 0.85 ? 'warn' : 'water')}>
                                <span style={{ width: `${((selected.water_consumed || 0) / (selected.daily_water_limit || 1)) * 100}%` }} />
                            </div>
                        </div>

                        <div className="divider" />

                        <div>
                            <div className="field-label" style={{ marginBottom: 8 }}>Объём полива</div>
                            <div style={{ display: 'flex', gap: 6, marginBottom: 10 }}>
                                {[5, 10, 20, 40].map(v => (
                                    <button key={v} className={classNames('btn btn-sm', volume === v && 'btn-primary')}
                                            onClick={() => setVolume(v)} style={{ flex: 1, justifyContent: 'center' }}>{v}л</button>
                                ))}
                            </div>
                            <input type="range" min="1" max="100" value={volume}
                                   onChange={e => setVolume(+e.target.value)}
                                   style={{ width: '100%', accentColor: 'var(--brand)' }} />
                            <div style={{ textAlign: 'center', fontFamily: 'var(--mono)', fontSize: 24, fontWeight: 600, margin: '6px 0 12px' }}>
                                {volume} л
                            </div>
                            <button className="btn btn-primary btn-lg"
                                    style={{ width: '100%', justifyContent: 'center', transform: pressed ? 'scale(0.98)' : 'none', transition: 'transform 120ms' }}
                                    onClick={handleWater}
                                    disabled={!canWater || pressed}>
                                {ICONS.droplet} Полить сектор
                            </button>
                            {!canWater && (
                                <div style={{ fontSize: 11, color: 'var(--ink-3)', textAlign: 'center', marginTop: 8 }}>
                                    Сектор закреплён за другим оператором
                                </div>
                            )}
                        </div>

                        {isAgro && (
                            <>
                                <div className="divider" />
                                <div className="field-label" style={{ marginBottom: 8 }}>Назначение оператора</div>
                                {selected.operator_id ? (
                                    <div style={{ display: 'flex', gap: 6 }}>
                                        <div className="input" style={{ flex: 1, fontFamily: 'var(--mono)', fontSize: 12 }}>
                                            #{shortOperatorId(selected.operator_id)}
                                        </div>
                                        <button className="btn btn-danger btn-sm" onClick={() => handleUnassign(selected.id)}>Снять</button>
                                    </div>
                                ) : assignOpen ? (
                                    <div style={{ display: 'flex', gap: 6 }}>
                                        <input className="input" placeholder="UUID оператора" value={assignId}
                                               onChange={e => setAssignId(e.target.value)} style={{ fontSize: 12 }} />
                                        <button className="btn btn-primary btn-sm" onClick={handleAssign}>OK</button>
                                        <button className="btn btn-sm" onClick={() => { setAssignOpen(false); setAssignId('') }}>
                                            {ICONS.x}
                                        </button>
                                    </div>
                                ) : (
                                    <button className="btn btn-sm" style={{ width: '100%', justifyContent: 'center' }}
                                            onClick={() => setAssignOpen(true)}>
                                        {ICONS.plus} Назначить оператора
                                    </button>
                                )}
                            </>
                        )}
                    </div>

                    <div className="card">
                        <div className="section-head" style={{ margin: 0 }}>
                            <h2 className="section-title">Телеметрия · 24 часа</h2>
                            <span className="section-meta">{tel.length} точек</span>
                        </div>
                        <div style={{ marginTop: 12 }}>
                            <LineChart series={[
                                { label: 'Влажность, %', color: 'var(--data-water)',  values: tel.map(p => ({ t: new Date(p.recorded_at).getTime(), v: p.soil_moisture })) },
                                { label: 'Здоровье',     color: 'var(--data-health)', values: tel.map(p => ({ t: new Date(p.recorded_at).getTime(), v: (p.health_index || 0) * 100 })) },
                            ]} />
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}