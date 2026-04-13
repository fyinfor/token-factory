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
import React, { useContext } from 'react';
import { Avatar, Button, Card, Tag, Typography, Divider, Empty } from '@douyinfe/semi-ui';
import { IllustrationNoAccess, IllustrationNoAccessDark } from '@douyinfe/semi-illustrations';
import { Store, Package, Layers, TrendingUp } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { USER_ROLES } from '../../constants/user.constants';
import { stringToColor } from '../../helpers';
import ProvidersPage from '../../components/table/providers';

const ProviderInfoHeader = ({ t, userState }) => {
  const username = userState?.user?.username || '';
  const avatarText = username.length > 0 ? username.slice(0, 2).toUpperCase() : 'NA';

  return (
    <Card
      className='!rounded-2xl overflow-hidden mb-4'
      cover={
        <div
          className='relative h-32'
          style={{
            '--palette-primary-channel': '30 64 175',
            backgroundImage: `linear-gradient(135deg, rgba(var(--palette-primary-channel) / 90%), rgba(99, 102, 241, 0.85)), url('/cover-4.webp')`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            backgroundRepeat: 'no-repeat',
          }}
        >
          <div className='relative z-10 h-full flex flex-col justify-end p-6'>
            <div className='flex items-center'>
              <div className='flex items-stretch gap-3 sm:gap-4 flex-1 min-w-0'>
                <Avatar size='large' color={stringToColor(username)}>
                  {avatarText}
                </Avatar>
                <div className='flex-1 min-w-0 flex flex-col justify-between'>
                  <div
                    className='text-3xl font-bold truncate'
                    style={{ color: 'white' }}
                  >
                    {username}
                  </div>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Tag size='large' shape='circle' style={{ color: 'white' }}>
                      <Store size={12} className='mr-1 inline' />
                      {t('供应商')}
                    </Tag>
                    <Tag size='large' shape='circle' style={{ color: 'white' }}>
                      ID: {userState?.user?.id}
                    </Tag>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      }
    >
      <div className='flex items-center justify-between gap-6'>
        <div className='text-sm text-semi-color-text-2'>
          {t('在这里管理您的供应商信息、渠道与模型配置。')}
        </div>
        <div className='hidden lg:block flex-shrink-0'>
          <Card
            size='small'
            className='!rounded-xl'
            bodyStyle={{ padding: '12px 16px' }}
          >
            <div className='flex items-center gap-4'>
              <div className='flex items-center gap-2'>
                <Layers size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('渠道')}
                </Typography.Text>
                <Typography.Text size='small' type='tertiary' strong>
                  —
                </Typography.Text>
              </div>
              <Divider layout='vertical' />
              <div className='flex items-center gap-2'>
                <Package size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('模型')}
                </Typography.Text>
                <Typography.Text size='small' type='tertiary' strong>
                  —
                </Typography.Text>
              </div>
              <Divider layout='vertical' />
              <div className='flex items-center gap-2'>
                <TrendingUp size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('调用量')}
                </Typography.Text>
                <Typography.Text size='small' type='tertiary' strong>
                  —
                </Typography.Text>
              </div>
            </div>
          </Card>
        </div>
      </div>

      <div className='lg:hidden mt-2'>
        <Card
          size='small'
          className='!rounded-xl'
          bodyStyle={{ padding: '12px 16px' }}
        >
          <div className='space-y-3'>
            <div className='flex items-center justify-between'>
              <div className='flex items-center gap-2'>
                <Layers size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('渠道')}
                </Typography.Text>
              </div>
              <Typography.Text size='small' type='tertiary' strong>
                —
              </Typography.Text>
            </div>
            <Divider margin='8px' />
            <div className='flex items-center justify-between'>
              <div className='flex items-center gap-2'>
                <Package size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('模型')}
                </Typography.Text>
              </div>
              <Typography.Text size='small' type='tertiary' strong>
                —
              </Typography.Text>
            </div>
            <Divider margin='8px' />
            <div className='flex items-center justify-between'>
              <div className='flex items-center gap-2'>
                <TrendingUp size={16} />
                <Typography.Text size='small' type='tertiary'>
                  {t('调用量')}
                </Typography.Text>
              </div>
              <Typography.Text size='small' type='tertiary' strong>
                —
              </Typography.Text>
            </div>
          </div>
        </Card>
      </div>
    </Card>
  );
};

const ProviderPage = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const userRole = userState?.user?.role ?? 0;
  const isProvider = userRole >= USER_ROLES.DISTRIBUTOR;

  return (
    <div className='mt-[60px] px-2'>
      <div className='w-full max-w-7xl mx-auto'>
        <ProviderInfoHeader t={t} userState={userState} />
      </div>
      {!isProvider ? (
        <div className='flex items-center justify-center' style={{ minHeight: 'calc(100vh - 360px)' }}>
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={<IllustrationNoAccessDark style={{ width: 200, height: 200 }} />}
            layout="horizontal"
            title={t('成为供应商')}
            description={t('您当前还不是供应商，申请成为供应商后即可管理您的供应商信息。')}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              style={{ fontWeight: 500 }}
            >
              {t('申请成为供应商')}
            </Button>
          </Empty>
        </div>
      ) : (
        <ProvidersPage />
      )}
    </div>
  );
};

export default ProviderPage;
