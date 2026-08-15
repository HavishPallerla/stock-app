import { NavLink } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Navbar() {
  const { session, signOut } = useAuth();
  if (!session) return null;

  return (
    <header className="navbar">
      <div className="navbar-inner">
        <span className="brand">📈 Stock Alerts</span>
        <nav className="navbar-links">
          <NavLink to="/dashboard" className={({ isActive }) => (isActive ? 'active' : '')}>
            Dashboard
          </NavLink>
          <NavLink to="/settings" className={({ isActive }) => (isActive ? 'active' : '')}>
            Settings
          </NavLink>
        </nav>
        <button type="button" className="btn-ghost" onClick={() => signOut()}>
          Sign out
        </button>
      </div>
    </header>
  );
}
