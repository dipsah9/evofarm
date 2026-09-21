import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from './AuthContext';

function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(email, password);
      navigate('/');
    } catch (err) {
      const msg = err.response?.data || 'Login failed. Check your credentials.';
      setError(typeof msg === 'string' ? msg : 'Login failed.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-layout">
      {/* Left panel: what this is */}
      <div className="auth-hero">
        <div className="auth-hero-inner">
          <h1 className="auth-hero-logo">🧬 EvoFarm</h1>
          <p className="auth-hero-tagline">
            A distributed platform for evolving neural networks and
            solving constraint optimization problems.
          </p>

          <ul className="auth-hero-features">
            <li>
              <strong>Two solvers.</strong> Genetic evolution and CP-SAT
              (OR-Tools) behind one API.
            </li>
            <li>
              <strong>Three problems.</strong> XOR, portfolio
              optimization, and nurse rostering.
            </li>
            <li>
              <strong>Live progress.</strong> Watch fitness climb in
              real time — or see a schedule solve to optimality.
            </li>
            <li>
              <strong>Durable history.</strong> Every job is saved and
              tied to your account.
            </li>
          </ul>

          <div className="auth-hero-footer">
            <a
              href="https://github.com/dipsah9/evofarm"
              target="_blank"
              rel="noopener noreferrer"
              className="auth-hero-link"
            >
              View source on GitHub →
            </a>
          </div>
        </div>
      </div>

      {/* Right panel: the form */}
      <div className="auth-form-panel">
        <div className="auth-card">
          <h2 className="auth-title">Sign in</h2>
          <p className="auth-subtitle">
            Welcome back. Enter your credentials to continue.
          </p>

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoFocus
                placeholder="you@example.com"
              />
            </div>
            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                placeholder="••••••••"
              />
            </div>

            {error && <div className="auth-error">{error}</div>}

            <button type="submit" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>

          <p className="auth-footer">
            No account?{' '}
            <Link to="/register">Create one — it's free</Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default LoginPage;