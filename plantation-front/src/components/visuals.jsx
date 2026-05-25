import { useEffect, useRef, useState } from 'react'

export function HealthRing({ value, size = 88, label }) {
    const R = (size - 14) / 2
    const C = 2 * Math.PI * R
    const pct = Math.max(0, Math.min(100, value || 0))
    const offset = C * (1 - pct / 100)
    const color = pct > 70 ? 'var(--good)' : pct > 40 ? 'var(--warn)' : 'var(--danger)'
    return (
        <div className="ring" style={{ width: size, height: size }}>
            <svg viewBox={`0 0 ${size} ${size}`}>
                <circle className="bg" cx={size/2} cy={size/2} r={R} />
                <circle className="fg" cx={size/2} cy={size/2} r={R}
                        strokeDasharray={C} strokeDashoffset={offset} style={{ stroke: color }} />
            </svg>
            <div className="ring-label">{label ?? Math.round(pct)}</div>
        </div>
    )
}

export function Sparkline({ data, w = 80, h = 28, color = 'var(--brand)', fill = true }) {
    if (!data || data.length < 2) return null
    const min = Math.min(...data)
    const max = Math.max(...data)
    const span = Math.max(1, max - min)
    const pts = data.map((v, i) => {
        const x = (i / (data.length - 1)) * w
        const y = h - ((v - min) / span) * (h - 4) - 2
        return [x, y]
    })
    const path = pts.map((p, i) => (i === 0 ? `M${p[0]},${p[1]}` : `L${p[0]},${p[1]}`)).join(' ')
    const area = `${path} L${w},${h} L0,${h} Z`
    return (
        <svg viewBox={`0 0 ${w} ${h}`} width={w} height={h} preserveAspectRatio="none">
            {fill && <path d={area} fill={color} opacity="0.15" />}
            <path d={path} fill="none" stroke={color} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
    )
}

export function LineChart({ series, height = 180, unit = '' }) {
    const ref = useRef(null)
    const [width, setWidth] = useState(600)

    useEffect(() => {
        if (!ref.current) return
        const ro = new ResizeObserver(e => setWidth(e[0].contentRect.width))
        ro.observe(ref.current)
        return () => ro.disconnect()
    }, [])

    const validSeries = (series || []).filter(s => s.values && s.values.length >= 2)
    if (!validSeries.length) {
        return <div ref={ref} className="empty" style={{ height }}>Недостаточно данных для построения графика.</div>
    }

    const padL = 40, padR = 10, padT = 10, padB = 24
    const W = Math.max(200, width), H = height
    const plotW = W - padL - padR
    const plotH = H - padT - padB

    const all = validSeries.flatMap(s => s.values.map(v => v.v))
    const tAll = validSeries[0].values.map(v => v.t)
    const min = Math.min(...all, 0)
    const max = Math.max(...all, 100)
    const tMin = Math.min(...tAll)
    const tMax = Math.max(...tAll)

    const x = t => padL + ((t - tMin) / Math.max(1, tMax - tMin)) * plotW
    const y = v => padT + (1 - (v - min) / Math.max(1, max - min)) * plotH

    const gridY = 4
    const gridLines = Array.from({ length: gridY + 1 }, (_, i) => {
        const v = min + (i / gridY) * (max - min)
        return { v, y: y(v) }
    })

    const xTicks = [
        { label: '−24ч', t: tMin },
        { label: '−12ч', t: tMin + (tMax - tMin) / 2 },
        { label: 'сейчас', t: tMax },
    ]

    return (
        <div className="chart-wrap" ref={ref}>
            <svg className="chart-svg" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" style={{ height }}>
                {gridLines.map((g, i) => (
                    <g key={i}>
                        <line x1={padL} x2={W - padR} y1={g.y} y2={g.y}
                              stroke="var(--line)" strokeDasharray={i === 0 || i === gridY ? '0' : '3 3'} />
                        <text x={padL - 6} y={g.y + 3} textAnchor="end"
                              fontFamily="var(--mono)" fontSize="10" fill="var(--ink-4)">
                            {Math.round(g.v)}{unit}
                        </text>
                    </g>
                ))}
                {xTicks.map((t, i) => (
                    <text key={i} x={x(t.t)} y={H - 6} textAnchor="middle"
                          fontFamily="var(--mono)" fontSize="10" fill="var(--ink-4)">{t.label}</text>
                ))}
                {validSeries.map((s, i) => {
                    const path = s.values.map((p, j) => `${j === 0 ? 'M' : 'L'}${x(p.t)},${y(p.v)}`).join(' ')
                    const last = s.values[s.values.length - 1]
                    const first = s.values[0]
                    const area = `${path} L${x(last.t)},${padT + plotH} L${x(first.t)},${padT + plotH} Z`
                    return (
                        <g key={i}>
                            <path d={area} fill={s.color} opacity="0.1" />
                            <path d={path} fill="none" stroke={s.color} strokeWidth="1.8"
                                  strokeLinecap="round" strokeLinejoin="round" />
                            <circle cx={x(last.t)} cy={y(last.v)} r="3" fill={s.color} />
                        </g>
                    )
                })}
            </svg>
            <div className="chart-legend">
                {validSeries.map((s, i) => (
                    <span key={i}>
            <i style={{ background: s.color, height: 2, display: 'inline-block', width: 14 }} />
                        {s.label}
          </span>
                ))}
            </div>
        </div>
    )
}

