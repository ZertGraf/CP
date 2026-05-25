import { useState, useEffect } from 'react'
import { api } from '../api'
import { ICONS } from '../components/icons'

export default function Plants({ sectors, onToast }) {
    const [plants, setPlants] = useState([])
    const [filter, setFilter] = useState('')
    const [species, setSpecies] = useState('Litchi chinensis')
    const [age, setAge] = useState('')
    const [sectorId, setSectorId] = useState('')
    const [adding, setAdding] = useState(false)

    async function load() {
        try {
            const p = await api.getPlants(filter)
            setPlants(p || [])
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    useEffect(() => { load() }, [filter])

    async function handleAdd(e) {
        e.preventDefault()
        if (!sectorId) return onToast('Выберите сектор', 'warn')
        try {
            await api.createPlant({ sector_id: sectorId, species, age_months: parseInt(age) || 0 })
            setAdding(false)
            setSpecies('Litchi chinensis')
            setAge('')
            load()
            onToast('Растение добавлено', 'success')
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    async function handleDelete(id) {
        if (!window.confirm('Удалить растение?')) return
        try {
            await api.deletePlant(id)
            load()
        } catch (err) {
            onToast(err.message, 'danger')
        }
    }

    return (
        <div className="fade-in">
            <div className="page-head">
                <div>
                    <h1 className="page-title">Растения</h1>
                    <p className="page-subtitle">{plants.length} растений на плантации.</p>
                </div>
                <div className="page-head-actions">
                    <select className="select" value={filter} onChange={e => setFilter(e.target.value)} style={{ width: 240 }}>
                        <option value="">Все секторы</option>
                        {sectors.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                    </select>
                    <button className="btn btn-primary" onClick={() => setAdding(!adding)}>
                        {ICONS.plus} Добавить
                    </button>
                </div>
            </div>

            {adding && (
                <div className="card fade-in" style={{ marginBottom: 16 }}>
                    <form onSubmit={handleAdd} style={{ display: 'grid', gridTemplateColumns: '1.5fr 1fr 1fr auto', gap: 8, alignItems: 'end' }}>
                        <div className="field">
                            <label className="field-label">Сектор</label>
                            <select className="select" value={sectorId} onChange={e => setSectorId(e.target.value)}>
                                <option value="">Выберите сектор</option>
                                {sectors.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                            </select>
                        </div>
                        <div className="field">
                            <label className="field-label">Вид</label>
                            <input className="input" value={species} onChange={e => setSpecies(e.target.value)} />
                        </div>
                        <div className="field">
                            <label className="field-label">Возраст, мес.</label>
                            <input className="input" type="number" min="0" value={age} onChange={e => setAge(e.target.value)} />
                        </div>
                        <button className="btn btn-primary" type="submit">Посадить</button>
                    </form>
                </div>
            )}

            <div className="card" style={{ padding: 0 }}>
                <table className="table">
                    <thead>
                    <tr>
                        <th style={{ width: 48 }}></th>
                        <th>Вид</th>
                        <th>Сектор</th>
                        <th style={{ textAlign: 'right' }}>Возраст</th>
                        <th style={{ textAlign: 'right' }}>Здоровье</th>
                        <th style={{ width: 60 }}></th>
                    </tr>
                    </thead>
                    <tbody>
                    {plants.map(p => {
                        const sec = sectors.find(s => s.id === p.sector_id)
                        return (
                            <tr key={p.id}>
                                <td>
                                    <div style={{
                                        width: 32, height: 32, borderRadius: 8,
                                        background: 'var(--brand-soft)',
                                        display: 'grid', placeItems: 'center',
                                        color: 'var(--brand)',
                                    }}>
                                        {ICONS.plants}
                                    </div>
                                </td>
                                <td>
                                    <div style={{ fontWeight: 500 }}>{p.species}</div>
                                    <div style={{ fontSize: 11, color: 'var(--ink-4)', fontFamily: 'var(--mono)' }}>
                                        #{p.id.slice(0, 8)}
                                    </div>
                                </td>
                                <td>{sec?.name || '—'}</td>
                                <td className="num" style={{ textAlign: 'right' }}>{p.age_months} мес</td>
                                <td style={{ textAlign: 'right' }}>
                                    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                                        <div style={{ width: 60, height: 4, background: 'var(--bg-sunken)', borderRadius: 2, overflow: 'hidden' }}>
                                            <div style={{
                                                width: `${p.health}%`, height: '100%',
                                                background: p.health > 70 ? 'var(--good)' : p.health > 40 ? 'var(--warn)' : 'var(--danger)',
                                            }} />
                                        </div>
                                        <span className="mono tnum" style={{ fontWeight: 500 }}>{p.health}</span>
                                    </div>
                                </td>
                                <td style={{ textAlign: 'right' }}>
                                    <button className="btn btn-sm btn-ghost" onClick={() => handleDelete(p.id)}>{ICONS.x}</button>
                                </td>
                            </tr>
                        )
                    })}
                    {plants.length === 0 && (
                        <tr><td colSpan={6} className="empty">Нет растений.</td></tr>
                    )}
                    </tbody>
                </table>
            </div>
        </div>
    )
}