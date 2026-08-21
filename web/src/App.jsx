/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { lazy, Suspense, useContext, useMemo } from 'react';
import { Route, Routes, useLocation, useParams } from 'react-router-dom';
import Loading from './components/common/ui/Loading';
import {
  AuthRedirect,
  PrivateRoute,
  AdminRoute,
  RootRoute,
  AdminOrDistributorRoute,
} from './helpers/auth';
import { StatusContext } from './context/Status';
import SetupCheck from './components/layout/SetupCheck';

const Home = lazy(() => import('./pages/Home'));
const User = lazy(() => import('./pages/User'));
const RegisterForm = lazy(() => import('./components/auth/RegisterForm'));
const LoginForm = lazy(() => import('./components/auth/LoginForm'));
const NotFound = lazy(() => import('./pages/NotFound'));
const Forbidden = lazy(() => import('./pages/Forbidden'));
const Setting = lazy(() => import('./pages/Setting'));
const PasswordResetForm = lazy(
  () => import('./components/auth/PasswordResetForm'),
);
const PasswordResetConfirm = lazy(
  () => import('./components/auth/PasswordResetConfirm'),
);
const Channel = lazy(() => import('./pages/Channel'));
const Token = lazy(() => import('./pages/Token'));
const SeedanceMaterial = lazy(() => import('./pages/Seedance'));
const Redemption = lazy(() => import('./pages/Redemption'));
const TopUp = lazy(() => import('./pages/TopUp'));
const Log = lazy(() => import('./pages/Log'));
const AliyunGuardrail = lazy(() => import('./pages/AliyunGuardrail'));
const RealNameVerification = lazy(() => import('./pages/RealNameVerification'));
const RealNameVerificationStart = lazy(
  () => import('./pages/RealNameVerificationStart'),
);
const RealNameVerificationResult = lazy(
  () => import('./pages/RealNameVerificationResult'),
);
const Chat = lazy(() => import('./pages/Chat'));
const Chat2Link = lazy(() => import('./pages/Chat2Link'));
const Midjourney = lazy(() => import('./pages/Midjourney'));
const Pricing = lazy(() => import('./pages/Pricing'));
const Task = lazy(() => import('./pages/Task'));
const ModelPage = lazy(() => import('./pages/Model'));
const ModelDeploymentPage = lazy(() => import('./pages/ModelDeployment'));
const ModelHeatPage = lazy(() => import('./pages/ModelHeat'));
const Playground = lazy(() => import('./pages/Playground'));
const Subscription = lazy(() => import('./pages/Subscription'));
const OAuth2Callback = lazy(() => import('./components/auth/OAuth2Callback'));
const PersonalSetting = lazy(
  () => import('./components/settings/PersonalSetting'),
);
const InvoicePage = lazy(() => import('./pages/Invoice'));
const InvoiceAdminPage = lazy(() => import('./pages/InvoiceAdmin'));
const SettlementExportPage = lazy(() => import('./pages/SettlementExport'));
const RoutePolicyPage = lazy(() => import('./pages/RoutePolicy'));
const AdminRoutePolicyPage = lazy(() => import('./pages/AdminRoutePolicy'));
const SupplierApplyPage = lazy(() => import('./pages/Supplier/Apply'));
const SupplierChannelPage = lazy(() => import('./pages/Supplier/Channel'));
const PricingSettingsPage = lazy(
  () => import('./pages/Supplier/PricingSettings'),
);
const SupplierDashboardPage = lazy(() => import('./pages/Supplier/Dashboard'));
const SupplierApplication = lazy(
  () => import('./pages/SupplierAdmin/application'),
);
const Suppliers = lazy(() => import('./pages/SupplierAdmin/list'));
const Setup = lazy(() => import('./pages/Setup'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Performance = lazy(() => import('./pages/Performance'));
const DistributorApply = lazy(() => import('./pages/DistributorApply'));
const DistributorCenter = lazy(() => import('./pages/DistributorCenter'));
const DistributorAdmin = lazy(() => import('./pages/DistributorAdmin'));
const InviteRedirect = lazy(() => import('./pages/InviteRedirect'));
const About = lazy(() => import('./pages/About'));
const Rankings = lazy(() => import('./pages/Rankings'));
const UserAgreement = lazy(() => import('./pages/UserAgreement'));
const PrivacyPolicy = lazy(() => import('./pages/PrivacyPolicy'));
const ComputePage = lazy(() => import('./pages/Compute'));

function DynamicOAuth2Callback() {
  const { provider } = useParams();
  return <OAuth2Callback type={provider} />;
}

function App() {
  const location = useLocation();
  const [statusState] = useContext(StatusContext);

  // 获取模型广场权限配置
  const pricingRequireAuth = useMemo(() => {
    const headerNavModulesConfig = statusState?.status?.HeaderNavModules;
    if (headerNavModulesConfig) {
      try {
        const modules = JSON.parse(headerNavModulesConfig);

        // 处理向后兼容性：如果pricing是boolean，默认不需要登录
        if (typeof modules.pricing === 'boolean') {
          return false; // 默认不需要登录鉴权
        }

        // 如果是对象格式，使用requireAuth配置
        return modules.pricing?.requireAuth === true;
      } catch (error) {
        console.error('解析顶栏模块配置失败:', error);
        return false; // 默认不需要登录
      }
    }
    return false; // 默认不需要登录
  }, [statusState?.status?.HeaderNavModules]);

  return (
    <SetupCheck>
      <Suspense fallback={<Loading />} key={location.pathname}>
        <Routes>
          <Route
            path='/'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Home />
              </Suspense>
            }
          />
          <Route
            path='/compute'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <ComputePage />
              </Suspense>
            }
          />
          <Route
            path='/setup'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Setup />
              </Suspense>
            }
          />
          <Route path='/forbidden' element={<Forbidden />} />
          <Route
            path='/console/models'
            element={
              <AdminRoute>
                <ModelPage />
              </AdminRoute>
            }
          />
          <Route
            path='/console/deployment'
            element={
              <AdminRoute>
                <ModelDeploymentPage />
              </AdminRoute>
            }
          />
          <Route
            path='/console/model-heat'
            element={
              <AdminRoute>
                <ModelHeatPage />
              </AdminRoute>
            }
          />
          <Route
            path='/console/subscription'
            element={
              <AdminRoute>
                <Subscription />
              </AdminRoute>
            }
          />
          <Route
            path='/console/channel'
            element={
              <AdminRoute>
                <Channel />
              </AdminRoute>
            }
          />
          <Route
            path='/console/token'
            element={
              <PrivateRoute>
                <Token />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/playground'
            element={
              <PrivateRoute>
                <Playground />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/redemption'
            element={
              <AdminOrDistributorRoute>
                <Redemption />
              </AdminOrDistributorRoute>
            }
          />
          <Route
            path='/console/user'
            element={
              <AdminRoute>
                <User />
              </AdminRoute>
            }
          />
          <Route
            path='/console/supplier-application'
            element={
              <AdminRoute>
                <SupplierApplication />
              </AdminRoute>
            }
          />
          <Route
            path='/console/suppliers'
            element={
              <AdminRoute>
                <Suppliers />
              </AdminRoute>
            }
          />
          <Route
            path='/user/reset'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <PasswordResetConfirm />
              </Suspense>
            }
          />
          <Route
            path='/login'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <AuthRedirect>
                  <LoginForm />
                </AuthRedirect>
              </Suspense>
            }
          />
          <Route
            path='/register'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <AuthRedirect>
                  <RegisterForm />
                </AuthRedirect>
              </Suspense>
            }
          />
          <Route
            path='/r/:aff'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <InviteRedirect />
              </Suspense>
            }
          />
          <Route
            path='/reset'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <PasswordResetForm />
              </Suspense>
            }
          />
          <Route
            path='/oauth/github'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <OAuth2Callback type='github'></OAuth2Callback>
              </Suspense>
            }
          />
          <Route
            path='/oauth/discord'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <OAuth2Callback type='discord'></OAuth2Callback>
              </Suspense>
            }
          />
          <Route
            path='/oauth/oidc'
            element={
              <Suspense fallback={<Loading></Loading>}>
                <OAuth2Callback type='oidc'></OAuth2Callback>
              </Suspense>
            }
          />
          <Route
            path='/oauth/linuxdo'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <OAuth2Callback type='linuxdo'></OAuth2Callback>
              </Suspense>
            }
          />
          <Route
            path='/oauth/:provider'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <DynamicOAuth2Callback />
              </Suspense>
            }
          />
          <Route
            path='/console/setting'
            element={
              <RootRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Setting />
                </Suspense>
              </RootRoute>
            }
          />
          <Route
            path='/real-name/start'
            element={<RealNameVerificationStart />}
          />
          <Route
            path='/real-name/result'
            element={<RealNameVerificationResult />}
          />
          <Route
            path='/console/real-name-verification'
            element={
              <PrivateRoute>
                <RealNameVerification />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/personal'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <PersonalSetting />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/invoice'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <InvoicePage />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/invoice-admin'
            element={
              <AdminRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <InvoiceAdminPage />
                </Suspense>
              </AdminRoute>
            }
          />
          <Route
            path='/console/settlement-export'
            element={
              <AdminRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <SettlementExportPage />
                </Suspense>
              </AdminRoute>
            }
          />
          <Route
            path='/console/route-policy'
            element={
              <PrivateRoute>
                <RoutePolicyPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/admin/route-policy'
            element={
              <AdminRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <AdminRoutePolicyPage />
                </Suspense>
              </AdminRoute>
            }
          />
          <Route
            path='/console/seedance/material'
            element={
              <PrivateRoute>
                <SeedanceMaterial />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/supplier'
            element={
              <PrivateRoute>
                <SupplierApplyPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/supplier/apply'
            element={
              <PrivateRoute>
                <SupplierApplyPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/supplier/channel'
            element={
              <PrivateRoute>
                <SupplierChannelPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/supplier/pricing-settings'
            element={
              <PrivateRoute>
                <PricingSettingsPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/supplier/dashboard'
            element={
              <PrivateRoute>
                <SupplierDashboardPage />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/distributor/apply'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <DistributorApply />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/distributor/center'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <DistributorCenter />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/distributor/admin'
            element={
              <AdminRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <DistributorAdmin />
                </Suspense>
              </AdminRoute>
            }
          />
          <Route
            path='/console/topup'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <TopUp />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/log'
            element={
              <PrivateRoute>
                <Log />
              </PrivateRoute>
            }
          />
          <Route
            path='/console/aliyun-guardrail'
            element={
              <AdminRoute>
                <AliyunGuardrail />
              </AdminRoute>
            }
          />
          <Route
            path='/console'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Dashboard />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/performance'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Performance />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/midjourney'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Midjourney />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/console/task'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Task />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route
            path='/pricing'
            element={
              pricingRequireAuth ? (
                <PrivateRoute>
                  <Suspense
                    fallback={<Loading></Loading>}
                    key={location.pathname}
                  >
                    <Pricing />
                  </Suspense>
                </PrivateRoute>
              ) : (
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Pricing />
                </Suspense>
              )
            }
          />
          <Route
            path='/rankings'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Rankings />
              </Suspense>
            }
          />
          <Route
            path='/about'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <About />
              </Suspense>
            }
          />
          <Route
            path='/user-agreement'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <UserAgreement />
              </Suspense>
            }
          />
          <Route
            path='/privacy-policy'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <PrivacyPolicy />
              </Suspense>
            }
          />
          <Route
            path='/console/chat/:id?'
            element={
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Chat />
              </Suspense>
            }
          />
          {/* 方便使用chat2link直接跳转聊天... */}
          <Route
            path='/chat2link'
            element={
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Chat2Link />
                </Suspense>
              </PrivateRoute>
            }
          />
          <Route path='*' element={<NotFound />} />
        </Routes>
      </Suspense>
    </SetupCheck>
  );
}

export default App;