export function PlantVisual({ moisture, health, watering }) {
    const m = moisture ?? 50
    const h = health ?? 50
    const wilt = m < 25 ? 1 - (m / 25) : 0
    const flood = m > 90 ? (m - 90) / 10 : 0
    const vigor = Math.max(0.3, h / 100)
    const leafColor = h > 70 ? 'oklch(0.55 0.14 140)' : h > 40 ? 'oklch(0.55 0.14 100)' : 'oklch(0.48 0.12 60)'
    const leafColor2 = h > 70 ? 'oklch(0.65 0.15 145)' : h > 40 ? 'oklch(0.62 0.14 95)' : 'oklch(0.55 0.13 55)'

    return (
        <svg viewBox="0 0 200 200" style={{ width: '100%', height: '100%' }}>
            <defs>
                <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0" stopColor="color-mix(in srgb, var(--info) 12%, var(--bg-sunken))" />
                    <stop offset="1" stopColor="var(--bg-sunken)" />
                </linearGradient>
                <linearGradient id="soil" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0" stopColor={flood > 0 ? 'oklch(0.5 0.06 240)' : 'oklch(0.38 0.04 50)'} />
                    <stop offset="1" stopColor="oklch(0.28 0.04 40)" />
                </linearGradient>
                <radialGradient id="canopy" cx="0.5" cy="0.6">
                    <stop offset="0" stopColor={leafColor2} />
                    <stop offset="1" stopColor={leafColor} />
                </radialGradient>
            </defs>
            <rect width="200" height="200" fill="url(#sky)" />
            <rect y="150" width="200" height="50" fill="url(#soil)" />
            {flood > 0 && (
                <ellipse cx="100" cy="152" rx="88" ry={4 + flood * 5}
                         fill="color-mix(in srgb, var(--data-water) 55%, transparent)" />
            )}
            <path
                d={`M 96 ${155 - wilt * 4} Q ${100 + wilt * 6} ${110 - vigor * 10} ${104 + wilt * 8} ${80 - vigor * 20}`}
                stroke="oklch(0.35 0.04 60)"
                strokeWidth={5 + vigor * 2}
                strokeLinecap="round"
                fill="none"
            />
            <path d={`M 102 ${95} Q ${80 + wilt*8} ${80 + wilt*10} ${72 + wilt*14} ${70 + wilt*18}`}
                  stroke="oklch(0.35 0.04 60)" strokeWidth="2" fill="none" strokeLinecap="round" />
            <path d={`M 100 ${85} Q ${118 - wilt*4} ${74 + wilt*8} ${128 - wilt*4} ${66 + wilt*14}`}
                  stroke="oklch(0.35 0.04 60)" strokeWidth="2" fill="none" strokeLinecap="round" />
            <g transform={`translate(${wilt * 4} ${wilt * 8})`}>
                <circle cx="72" cy={66 + wilt*10} r={20 + vigor * 8} fill="url(#canopy)" opacity="0.92" />
                <circle cx="108" cy={54 + wilt*10} r={26 + vigor * 10} fill="url(#canopy)" />
                <circle cx="134" cy={64 + wilt*10} r={22 + vigor * 9} fill="url(#canopy)" opacity="0.94" />
                {h > 55 && [[82,72],[118,58],[128,78],[100,72]].map(([cx,cy], i) => (
                    <circle key={i} cx={cx} cy={cy + wilt*10} r="3" fill="oklch(0.55 0.18 20)" />
                ))}
            </g>
            <rect x="20" y={160} width={160 * Math.min(1, m/100)} height="4" rx="2"
                  fill="var(--data-water)" opacity="0.6" />
            {watering && (
                <g>
                    {Array.from({ length: 6 }).map((_, i) => (
                        <circle key={i} cx={90 + (i * 6) - 15} cy="20" r="2" fill="var(--data-water)">
                            <animate attributeName="cy" from="20" to="150" dur="0.8s" begin={`${i * 0.1}s`} repeatCount="indefinite" />
                            <animate attributeName="opacity" from="1" to="0" dur="0.8s" begin={`${i * 0.1}s`} repeatCount="indefinite" />
                        </circle>
                    ))}
                </g>
            )}
        </svg>
    )
}