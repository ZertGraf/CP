import { useState } from 'react'
import { api } from '../api'
import { ICONS } from '../components/icons'

export default function Login({ onLogin }) {
    const [mode, setMode] = useState('login')
    const [role, setRole] = useState('operator')
    const [email, setEmail] = useState('agro@test.com')
    const [pass, setPass] = useState('123456')
    const [name, setName] = useState('')
    const [error, setError] = useState('')
    const [busy, setBusy] = useState(false)

    async function submit(e) {
        e.preventDefault()
        setError('')
        setBusy(true)
        try {
            if (mode === 'register') {
                await api.register(email, pass, name || email.split('@')[0], role)
            }
            const data = await api.login(email, pass)
            localStorage.setItem('token', data.token)
            localStorage.setItem('role', data.role)
            localStorage.setItem('name', data.name)
            localStorage.setItem('id', data.id)
            onLogin()
        } catch (err) {
            setError(err.message || 'Ошибка входа')
        } finally {
            setBusy(false)
        }
    }

    return (
        <div className="login-wrap">
            <div className="login-hero">
                <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 28 }}>
                        <div className="brand-mark" style={{ background: 'rgba(255,255,255,.15)' }}>
                            {ICONS.leaf}
                        </div>
                        <div>
                            <div style={{ fontWeight: 600, fontSize: 15 }}>Plantation Control</div>
                            <div style={{ fontSize: 11, opacity: 0.7, fontFamily: 'var(--mono)', letterSpacing: '0.04em', textTransform: 'uppercase' }}>
                                Irrigation Ops · v2.4
                            </div>
                        </div>
                    </div>
                    <h1>Управляйте ирригацией<br/>как живым организмом.</h1>
                    <p>Имитационная модель, real-time телеметрия и ролевой доступ — единый контур для агронома и оператора полива на плантациях личи.</p>
                </div>
                <div className="login-hero-stats">
                    <div><div className="v">6</div><div className="l">Секторов</div></div>
                    <div><div className="v">54</div><div className="l">Растений</div></div>
                    <div><div className="v">10с</div><div className="l">Tick модели</div></div>
                </div>
            </div>

            <div className="login-form-cell">
                <div className="login-card">
                    <h2>{mode === 'login' ? 'Вход в систему' : 'Регистрация'}</h2>
                    <p className="sub">
                        {mode === 'login'
                            ? 'Введите учётные данные для доступа к плантации.'
                            : 'Создайте учётную запись и выберите роль.'}
                    </p>
                    {error && <div className="error">{error}</div>}
                    <form onSubmit={submit}>
                        {mode === 'register' && (
                            <>
                                <div className="field">
                                    <label className="field-label">Имя</label>
                                    <input className="input" placeholder="Иванов Пётр" value={name} onChange={e => setName(e.target.value)} />
                                </div>
                                <div className="field">
                                    <label className="field-label">Роль</label>
                                    <div className="role-pick">
                                        <button type="button" className={role === 'operator' ? 'active' : ''} onClick={() => setRole('operator')}>
                                            <strong>Оператор</strong>
                                            <small>Полив закреплённых секторов</small>
                                        </button>
                                        <button type="button" className={role === 'agronomist' ? 'active' : ''} onClick={() => setRole('agronomist')}>
                                            <strong>Агроном</strong>
                                            <small>Управление плантацией</small>
                                        </button>
                                    </div>
                                </div>
                            </>
                        )}
                        <div className="field">
                            <label className="field-label">Email</label>
                            <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} />
                        </div>
                        <div className="field">
                            <label className="field-label">Пароль</label>
                            <input className="input" type="password" value={pass} onChange={e => setPass(e.target.value)} />
                        </div>
                        <button className="btn btn-primary btn-lg" type="submit" disabled={busy}
                                style={{ marginTop: 6, justifyContent: 'center' }}>
                            {busy ? '...' : (mode === 'login' ? 'Войти' : 'Создать аккаунт')}
                        </button>
                    </form>
                    <div className="login-demo-creds">
                        demo: <strong>agro@test.com</strong> (агроном) · <strong>op1@test.com</strong> (оператор) · пароль <strong>123456</strong>
                    </div>
                    <p style={{ marginTop: 14, fontSize: 12, color: 'var(--ink-3)', textAlign: 'center' }}>
                        <a style={{ color: 'var(--brand)', cursor: 'pointer' }}
                           onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
                            {mode === 'login' ? 'Нет аккаунта? Регистрация' : 'Уже есть аккаунт? Войти'}
                        </a>
                    </p>
                </div>
            </div>
        </div>
    )
}