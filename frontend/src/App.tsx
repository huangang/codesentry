import React, { Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import { useTranslation } from 'react-i18next';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import { useAuthStore } from './stores/authStore';

const Dashboard = React.lazy(() => import('./pages/Dashboard'));
const ReviewLogs = React.lazy(() => import('./pages/ReviewLogs'));
const Projects = React.lazy(() => import('./pages/Projects'));
const MemberAnalysis = React.lazy(() => import('./pages/MemberAnalysis'));
const LLMModels = React.lazy(() => import('./pages/LLMModels'));
const IMBots = React.lazy(() => import('./pages/IMBots'));
const Prompts = React.lazy(() => import('./pages/Prompts'));
const SystemLogs = React.lazy(() => import('./pages/SystemLogs'));
const GitCredentials = React.lazy(() => import('./pages/GitCredentials'));
const Settings = React.lazy(() => import('./pages/Settings'));
const Users = React.lazy(() => import('./pages/Users'));

const PageLoader: React.FC = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 300 }}>
    <Spin size="large" />
  </div>
);

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

import { theme } from './theme';

const App: React.FC = () => {
  const { i18n } = useTranslation();
  const locale = i18n.language?.startsWith('zh') ? zhCN : enUS;

  return (
    <ConfigProvider locale={locale} theme={theme}>
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
            <Route path="dashboard" element={<Suspense fallback={<PageLoader />}><Dashboard /></Suspense>} />
            <Route path="review-logs" element={<Suspense fallback={<PageLoader />}><ReviewLogs /></Suspense>} />
            <Route path="projects" element={<Suspense fallback={<PageLoader />}><Projects /></Suspense>} />
            <Route path="member-analysis" element={<Suspense fallback={<PageLoader />}><MemberAnalysis /></Suspense>} />
            <Route path="llm-models" element={<Suspense fallback={<PageLoader />}><LLMModels /></Suspense>} />
            <Route path="im-bots" element={<Suspense fallback={<PageLoader />}><IMBots /></Suspense>} />
            <Route path="prompts" element={<Suspense fallback={<PageLoader />}><Prompts /></Suspense>} />
            <Route path="sys-logs" element={<Suspense fallback={<PageLoader />}><SystemLogs /></Suspense>} />
            <Route path="git-credentials" element={<Suspense fallback={<PageLoader />}><GitCredentials /></Suspense>} />
            <Route path="settings" element={<Suspense fallback={<PageLoader />}><Settings /></Suspense>} />
            <Route path="users" element={<Suspense fallback={<PageLoader />}><Users /></Suspense>} />
          </Route>
          <Route path="/" element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="*" element={<Navigate to="/admin/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
};

export default App;
