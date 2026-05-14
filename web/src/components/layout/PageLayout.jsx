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

import HeaderBar from './headerbar';
import { Layout, Button, Space } from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';
import SiderBar from './SiderBar';
import App from '../../App';
import FooterBar from './Footer';
import { ToastContainer } from 'react-toastify';
import React, { useContext, useEffect, useState } from 'react';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { useTranslation } from 'react-i18next';
import { API, getLogo, showError, setStatusData } from '../../helpers';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useLocation } from 'react-router-dom';
import {
  normalizeLanguage,
  pickPrimaryNavigatorLanguage,
  isSupportedUiLanguage,
  I18N_ANON_LANG_INITIALIZED_KEY,
  I18N_BROWSER_LANG_MISMATCH_PROMPT_KEY,
  I18N_BROWSER_LANG_BANNER_PENDING_KEY,
} from '../../i18n/language';
const { Sider, Content, Header } = Layout;

const PageLayout = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const [collapsed, , setCollapsed] = useSidebarCollapsed();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [browserLangBanner, setBrowserLangBanner] = useState(null);
  const { i18n, t } = useTranslation();
  const location = useLocation();

  const cardProPages = [
    '/',
    '/console/channel',
    '/console/log',
    '/console/redemption',
    '/console/user',
    '/console/token',
    '/console/midjourney',
    '/console/task',
    '/console/models',
    '/console/supplier/apply',
    '/console/distributor/apply',
    '/console/supplier/channel',
    '/console/supplier/models',
    '/console/supplier-application',
    '/console/suppliers',
    '/console/playground',
    '/pricing',
  ];

  const shouldHideFooter = cardProPages.includes(location.pathname);

  const pricingNaturalScroll = location.pathname === '/pricing';

  const shouldInnerPadding =
    location.pathname.includes('/console') &&
    !location.pathname.startsWith('/console/chat') &&
    location.pathname !== '/console/playground';

  const isConsoleRoute = location.pathname.startsWith('/console');
  const showSider = isConsoleRoute && (!isMobile || drawerOpen);

  useEffect(() => {
    if (isMobile && drawerOpen && collapsed) {
      setCollapsed(false);
    }
  }, [isMobile, drawerOpen, collapsed, setCollapsed]);

  const loadUser = () => {
    let user = localStorage.getItem('user');
    if (user) {
      let data = JSON.parse(user);
      userDispatch({ type: 'login', payload: data });
    }
  };

  /** 与 localStorage 中的 status 派生字段一致（须在 setStatusData 之后调用） */
  const syncDocumentBrandingFromStorage = () => {
    // 仅用显式缓存：避免用 getSystemName() 的占位文案抢在 /api/status 之前改标题，
    // 也避免旧缓存里的默认名（如 TokenFactory）闪一下。
    const systemName = localStorage.getItem('system_name');
    if (systemName) {
      document.title = systemName;
    }
    const logo = getLogo();
    if (logo) {
      const linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  };

  const loadStatus = async () => {
    try {
      const res = await API.get('/api/status');
      const { success, data } = res.data;
      if (success) {
        statusDispatch({ type: 'set', payload: data });
        setStatusData(data);
        syncDocumentBrandingFromStorage();
      } else {
        showError('Unable to connect to server');
        syncDocumentBrandingFromStorage();
      }
    } catch (error) {
      showError('Failed to load status');
      syncDocumentBrandingFromStorage();
    }
  };

  useEffect(() => {
    loadUser();
    loadStatus().catch(console.error);
  }, []);

  useEffect(() => {
    const status = statusState?.status;
    if (!status) return;

    let userLang;
    if (userState?.user?.setting) {
      try {
        const settings = JSON.parse(userState.user.setting);
        userLang = normalizeLanguage(settings.language);
      } catch (e) {
        /* ignore */
      }
    }

    if (userLang && isSupportedUiLanguage(userLang)) {
      localStorage.setItem('i18nextLng', userLang);
      if (userLang !== normalizeLanguage(i18n.language)) {
        i18n.changeLanguage(userLang);
      }
      setBrowserLangBanner(null);
      return;
    }

    if (localStorage.getItem(I18N_ANON_LANG_INITIALIZED_KEY) === '1') {
      const saved = normalizeLanguage(localStorage.getItem('i18nextLng') || '');
      if (
        saved &&
        isSupportedUiLanguage(saved) &&
        saved !== normalizeLanguage(i18n.language)
      ) {
        i18n.changeLanguage(saved);
      }
      if (localStorage.getItem(I18N_BROWSER_LANG_MISMATCH_PROMPT_KEY) !== 'done') {
        try {
          const raw = sessionStorage.getItem(I18N_BROWSER_LANG_BANNER_PENDING_KEY);
          if (raw) {
            const parsed = JSON.parse(raw);
            if (parsed?.browserLang && parsed?.siteLang) {
              setBrowserLangBanner(parsed);
            }
          }
        } catch (e) {
          /* ignore */
        }
      }
      return;
    }

    const siteDefaultRaw = normalizeLanguage(
      status.default_site_language || 'zh-CN',
    );
    const siteDefault = isSupportedUiLanguage(siteDefaultRaw)
      ? siteDefaultRaw
      : 'zh-CN';

    localStorage.setItem(I18N_ANON_LANG_INITIALIZED_KEY, '1');
    localStorage.setItem('i18nextLng', siteDefault);
    if (normalizeLanguage(i18n.language) !== siteDefault) {
      i18n.changeLanguage(siteDefault);
    }

    if (localStorage.getItem(I18N_BROWSER_LANG_MISMATCH_PROMPT_KEY) === 'done') {
      return;
    }

    const browserLang = pickPrimaryNavigatorLanguage();
    if (browserLang && browserLang !== siteDefault) {
      const payload = { browserLang, siteLang: siteDefault };
      try {
        sessionStorage.setItem(
          I18N_BROWSER_LANG_BANNER_PENDING_KEY,
          JSON.stringify(payload),
        );
      } catch (e) {
        /* ignore */
      }
      setBrowserLangBanner(payload);
    }
  }, [statusState?.status, userState?.user?.setting, userState?.user?.id, i18n]);

  const resolveLangLabel = (code) => {
    try {
      const loc = normalizeLanguage(i18n.language) || 'zh-CN';
      const dn = new Intl.DisplayNames([loc], { type: 'language' });
      return dn.of(code) || code;
    } catch (e) {
      return code;
    }
  };

  const dismissBrowserLangBanner = (mode) => {
    if (mode === 'browser' && browserLangBanner) {
      const { browserLang } = browserLangBanner;
      i18n.changeLanguage(browserLang);
      localStorage.setItem('i18nextLng', browserLang);
    }
    localStorage.setItem(I18N_BROWSER_LANG_MISMATCH_PROMPT_KEY, 'done');
    try {
      sessionStorage.removeItem(I18N_BROWSER_LANG_BANNER_PENDING_KEY);
    } catch (e) {
      /* ignore */
    }
    setBrowserLangBanner(null);
  };

  return (
    <Layout
      className='app-layout'
      style={{
        display: 'flex',
        flexDirection: 'column',
        overflow: isMobile ? 'visible' : 'hidden',
      }}
    >
      {browserLangBanner ? (
        <div
          style={{
            position: 'fixed',
            top: isMobile ? 68 : 72,
            right: isMobile ? 8 : 16,
            left: isMobile ? 8 : 'auto',
            zIndex: 99,
            maxWidth: isMobile ? 'none' : 300,
            width: isMobile ? 'auto' : 300,
            padding: 1,
            borderRadius: 14,
            background:
              'linear-gradient(135deg, var(--semi-color-primary), #8b5cf6 45%, #22d3ee)',
            boxShadow:
              '0 4px 16px rgba(0, 0, 0, 0.12), 0 0 0 1px rgba(255,255,255,0.06) inset',
            pointerEvents: 'auto',
          }}
        >
          <div
            style={{
              position: 'relative',
              borderRadius: 13,
              background: 'var(--semi-color-bg-2)',
              padding: '10px 32px 10px 10px',
            }}
          >
            <Button
              aria-label={t('取消')}
              theme='borderless'
              type='tertiary'
              size='small'
              icon={<IconClose />}
              onClick={() => dismissBrowserLangBanner('site')}
              style={{
                position: 'absolute',
                top: 2,
                right: 2,
                minWidth: 28,
              }}
            />
            <p
              style={{
                margin: '0 0 8px 0',
                fontSize: 12,
                lineHeight: 1.45,
                color: 'var(--semi-color-text-1)',
              }}
            >
              {t('检测到浏览器语言与站点默认不同描述', {
                browser: resolveLangLabel(browserLangBanner.browserLang),
                site: resolveLangLabel(browserLangBanner.siteLang),
              })}
            </p>
            <Space spacing='tight' wrap>
              <Button
                type='primary'
                size='small'
                onClick={() => dismissBrowserLangBanner('browser')}
              >
                {t('确认')}
              </Button>
              <Button size='small' onClick={() => dismissBrowserLangBanner('site')}>
                {t('取消')}
              </Button>
            </Space>
          </div>
        </div>
      ) : null}
      <Header
        style={{
          padding: 0,
          height: 'auto',
          lineHeight: 'normal',
          position: 'fixed',
          width: '100%',
          top: 0,
          zIndex: 100,
        }}
      >
        <HeaderBar
          onMobileMenuToggle={() => setDrawerOpen((prev) => !prev)}
          drawerOpen={drawerOpen}
        />
      </Header>
      <Layout
        style={{
          overflow: isMobile ? 'visible' : 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {showSider && (
          <Sider
            className='app-sider'
            style={{
              position: 'fixed',
              left: 0,
              top: '64px',
              zIndex: 99,
              border: 'none',
              paddingRight: '0',
              width: 'var(--sidebar-current-width)',
            }}
          >
            <SiderBar
              onNavigate={() => {
                if (isMobile) setDrawerOpen(false);
              }}
            />
          </Sider>
        )}
        <Layout
          style={{
            marginLeft: isMobile
              ? '0'
              : showSider
                ? 'var(--sidebar-current-width)'
                : '0',
            flex: '1 1 auto',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Content
            style={{
              flex: '1 0 auto',
              overflowY: isMobile ? 'visible' : 'hidden',
              WebkitOverflowScrolling: 'touch',
              padding: shouldInnerPadding ? (isMobile ? '5px' : '24px') : '0',
              position: 'relative',
            }}
          >
            <App />
          </Content>
          {!shouldHideFooter && (
            <Layout.Footer
              style={{
                flex: '0 0 auto',
                width: '100%',
              }}
            >
              <FooterBar />
            </Layout.Footer>
          )}
        </Layout>
      </Layout>
      <ToastContainer />
    </Layout>
  );
};

export default PageLayout;
