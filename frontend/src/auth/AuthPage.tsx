import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { login, register } from './api';
import { passwordsMatch } from './passwords';
import { useSession } from './session';

type Mode = 'login' | 'register';

const title: Record<Mode, string> = {
  login: 'Вход',
  register: 'Регистрация',
};

const submitLabel: Record<Mode, string> = {
  login: 'Войти',
  register: 'Создать игрока',
};

export function AuthPage() {
  const navigate = useNavigate();
  const { setPlayer } = useSession();
  const [mode, setMode] = useState<Mode>('login');
  const [loginName, setLoginName] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);

  function switchMode(next: Mode) {
    setMode(next);
    setError('');
    setConfirm('');
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (mode === 'register' && !passwordsMatch(password, confirm)) {
      setError('Пароли не совпадают');
      return;
    }

    setPending(true);
    setError('');
    const result =
      mode === 'register'
        ? await register(loginName, password)
        : await login(loginName, password);
    setPending(false);

    if (!result.ok) {
      setError(result.message);
      return;
    }
    setPlayer(result.data);
    navigate('/lobby', { replace: true });
  }

  return (
    <main className="screen">
      <form className="panel" onSubmit={onSubmit}>
        <h1>{title[mode]}</h1>
        <div className="mode-switch" role="group" aria-label="Режим">
          <button
            type="button"
            aria-pressed={mode === 'login'}
            onClick={() => switchMode('login')}
          >
            Вход
          </button>
          <button
            type="button"
            aria-pressed={mode === 'register'}
            onClick={() => switchMode('register')}
          >
            Регистрация
          </button>
        </div>
        <label>
          Логин
          <input
            name="login"
            autoComplete="username"
            value={loginName}
            onChange={(event) => setLoginName(event.target.value)}
            required
          />
        </label>
        <label>
          Пароль
          <input
            name="password"
            type="password"
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </label>
        {mode === 'register' ? (
          <label>
            Пароль ещё раз
            <input
              name="confirm"
              type="password"
              autoComplete="new-password"
              value={confirm}
              onChange={(event) => setConfirm(event.target.value)}
              required
            />
          </label>
        ) : null}
        {error ? (
          <p className="form-error" role="alert">
            {error}
          </p>
        ) : null}
        <button type="submit" disabled={pending}>
          {pending ? 'Подождите…' : submitLabel[mode]}
        </button>
      </form>
    </main>
  );
}
