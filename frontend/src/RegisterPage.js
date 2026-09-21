import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from './AuthContext';

function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    setLoading(true);
    try {
      await register(email, password, name);
      navigate('/');
    } catch (err) {
      const msg = err.response?.data || 'Registration failed.';
      setError(typeof msg === 'string' ? msg : 'Registration failed.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-layout">
      <div className="auth-hero">
        <div className="auth-hero-inner">
          <h1 className="auth-hero-logo">🧬 EvoFarm</h1>
          <p className="auth-hero-tagline">
            Create an account to submit neuroevolution and constraint
            optimization jobs — and watch them solve in real time.
          </p>

          <ul className="auth-hero-features">
            <li>
              <strong>Free to use.</strong> No credit card, no
              commitment.
            </li>
            <li>
              <strong>Your jobs, your data.</strong> Only you can see
              your history.
            </li>
            <li>
              <strong>Built with Go, Python, React.</strong> Open
              source on GitHub.
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

      <div className="auth-form-panel">
        <div className="auth-card">
          <h2 className="auth-title">Create your account</h2>
          <p className="auth-subtitle">
            Takes 10 seconds. No credit card required.
          </p>

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label>Name <span className="optional">(optional)</span></label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus
                placeholder="Your name"
              />
            </div>
            <div className="form-group">
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                placeholder="you@example.com"
              />
            </div>
            <div className="form-group">
              <label>Password <span className="optional">(min 8 characters)</span></label>
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
              {loading ? 'Creating account...' : 'Create Account'}
            </button>
          </form>

          <p className="auth-footer">
            Already have an account?{' '}
            <Link to="/login">Sign in</Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default RegisterPage;