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

import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  Button,
  Typography,
  Input,
  ScrollList,
  ScrollItem,
} from '@douyinfe/semi-ui';
import {
  API,
  showError,
  copy,
  showSuccess,
  userIsDistributorUser,
} from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { UserContext } from '../../context/User';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconFile,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  OPEN_NOTIFICATION_CENTER_EVENT,
  acknowledgeAnnouncementPopup,
  getAnnouncementPopupKey,
  getLegacyNoticePopupKey,
  isAnnouncementPopupAcknowledged,
} from '../../hooks/common/useNotifications';
import HomeModelList from '../../components/home/HomeModelList';
import HomeHeroCarousel from '../../components/home/HomeHeroCarousel';
import HomeFooterCertificates from '../../components/home/HomeFooterCertificates';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
  XiaomiMiMo,
} from '@lobehub/icons';
import FooterBar from '../../components/layout/Footer';

const { Text } = Typography;

/** 首页功能卡片：与首张相同的左图右文布局，顺序对应 public/home-card-1..4.png */
const HOME_FEATURE_CARDS = [
  {
    image: '/home-card-1.png',
    titleKey: '一个 API key，调用任意模型',
    descKey: '通过统一接口调用主流模型，OpenAI 兼容 SDK 可直接使用。',
  },
  {
    image: '/home-card-2.png',
    titleKey: '大模型部署定制服务',
    descKey:
      '支持构建高效稳定的 Token 工厂，实现大规模生成能力的标准化与可控化',
  },
  {
    image: '/home-card-3.png',
    titleKey: '灵活计费方式',
    descKey: '按需付费无需订阅，支持多种计费模式和用户分组定价。',
  },
  {
    image: '/home-card-4.png',
    titleKey: '完整使用日志',
    descKey: '实时监控每次调用，详细记录请求和响应便于调试分析。',
  },
];

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [noticePopupKey, setNoticePopupKey] = useState('');
  const [legacyNoticeContent, setLegacyNoticeContent] = useState('');
  const noticeCheckedRef = useRef(false);
  const latestAnnouncement = statusState?.status?.announcements?.[0];
  const latestAnnouncementPopupKey = latestAnnouncement
    ? getAnnouncementPopupKey(latestAnnouncement)
    : '';
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  let u = userState?.user;
  if (!u) {
    try {
      const raw = localStorage.getItem('user');
      if (raw) u = JSON.parse(raw);
    } catch {
      u = null;
    }
  }
  const userRole = u?.role ?? null;
  const showDistributorRecruit = !userIsDistributorUser(u);

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      if (!statusState?.status) {
        return;
      }

      if (latestAnnouncementPopupKey) {
        if (
          noticeCheckedRef.current === latestAnnouncementPopupKey ||
          isAnnouncementPopupAcknowledged(latestAnnouncementPopupKey)
        ) {
          return;
        }
        noticeCheckedRef.current = latestAnnouncementPopupKey;
        setLegacyNoticeContent('');
        setNoticePopupKey(latestAnnouncementPopupKey);
        setNoticeVisible(true);
        return;
      }

      if (noticeCheckedRef.current === 'legacy') {
        return;
      }
      noticeCheckedRef.current = 'legacy';
      try {
        const res = await API.get('/api/notice');
        const { success, data } = res.data;
        if (success && data && data.trim() !== '') {
          const legacyPopupKey = getLegacyNoticePopupKey(data);
          if (isAnnouncementPopupAcknowledged(legacyPopupKey)) {
            return;
          }
          setLegacyNoticeContent(data);
          setNoticePopupKey(legacyPopupKey);
          setNoticeVisible(true);
        }
      } catch (error) {
        console.error('获取公告失败:', error);
      }
    };

    checkNoticeAndShow();
  }, [latestAnnouncementPopupKey, statusState?.status]);

  const acknowledgeNotice = () => {
    acknowledgeAnnouncementPopup(noticePopupKey);
    setNoticeVisible(false);
  };

  const handleViewMoreNotices = () => {
    acknowledgeNotice();
    window.dispatchEvent(new Event(OPEN_NOTIFICATION_CENTER_EVENT));
  };

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  return (
    <>
      <style>{`
        .home-scroll-container {
          scrollbar-width: none;
        }
        .home-scroll-container::-webkit-scrollbar {
          display: none;
        }
      `}</style>
      <div className='w-full h-[100dvh] overflow-y-auto home-scroll-container'>
        <NoticeModal
          visible={noticeVisible}
          fallbackContent={legacyNoticeContent}
          onClose={acknowledgeNotice}
          onViewMore={handleViewMoreNotices}
        />
        {homePageContentLoaded && homePageContent === '' ? (
          <div className='w-full'>
            {/* Banner 部分 */}
            <div className='home-banner-bg w-full'>
              <div className='h-full px-4 pt-20 md:pt-24 pb-2'>
                <div className='mb-6'>
                  <HomeHeroCarousel
                    enabled={statusState?.status?.home_hero_carousel_enabled}
                    rawSlides={statusState?.status?.home_hero_carousel_slides}
                    intervalSec={
                      statusState?.status?.home_hero_carousel_interval_sec
                    }
                    aspectRatio={
                      statusState?.status?.home_hero_carousel_aspect_ratio
                    }
                  />

                  {/* 操作按钮 */}
                  {/* <div className='flex flex-row gap-3 justify-center items-center mb-8'>
                    <Link to='/about'>
                      <Button
                        theme='solid'
                        type='primary'
                        size={isMobile ? 'default' : 'large'}
                        className='!rounded-md px-8'
                        style={{ fontWeight: 500 }}
                      >
                        {t('立即获取专属方案')}
                      </Button>
                    </Link>
                  </div>

                  {showDistributorRecruit && (
                    <div className='home-distributor-recruit-card w-full max-w-xl mx-auto mb-6 rounded-2xl border border-semi-color-border bg-semi-color-bg-1/90 backdrop-blur-sm px-4 py-4 text-left'>
                      <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3'>
                        <div>
                          <div className='font-semibold text-semi-color-text-0'>
                            {t('分销伙伴招募')}
                          </div>
                          <div className='text-sm text-semi-color-text-2 mt-1'>
                            {userRole == null
                              ? t('登录后可提交申请，成为代理获得邀请分成')
                              : t('提交资料申请成为代理，邀请好友获得充值分成')}
                          </div>
                        </div>
                        <Link
                          to={
                            userRole == null
                              ? '/login?redirect=' +
                                encodeURIComponent('/console/distributor/apply')
                              : '/console/distributor/apply'
                          }
                        >
                          <Button
                            theme='solid'
                            type='primary'
                            className='home-distributor-cta-btn !rounded-lg shrink-0'
                          >
                            {userRole == null
                              ? t('登录并申请')
                              : t('申请成为代理')}
                          </Button>
                        </Link>
                      </div>
                    </div>
                  )} */}
                </div>

                {/* 广告展示位 */}
                {/* <div className='w-full max-w-[800px] mx-auto mb-8'>
                  <Link to='/console/supplier/apply'>
                    <div className='relative w-full h-[140px] md:h-[280px] overflow-hidden cursor-pointer transition-transform duration-300 hover:scale-[1.01]'>
                      <img
                        src='/ad.jpg'
                        alt='Advertisement'
                        className='w-full h-full object-cover rounded-[10px]'
                      />
                    </div>
                  </Link>
                </div> */}

                {/* 模型列表区域 */}
                <HomeModelList />
              </div>
            </div>

            {/* 功能卡片区域 */}
            <div className='relative w-full overflow-hidden bg-[linear-gradient(135deg,rgba(239,246,255,0.55)_0%,rgba(255,255,255,0.15)_46%,rgba(245,243,255,0.55)_100%)] px-4 py-16 dark:bg-none md:py-20'>
              <div
                aria-hidden='true'
                className='pointer-events-none absolute -left-12 top-4 h-80 w-80 rounded-full bg-[rgba(56,189,248,0.2)] blur-[90px] dark:bg-[rgba(56,189,248,0.08)]'
              />
              <div
                aria-hidden='true'
                className='pointer-events-none absolute -right-8 bottom-0 h-80 w-80 rounded-full bg-[rgba(167,139,250,0.18)] blur-[90px] dark:bg-[rgba(167,139,250,0.07)]'
              />
              <div
                aria-hidden='true'
                className='pointer-events-none absolute left-[44%] top-[38%] h-56 w-56 rounded-full bg-[rgba(103,232,249,0.13)] blur-[80px] dark:opacity-40'
              />
              <div className='relative mx-auto max-w-6xl'>
                <div className='grid grid-cols-1 gap-6 md:grid-cols-2'>
                  {HOME_FEATURE_CARDS.map((card) => (
                    <div
                      key={card.image}
                      className='group relative isolate flex flex-col overflow-hidden rounded-[30px] border-0 bg-[rgba(255,255,255,0.3)] shadow-[0_20px_60px_-32px_rgba(84,110,145,0.3),inset_0_0_0_1px_rgba(255,255,255,0.68),inset_0_1px_0_rgba(255,255,255,0.95)] backdrop-blur-[28px] backdrop-saturate-[180%] transition-[background-color,box-shadow] duration-500 ease-out hover:bg-[rgba(255,255,255,0.4)] hover:shadow-[0_28px_70px_-34px_rgba(84,110,145,0.36),inset_0_0_0_1px_rgba(255,255,255,0.78),inset_0_1px_0_rgba(255,255,255,1)] dark:bg-[rgba(15,23,42,0.35)] dark:shadow-[0_20px_55px_-32px_rgba(15,23,42,0.45),inset_0_0_0_1px_rgba(255,255,255,0.1),inset_0_1px_0_rgba(255,255,255,0.08)] dark:hover:bg-[rgba(15,23,42,0.45)] md:flex-row md:items-stretch md:gap-8'
                    >
                      <div
                        aria-hidden='true'
                        className='pointer-events-none absolute -left-20 -top-20 h-48 w-48 rounded-full bg-[rgba(125,211,252,0.2)] blur-3xl transition-opacity duration-500 group-hover:opacity-80 dark:bg-[rgba(56,189,248,0.1)]'
                      />
                      <div
                        aria-hidden='true'
                        className='pointer-events-none absolute inset-x-8 top-0 h-px bg-[linear-gradient(to_right,transparent,rgba(255,255,255,0.9),transparent)] dark:bg-[linear-gradient(to_right,transparent,rgba(255,255,255,0.3),transparent)]'
                      />
                      <div
                        aria-hidden='true'
                        className='pointer-events-none absolute right-5 top-4 h-20 w-36 rounded-full bg-[rgba(255,255,255,0.22)] blur-2xl dark:opacity-20'
                      />
                      <div className='relative flex shrink-0 items-stretch px-4 pb-1 pt-4 md:w-[43%] md:max-w-[300px] md:p-4'>
                        <div className='relative aspect-[4/3] w-full overflow-hidden rounded-[20px] md:aspect-auto md:min-h-[170px]'>
                          <img
                            src={card.image}
                            alt=''
                            className='absolute inset-0 h-full w-full rounded-[20px] object-cover object-center'
                            decoding='async'
                          />
                        </div>
                      </div>
                      <div className='relative flex flex-1 flex-col justify-center px-5 pb-5 pt-4 md:py-6 md:pl-1 md:pr-6'>
                        <h3 className='text-xl font-semibold leading-snug tracking-[-0.015em] text-semi-color-text-0 md:text-[1.35rem]'>
                          {t(card.titleKey)}
                        </h3>
                        <p className='mt-3 text-sm leading-7 text-semi-color-text-2'>
                          {t(card.descKey)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className='overflow-x-hidden w-full'>
            {homePageContent.startsWith('https://') ? (
              <iframe
                src={homePageContent}
                className='w-full h-screen border-none'
              />
            ) : (
              <div
                className='mt-[60px]'
                dangerouslySetInnerHTML={{ __html: homePageContent }}
              />
            )}
          </div>
        )}
        <HomeFooterCertificates
          enabled={statusState?.status?.home_footer_certificates_enabled}
          rawCertificates={statusState?.status?.home_footer_certificates}
        />
        <FooterBar />
      </div>
    </>
  );
};

export default Home;
