import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import authService from '../services/auth';

const OAuthCallback = () => {
  const navigate = useNavigate();

  useEffect(() => {
    const handleCallback = async () => {
    const hasValidTokens = !!localStorage.getItem('access_token');
    const tokenExpiryTime = localStorage.getItem('token_expiry');
    const hasTokenExpired = tokenExpiryTime && (Date.now() > parseInt(tokenExpiryTime as string, 10) * 1000)
    
    if (!hasValidTokens || hasTokenExpired) {
      const success = authService.handleCallback();
      
      if (success) {
        navigate('/dashboard', { replace: true });
      } else {
        navigate('/login?error=oauth_failed', { replace: true });
      }
    } else {
      navigate('/dashboard', { replace: true });
    }
  };

    handleCallback();
  }, [navigate]);

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center px-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
        {/* Loading spinner */}
        <div className="mb-6">
          <div className="inline-block h-12 w-12 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]"></div>
        </div>
        
        <h2 className="text-2xl font-bold text-gray-900 mb-3">
          Processing login...
        </h2>
        
        <p className="text-gray-600">
          Please wait while we complete your authentication.
        </p>
      </div>
    </div>
  );
};

export default OAuthCallback;