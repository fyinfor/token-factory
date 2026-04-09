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
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
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

const { Text } = Typography;

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);
  const isChinese = i18n.language.startsWith('zh');

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
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='w-full overflow-x-hidden'>
          {/* Banner 部分 */}
          <div className='home-banner-bg w-full min-h-[400px] md:min-h-[500px]'>
            <div className='h-full px-4 py-16 md:py-20'>
              {/* 居中内容区 */}
              <div className='flex flex-col items-center justify-center text-center max-w-3xl mx-auto my-32'>
                <h1
                  className={`text-4xl md:text-5xl lg:text-6xl font-medium text-semi-color-text-0 leading-tight mb-4 ${isChinese ? 'tracking-wide' : ''}`}
                >
                  {t('定制化大模型服务，一站式统一入口')}
                </h1>
                <p className='text-sm md:text-base text-semi-color-text-2 mb-8'>
                  {t('按需定制，更优')}
                  <span className='font-semibold text-semi-color-text-0'>{t('价格')}</span>
                  {t('，更稳的')}
                  <span className='font-semibold text-semi-color-text-0'>{t('可靠')}</span>
                  {t('，开箱即用')}
                </p>

                {/* 操作按钮 */}
                <div className='flex flex-row gap-3 justify-center items-center'>
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
              </div>

              {/* 模型列表区域 */}
              <HomeModelList />
            </div>
          </div>

          {/* 功能卡片区域 */}
          <div className='w-full px-4 py-16 md:py-20'>
            <div className='max-w-6xl mx-auto'>
              <div className='grid grid-cols-1 md:grid-cols-2 gap-6'>
                {/* 卡片1: 一个 API，调用任意模型 */}
                <div className='bg-semi-color-bg-1 rounded-2xl p-8 border border-semi-color-border hover:shadow-lg transition-shadow'>
                  <div className='mb-6 h-24 flex items-center justify-center'>
                    <svg className='w-20 h-20' viewBox='0 0 200 200' fill='none' xmlns='http://www.w3.org/2000/svg'>
                      <circle cx='40' cy='40' r='8' fill='#c7d2fe' />
                      <circle cx='160' cy='40' r='8' fill='#bfdbfe' />
                      <line x1='50' y1='100' x2='150' y2='50' stroke='#c7d2fe' strokeWidth='4' strokeLinecap='round' />
                      <line x1='50' y1='100' x2='150' y2='150' stroke='#c7d2fe' strokeWidth='4' strokeLinecap='round' />
                    </svg>
                  </div>
                  <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                    {t('一个 API，调用任意模型')}
                  </h3>
                  <p className='text-sm text-semi-color-text-2 mb-4'>
                    {t('通过统一接口同主流模型，OpenAI 兼容 SDK 可直接使用。')}
                  </p>
                  {/* <Link to='/model' className='text-sm text-semi-color-primary font-medium hover:underline'>
                    {t('浏览全部')} →
                  </Link> */}
                </div>

                {/* 卡片2: 更高可用性 */}
                <div className='bg-semi-color-bg-1 rounded-2xl p-8 border border-semi-color-border hover:shadow-lg transition-shadow'>
                  <div className='mb-6 h-24 flex items-center justify-center'>
                    <svg className='w-20 h-20' viewBox='0 0 200 200' fill='none' xmlns='http://www.w3.org/2000/svg'>
                      <rect x='40' y='120' width='120' height='4' rx='2' fill='#86efac' />
                      <path d='M 60 80 Q 100 20 140 80' stroke='#d1d5db' strokeWidth='3' strokeDasharray='8 4' strokeLinecap='round' fill='none' />
                      <text x='20' y='100' fontSize='14' fill='#6366f1' fontFamily='monospace'>anthropic/claude-opus-4.6</text>
                    </svg>
                  </div>
                  <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                    {t('更高可用性')}
                  </h3>
                  <p className='text-sm text-semi-color-text-2 mb-4'>
                    {t('分布式集群设施承载库，单路故障时可自动切换其他应路。')}
                  </p>
                  {/* <Link to='/channel' className='text-sm text-semi-color-primary font-medium hover:underline'>
                    {t('了解更多')} →
                  </Link> */}
                </div>

                {/* 卡片3: 灵活计费方式 */}
                <div className='bg-semi-color-bg-1 rounded-2xl p-8 border border-semi-color-border hover:shadow-lg transition-shadow'>
                  <div className='mb-6 h-24 flex items-center justify-center'>
                    <svg className='w-20 h-20' viewBox='0 0 200 200' fill='none' xmlns='http://www.w3.org/2000/svg'>
                      <rect x='30' y='80' width='40' height='80' rx='4' fill='#e0e7ff' />
                      <rect x='80' y='50' width='40' height='110' rx='4' fill='#c7d2fe' />
                      <rect x='130' y='100' width='40' height='60' rx='4' fill='#a5b4fc' />
                    </svg>
                  </div>
                  <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                    {t('灵活计费方式')}
                  </h3>
                  <p className='text-sm text-semi-color-text-2 mb-4'>
                    {t('按需付费无需订阅，支持多种计费模式和用户分组定价。')}
                  </p>
                  {/* <Link to='/pricing' className='text-sm text-semi-color-primary font-medium hover:underline'>
                    {t('查看定价')} →
                  </Link> */}
                </div>

                {/* 卡片4: 完整使用日志 */}
                <div className='bg-semi-color-bg-1 rounded-2xl p-8 border border-semi-color-border hover:shadow-lg transition-shadow'>
                  <div className='mb-6 h-24 flex items-center justify-center'>
                    <svg className='w-20 h-20' viewBox='0 0 200 200' fill='none' xmlns='http://www.w3.org/2000/svg'>
                      <rect x='40' y='40' width='120' height='120' rx='8' stroke='#d1d5db' strokeWidth='3' fill='none' />
                      <line x1='60' y1='70' x2='140' y2='70' stroke='#a5b4fc' strokeWidth='3' strokeLinecap='round' />
                      <line x1='60' y1='100' x2='120' y2='100' stroke='#c7d2fe' strokeWidth='3' strokeLinecap='round' />
                      <line x1='60' y1='130' x2='140' y2='130' stroke='#c7d2fe' strokeWidth='3' strokeLinecap='round' />
                    </svg>
                  </div>
                  <h3 className='text-xl font-semibold text-semi-color-text-0 mb-3'>
                    {t('完整使用日志')}
                  </h3>
                  <p className='text-sm text-semi-color-text-2 mb-4'>
                    {t('实时监控每次调用，详细记录请求和响应便于调试分析。')}
                  </p>
                  {/* <Link to='/log' className='text-sm text-semi-color-primary font-medium hover:underline'>
                    {t('查看文档')} →
                  </Link> */}
                </div>
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
    </div>
  );
};

export default Home;
