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

import { getPayMethodDisplayName } from '../../../helpers';
import { Modal, Typography, Card } from '@douyinfe/semi-ui';
import { SiStripe } from 'react-icons/si';
import { CreditCard } from 'lucide-react';
import { AlipayPayLogo, AntomPayLogo, WeChatPayLogo } from '../PaymentBrandIcons';

const { Text } = Typography;

const PaymentConfirmModal = ({
  t,
  open,
  onlineTopUp,
  handleCancel,
  confirmLoading,
  topUpCount,
  renderTopUpCount,
  payWay,
  payMethods,
  creditDisplay = '',
}) => {
  return (
    <Modal
      title={
        <div className='flex items-center'>
          <CreditCard className='mr-2' size={18} />
          {t('充值确认')}
        </div>
      }
      visible={open}
      onOk={onlineTopUp}
      onCancel={handleCancel}
      maskClosable={false}
      size='small'
      centered
      confirmLoading={confirmLoading}
    >
      <div className='space-y-4'>
        <Card className='!rounded-xl !border-0 bg-slate-50 dark:bg-slate-800'>
          <div className='space-y-3'>
            <div className='flex justify-between items-center'>
              <Text strong className='text-slate-700 dark:text-slate-200'>
                {t('充值金额')}：
              </Text>
              <Text className='text-slate-900 dark:text-slate-100'>
                {renderTopUpCount(topUpCount)}
              </Text>
            </div>
            {creditDisplay && (
              <div className='flex justify-between items-center'>
                <Text strong className='text-slate-700 dark:text-slate-200'>
                  {t('到账积分')}：
                </Text>
                <Text className='text-slate-900 dark:text-slate-100'>
                  {creditDisplay}
                </Text>
              </div>
            )}
            <div className='flex justify-between items-center'>
              <Text strong className='text-slate-700 dark:text-slate-200'>
                {t('支付方式')}：
              </Text>
              <div className='flex items-center'>
                {(() => {
                  const payMethod = payMethods.find(
                    (method) => method.type === payWay,
                  );
                  if (payMethod) {
                    return (
                      <>
                        {payMethod.type === 'alipay' ? (
                          <span className='mr-2 inline-flex'>
                            <AlipayPayLogo size={18} />
                          </span>
                        ) : payMethod.type === 'antom' ? (
                          <span className='mr-2 inline-flex'>
                            <AntomPayLogo height={16} />
                          </span>
                        ) : payMethod.type === 'wxpay' ? (
                          <span className='mr-2 inline-flex'>
                            <WeChatPayLogo size={18} />
                          </span>
                        ) : payMethod.type === 'stripe' ? (
                          <SiStripe
                            className='mr-2'
                            size={16}
                            color='#635BFF'
                          />
                        ) : (
                          <CreditCard
                            className='mr-2'
                            size={16}
                            color={
                              payMethod.color || 'var(--semi-color-text-2)'
                            }
                          />
                        )}
                        <Text className='text-slate-900 dark:text-slate-100'>
                          {getPayMethodDisplayName(payMethod, t)}
                        </Text>
                      </>
                    );
                  } else {
                    // 默认充值方式
                    if (payWay === 'alipay') {
                      return (
                        <>
                          <span className='mr-2 inline-flex'>
                            <AlipayPayLogo size={18} />
                          </span>
                          <Text className='text-slate-900 dark:text-slate-100'>
                            {t('支付宝')}
                          </Text>
                        </>
                      );
                    } else if (payWay === 'antom') {
                      return (
                        <>
                          <span className='mr-2 inline-flex'>
                            <AntomPayLogo height={16} />
                          </span>
                          <Text className='text-slate-900 dark:text-slate-100'>
                            {t('Antom 收银台')}
                          </Text>
                        </>
                      );
                    } else if (payWay === 'stripe') {
                      return (
                        <>
                          <SiStripe
                            className='mr-2'
                            size={16}
                            color='#635BFF'
                          />
                          <Text className='text-slate-900 dark:text-slate-100'>
                            Stripe
                          </Text>
                        </>
                      );
                    } else {
                      return (
                        <>
                          <span className='mr-2 inline-flex'>
                            <WeChatPayLogo size={18} />
                          </span>
                          <Text className='text-slate-900 dark:text-slate-100'>
                            {t('微信')}
                          </Text>
                        </>
                      );
                    }
                  }
                })()}
              </div>
            </div>
          </div>
        </Card>
      </div>
    </Modal>
  );
};

export default PaymentConfirmModal;
