export function classNames(...xs) {
    return xs.filter(Boolean).join(' ')
}

export function fmtTimeAgo(isoOrDate) {
    if (!isoOrDate) return '—'
    const t = new Date(isoOrDate).getTime()
    if (Number.isNaN(t)) return '—'
    const min = Math.floor((Date.now() - t) / 60000)
    if (min < 1) return 'только что'
    if (min < 60) return `${min} мин назад`
    const h = Math.floor(min / 60)
    if (h < 24) return `${h} ч назад`
    return `${Math.floor(h / 24)} д назад`
}

export function resolveStatus(sector) {
    if (!sector) return 'normal'
    if (sector.status) return sector.status
    const m = sector.soil_moisture
    if (m < 20) return 'drought'
    if (m > 90) return 'overwatered'
    if (m < 30) return 'warn'
    return 'normal'
}

export const STATUS_LABEL = {
    normal:      { ru: 'В норме',          chip: 'chip-good' },
    drought:     { ru: 'Засуха',           chip: 'chip-danger' },
    overwatered: { ru: 'Переувлажнено',    chip: 'chip-info' },
    heat_stress: { ru: 'Тепловой стресс',  chip: 'chip-warn' },
    critical:    { ru: 'Критическое',      chip: 'chip-danger' },
    recovering:  { ru: 'Восстановление',   chip: 'chip-good' },
    dead:        { ru: 'Гибель',           chip: 'chip-danger' },
    warn:        { ru: 'Внимание',         chip: 'chip-warn' },
}

export function shortOperatorId(id) {
    if (!id) return null
    return id.slice(0, 4).toUpperCase()
}

export function notifKindLabel(kind) {
    switch (kind) {
        case 'critical_drought': return 'Критическая засуха'
        case 'drought_warning':  return 'Низкая влажность'
        case 'flood_warning':    return 'Переувлажнение'
        case 'health_critical':  return 'Критическое здоровье'
        case 'plant_dead':       return 'Гибель растений'
        default: return kind
    }
}
