const BASE = '/api'
const WS_BASE = `ws://${window.location.host}/ws`

function getToken() {
    return localStorage.getItem('token')
}

async function request(path, options = {}) {
    const token = getToken()
    const headers = { ...options.headers }

    if (token) {
        headers['Authorization'] = `Bearer ${token}`
    }

    if (!(options.body instanceof FormData)) {
        headers['Content-Type'] = 'application/json'
    }

    const res = await fetch(`${BASE}${path}`, { ...options, headers })
    if (res.status === 204) return null
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'request failed')
    return data
}

export const api = {
    // auth
    login: (email, password) =>
        request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),

    register: (email, password, name, role) =>
        request('/auth/register', { method: 'POST', body: JSON.stringify({ email, password, name, role }) }),

    // sectors
    getSectors: () => request('/sectors'),
    getMySectors: () => request('/sectors/my'),
    getSector: (id) => request(`/sectors/${id}`),
    createSector: (data) => request('/sectors', { method: 'POST', body: JSON.stringify(data) }),
    updateSector: (id, data) => request(`/sectors/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    deleteSector: (id) => request(`/sectors/${id}`, { method: 'DELETE' }),

    // operator assignment
    assignOperator: (sectorId, operatorId) =>
        request(`/sectors/${sectorId}/assign`, { method: 'PUT', body: JSON.stringify({ operator_id: operatorId }) }),
    unassignOperator: (sectorId) =>
        request(`/sectors/${sectorId}/assign`, { method: 'DELETE' }),

    // plants
    getPlants: (sectorId) => request(`/plants${sectorId ? `?sector_id=${sectorId}` : ''}`),
    createPlant: (data) => request('/plants', { method: 'POST', body: JSON.stringify(data) }),
    deletePlant: (id) => request(`/plants/${id}`, { method: 'DELETE' }),

    // watering
    water: (sectorId, volume) =>
        request('/water', { method: 'POST', body: JSON.stringify({ sector_id: sectorId, volume_liters: volume }) }),
    getWaterStats: (sectorId) => request(`/water/stats/${sectorId}`),

    // notifications
    getNotifications: (unreadOnly = false) =>
        request(`/notifications${unreadOnly ? '?unread=true' : ''}`),
    markNotificationRead: (id) =>
        request(`/notifications/${id}/read`, { method: 'PUT' }),

    // telemetry & reports
    getTelemetry: (sectorId, limit = 100) =>
        request(`/telemetry/${sectorId}?limit=${limit}`),
    getReport: (sectorId) => request(`/reports/${sectorId}`),

    // file import/export
    exportSectors: () => {
        const token = getToken()
        return fetch(`${BASE}/export/sectors`, {
            headers: { Authorization: `Bearer ${token}` }
        }).then(r => r.blob())
    },
    importSectors: (file) => {
        const form = new FormData()
        form.append('file', file)
        return request('/import/sectors', { method: 'POST', body: form })
    },
}

export function connectWS(onMessage, onStatus) {
    let ws = null
    let reconnectTimer = null
    let closed = false

    function connect() {
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        ws = new WebSocket(`${proto}//${window.location.host}/ws`)

        ws.onopen = () => { onStatus?.(true) }
        ws.onmessage = e => {
            try { onMessage(JSON.parse(e.data)) } catch {}
        }
        ws.onclose = () => {
            onStatus?.(false)
            if (!closed) reconnectTimer = setTimeout(connect, 3000)
        }
        ws.onerror = () => { try { ws.close() } catch {} }
    }

    connect()
    return () => {
        closed = true
        clearTimeout(reconnectTimer)
        if (ws) { try { ws.close() } catch {} }
    }
}