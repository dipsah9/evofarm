import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './AuthContext';
import ProtectedRoute from './ProtectedRoute';
import LoginPage from './LoginPage';
import RegisterPage from './RegisterPage';
import DashboardPage from './DashboardPage';
import UserMenu from './UserMenu';
import './App.css';

function App() {
  return (
    <AuthProvider>
      <div className="app-shell">
        <header className="app-header">
          <div className="app-header-left">
            <span className="app-logo">🧬 EvoFarm</span>
            <span className="app-tagline">Neuroevolution-as-a-Service</span>
          </div>
          <UserMenu />
        </header>

        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </AuthProvider>
  );
}

export default App;