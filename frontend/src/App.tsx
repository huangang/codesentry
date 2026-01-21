import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './layouts/MainLayout';
import { 
  Login, 
  Dashboard, 
  ReviewLogs, 
  Projects, 
  MemberAnalysis, 
  LLMModels, 
  IMBots, 
  SystemLogs,
  Prompts 
} from './pages';
import { useAuthStore } from './stores/authStore';

// Protected Route wrapper
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  
  return <>{children}</>;
};

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Navigate to="/admin/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="review-logs" element={<ReviewLogs />} />
            <Route path="projects" element={<Projects />} />
            <Route path="member-analysis" element={<MemberAnalysis />} />
            <Route path="llm-models" element={<LLMModels />} />
            <Route path="im-bots" element={<IMBots />} />
            <Route path="prompts" element={<Prompts />} />
            <Route path="sys-logs" element={<SystemLogs />} />
          </Route>
          <Route path="/" element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="*" element={<Navigate to="/admin/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
