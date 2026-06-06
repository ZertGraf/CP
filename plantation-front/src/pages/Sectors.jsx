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
    const [createOpen, setCreateOpen] = useState(false)
    const [createName, setCreateName] = useState('')
    const [createArea, setCreateArea] = useState('')

    // simulation override state (agronomist only)
    const [ovTemp, setOvTemp] = useState(25)
    const [ovMoist, setOvMoist] = useState(60)
    const [ovHealth, setOvHealth] = useState(100)
    const [ovLimit, setOvLimit] = useState(500)
    const [ovApplying, setOvApplying] = useState(false)

    const filtered = sectors.filter(s => s.name.toLowerCase().includes(query.toLowerCase()))
    const canWater = selected && (isAgro || selected.operator_id === user.id)

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

    async function handleTreat() {
        if (!selected) return
        try {
            const res = await api.treat(selected.id)
            setSectors(prev => prev.map(s => s.id === res.sector.id ? res.sector : s))
            onToast('Сектор обработан от вредителей', 'success')
        } catch (err) {
            onToast(err.message || 'Ошибка обработки', 'danger')
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

    async function handleExportXlsx() {
        if (!selected) { onToast('Выберите сектор для экспорта телеметрии', 'danger'); return }
        try {
            const blob = await api.exportTelemetry(selected.id)
            const url = URL.createObjectURL(blob)
            const a = document.createElement('a')
            a.href = url; a.download = `telemetry_${selected.name}.xlsx`; a.click()
            URL.revokeObjectURL(url)
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    async function handleCreate() {
        const name = createName.trim()
        const area = parseFloat(createArea)
        if (!name) { onToast('Введите название сектора', 'danger'); return }
        if (!(area > 0)) { onToast('Введите корректную площадь', 'danger'); return }
        try {
            await api.createSector({ name, area_sqm: area })
            const list = await api.getSectors()
            setSectors(list || [])
            setCreateOpen(false)
            setCreateName('')
            setCreateArea('')
            onToast(`Сектор «${name}» создан`, 'success')
        } catch (err) {
            onToast(err.message || 'Ошибка создания', 'danger')
        }
    }

    async function handleImport(file) {
        const f = file || importFile
        if (!f) return
        try {
            const r = await api.importSectors(f)
            setImportFile(null)
            const list = await api.getSectors()
            setSectors(list || [])
            const skipped = r.skipped ? `, пропущено: ${r.skipped}` : ''
            onToast(`Импортировано: ${r.imported}${skipped}`, 'success')
        } catch (err) {
            onToast(err.message || 'Ошибка импорта', 'danger')
        }
    }

    const tel = selected ? (telemetry[selected.id] || []) : []

    // sync override sliders when sector selection changes
    useEffect(() => {
        if (!selected) return
        setOvTemp(Math.round(selected.temperature || 25))
        setOvMoist(Math.round(selected.soil_moisture || 60))
        setOvHealth(Math.round((selected.health_index || 1) * 100))
        setOvLimit(Math.round(selected.daily_water_limit || 500))
    }, [selected?.id])

    async function sendOverride(payload) {
        if (!selected) return
        setOvApplying(true)
        try {
            const res = await api.overrideSector(selected.id, payload)
            setSectors(prev => prev.map(s => s.id === res.id ? res : s))
            onToast('Параметры применены', 'success')
        } catch (err) {
            onToast(err.message || 'Ошибка', 'danger')
        } finally {
            setOvApplying(false)
        }
    }

    function handleApplyOverride() {
        sendOverride({
            temperature: ovTemp,
            soil_moisture: ovMoist,
            health_index: ovHealth / 100,
            daily_water_limit: ovLimit,
        })
    }

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
                            <button className="btn btn-primary" onClick={() => setCreateOpen(true)}>{ICONS.plus} Новый сектор</button>
                            <label className="btn" style={{ cursor: 'pointer' }}>
                                {ICONS.upload} Импорт CSV
                                <input type="file" accept=".csv" hidden
                                       onChange={e => { const f = e.target.files[0]; e.target.value = ''; if (f) handleImport(f) }} />
                            </label>
                            <button className="btn" onClick={handleExport}>{ICONS.download} CSV</button>
                            <button className="btn" onClick={handleExportXlsx} title="Экспорт телеметрии выбранного сектора в XLSX">{ICONS.download} XLSX</button>
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

            {createOpen && (
                <div className="modal-overlay" onClick={() => setCreateOpen(false)}>
                    <div className="modal-box" onClick={e => e.stopPropagation()}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                            <h2 style={{ margin: 0, fontSize: 16 }}>Новый сектор</h2>
                            <button className="btn btn-sm" onClick={() => setCreateOpen(false)}>{ICONS.x}</button>
                        </div>
                        <div className="stack" style={{ gap: 12 }}>
                            <div>
                                <div className="field-label" style={{ marginBottom: 4 }}>Название</div>
                                <input className="input" placeholder="Сектор A-1" value={createName}
                                       onChange={e => setCreateName(e.target.value)}
                                       onKeyDown={e => e.key === 'Enter' && handleCreate()} />
                            </div>
                            <div>
                                <div className="field-label" style={{ marginBottom: 4 }}>Площадь, м²</div>
                                <input className="input" type="number" min="1" placeholder="1000"
                                       value={createArea} onChange={e => setCreateArea(e.target.value)}
                                       onKeyDown={e => e.key === 'Enter' && handleCreate()} />
                            </div>
                            <button className="btn btn-primary" style={{ width: '100%', justifyContent: 'center' }}
                                    onClick={handleCreate}>
                                Создать
                            </button>
                        </div>
                    </div>
                </div>
            )}

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

                        {(selected.equipment_locked_ticks || 0) > 0 && (
                            <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--danger)', textAlign: 'center', padding: '6px 0' }}>
                                🔧 Поломка оборудования · полив недоступен ({selected.equipment_locked_ticks} тик.)
                            </div>
                        )}

                        {selected.pest_active && (
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, padding: '8px 10px', background: '#fffbeb', border: '1px solid #fde68a', borderRadius: 8 }}>
                                <span style={{ fontSize: 12, fontWeight: 600, color: '#b45309' }}>🐛 Вредители · здоровье падает</span>
                                {canWater && <button className="btn btn-sm btn-primary" onClick={handleTreat}>Обработать</button>}
                            </div>
                        )}

                        {/* math-model coefficients incl. CWSI (chapter 2.3.5) */}
                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, fontSize: 12 }}>
                            <div title="Коэффициент водного стресса (FAO-56)">
                                <span style={{ color: 'var(--ink-3)' }}>Ks</span>{' '}
                                <strong className="mono tnum">{(selected.ks_water ?? 1).toFixed(2)}</strong>
                            </div>
                            <div title="Аэрационный стресс при переувлажнении">
                                <span style={{ color: 'var(--ink-3)' }}>Ks,aer</span>{' '}
                                <strong className="mono tnum">{(selected.ks_aeration ?? 1).toFixed(2)}</strong>
                            </div>
                            <div title="Crop Water Stress Index (0 — обводнено, 1 — критический стресс)">
                                <span style={{ color: 'var(--ink-3)' }}>CWSI</span>{' '}
                                <strong className="mono tnum">{(selected.cwsi ?? 0).toFixed(2)}</strong>
                            </div>
                            <div title="Дефицит влаги корневой зоны, мм">
                                <span style={{ color: 'var(--ink-3)' }}>Dr</span>{' '}
                                <strong className="mono tnum">{(selected.deficit_dr || 0).toFixed(1)} мм</strong>
                            </div>
                            <div title="Сумма эффективных температур / фенофаза BBCH" style={{ gridColumn: '1 / -1' }}>
                                <span style={{ color: 'var(--ink-3)' }}>GDD / фаза</span>{' '}
                                <strong className="mono tnum">{(selected.gdd_cumulative || 0).toFixed(0)} / {selected.phenophase || '00'}</strong>
                            </div>
                        </div>

                        <div className="divider" />

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

                    {isAgro && (
                        <div className="card">
                            <div className="section-head" style={{ margin: 0 }}>
                                <h2 className="section-title">Настройки симуляции</h2>
                                <span className="section-meta">{selected.name}</span>
                            </div>

                            <div style={{ marginTop: 14 }}>
                                <div className="field-label" style={{ marginBottom: 8 }}>Быстрые события</div>
                                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                                    {[
                                        { event: 'drought',    label: 'Засуха',         color: 'var(--danger)' },
                                        { event: 'flood',      label: 'Переувлажнение', color: 'var(--info)'   },
                                        { event: 'heat',       label: 'Жара',           color: '#b06f10'       },
                                        { event: 'pest',       label: 'Вредители',      color: '#8a5030'       },
                                        { event: 'restore',    label: 'Восстановить',   color: 'var(--good)'   },
                                    ].map(({ event, label, color }) => (
                                        <button key={event}
                                                className="btn btn-sm"
                                                style={{ color, borderColor: color, fontWeight: 600 }}
                                                disabled={ovApplying}
                                                onClick={() => sendOverride({ event })}>
                                            {label}
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <div className="divider" style={{ margin: '14px 0' }} />

                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                                <div>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                        <span className="field-label">Температура</span>
                                        <span className="mono tnum" style={{ fontWeight: 600, color: 'var(--ink)' }}>{ovTemp}°C</span>
                                    </div>
                                    <input type="range" min="15" max="48" value={ovTemp}
                                           onChange={e => setOvTemp(+e.target.value)}
                                           style={{ width: '100%', accentColor: 'var(--data-temp)' }} />
                                </div>
                                <div>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                        <span className="field-label">Влажность</span>
                                        <span className="mono tnum" style={{ fontWeight: 600, color: 'var(--ink)' }}>{ovMoist}%</span>
                                    </div>
                                    <input type="range" min="0" max="100" value={ovMoist}
                                           onChange={e => setOvMoist(+e.target.value)}
                                           style={{ width: '100%', accentColor: 'var(--data-water)' }} />
                                </div>
                                <div>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                        <span className="field-label">Здоровье</span>
                                        <span className="mono tnum" style={{ fontWeight: 600, color: 'var(--ink)' }}>{ovHealth}/100</span>
                                    </div>
                                    <input type="range" min="0" max="100" value={ovHealth}
                                           onChange={e => setOvHealth(+e.target.value)}
                                           style={{ width: '100%', accentColor: 'var(--data-health)' }} />
                                </div>
                                <div>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--ink-3)', marginBottom: 4 }}>
                                        <span className="field-label">Лимит / сутки</span>
                                        <span className="mono tnum" style={{ fontWeight: 600, color: 'var(--ink)' }}>{ovLimit} л</span>
                                    </div>
                                    <input type="range" min="50" max="2000" step="50" value={ovLimit}
                                           onChange={e => setOvLimit(+e.target.value)}
                                           style={{ width: '100%', accentColor: 'var(--brand)' }} />
                                </div>
                            </div>

                            <div style={{ display: 'flex', gap: 8, marginTop: 14 }}>
                                <button className="btn btn-primary" style={{ flex: 1, justifyContent: 'center' }}
                                        disabled={ovApplying} onClick={handleApplyOverride}>
                                    {ovApplying ? 'Применяю...' : 'Применить'}
                                </button>
                                <button className="btn btn-sm" title="Сбросить потребление воды"
                                        disabled={ovApplying}
                                        onClick={() => sendOverride({ water_consumed: 0 })}>
                                    Сброс воды
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}