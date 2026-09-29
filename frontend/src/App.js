import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from 'contexts/AuthContext';
import ProtectedRoute from 'routes/ProtectedRoute';
import LoginPage from 'pages/LoginPage';
import RegisterPage from 'pages/RegisterPage';
import DashboardPage from 'pages/DashboardPage';
import UserMenu from 'components/layout/UserMenu';
import './App.css';
import DnaLogo from 'components/brand/DnaLogo';
import ChatWidget from 'components/chat/ChatWidget';

function App() {
  return (
    <AuthProvider>
      <div className="app-shell">
        <header className="app-header">
          <div className="app-header-left">
            <span className="app-logo">
            <DnaLogo size={22} speed={16} />
            <span>EvoFarm</span>
            <span className="app-tagline">Neuroevolution-as-a-Service</span>
            </span>
           
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
        <ChatWidget />
      </div>
    </AuthProvider>
  );
}

export default App;