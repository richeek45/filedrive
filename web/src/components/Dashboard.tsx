import { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import authService from '../services/auth';

const Dashboard = () => {
  const { user, logout } = useAuth();
  const [data, setData] = useState(null);

  useEffect(() => {
    // fetchProtectedData();
  }, []);

  const fetchProtectedData = async () => {
    try {
      const response = await authService.fetchWithAuth(
        `${authService.getApiUrl()}/protected-data`
      );
      const result = await response.json();
      setData(result);
    } catch (error) {
      console.error('Failed to fetch protected data:', error);
    }
  };

  return (
    <div>
      <h1>Dashboard</h1>
      <p>Welcome, {user?.email}!</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
};

export default Dashboard;