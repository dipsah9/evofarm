import React, { useState } from 'react';
import { useAuth } from './AuthContext';

function UserMenu() {
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);

  if (!user) return null;

  return (
    <div className="user-menu">
      <button
        className="user-menu-trigger"
        onClick={() => setOpen(!open)}
      >
        <span className="user-avatar">
          {(user.name || user.email || '?')[0].toUpperCase()}
        </span>
        <span className="user-email">{user.email}</span>
        <span className="user-chevron">▾</span>
      </button>

      {open && (
        <div className="user-menu-dropdown">
          {user.name && (
            <div className="user-menu-name">{user.name}</div>
          )}
          <button className="user-menu-item" onClick={logout}>
            Sign out
          </button>
        </div>
      )}
    </div>
  );
}

export default UserMenu;