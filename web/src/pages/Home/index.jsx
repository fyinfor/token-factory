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

import React, { useContext, useEffect, useState } from 'react';
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
import HomeModelList from '../../components/home/HomeModelList';
import HomeLandingHeroCopy from '../../components/home/HomeLandingHeroCopy';
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
} from '@lobehub/icons';
import FooterBar from '../../components/layout/Footer';

const { Text } = Typography;

/** 首页功能卡片：与首张相同的左图右文布局，顺序对应 public/home-card-1..4.png */
const HOME_FEATURE_CARDS = [
  {
    image: '/home-card-1.png',
    titleKey: '一个 API，调用任意模型',
    descKey: '通过统一接口同主流模型，OpenAI 兼容 SDK 可直接使用。',
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
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

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
          onClose={() => setNoticeVisible(false)}
          isMobile={isMobile}
        />
        {homePageContentLoaded && homePageContent === '' ? (
          <div className='w-full'>
            {/* Banner 部分 */}
            <div className='home-banner-bg w-full min-h-[400px] md:min-h-[500px]'>
              <div className='h-full px-4 pt-16 md:pt-20'>
                {/* 居中内容区 */}
                <div className='my-16'>
                  <HomeLandingHeroCopy />

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
                              ? t('登录后可提交申请，成为分销商获得邀请分成')
                              : t('提交资料申请成为分销商，邀请好友获得充值分成')}
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
                              : t('申请成为分销商')}
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
            <div className='w-full px-4 py-16 md:py-20'>
              <div className='max-w-6xl mx-auto'>
                <div className='grid grid-cols-1 md:grid-cols-2 gap-6'>
                  {HOME_FEATURE_CARDS.map((card) => (
                    <div
                      key={card.image}
                      className='group flex flex-col overflow-hidden rounded-2xl border border-semi-color-border bg-semi-color-bg-1 shadow-sm transition-shadow hover:shadow-lg md:flex-row md:items-stretch md:gap-10'
                    >
                      <div className='relative flex shrink-0 items-center justify-center bg-semi-color-bg-1 px-5 pb-2 pt-6 md:w-[42%] md:max-w-[300px] md:p-6 md:pb-6'>
                        <div className='relative aspect-[4/3] w-full max-w-[320px] overflow-hidden rounded-xl bg-semi-color-bg-0 shadow-md ring-1 ring-semi-color-border md:max-w-none'>
                          <img
                            src={card.image}
                            alt=''
                            className='h-full w-full object-cover object-center transition-transform duration-500 ease-out group-hover:scale-[1.02]'
                            decoding='async'
                          />
                        </div>
                      </div>
                      <div className='flex flex-1 flex-col justify-center px-6 pb-6 pt-5 md:px-8 md:py-8 md:pl-0'>
                        <h3 className='text-xl font-semibold leading-snug text-semi-color-text-0 md:text-[1.35rem]'>
                          {t(card.titleKey)}
                        </h3>
                        <p className='mt-3 text-sm leading-relaxed text-semi-color-text-2'>
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
        <FooterBar />
      </div>
    </>
  );
};

export default Home;
