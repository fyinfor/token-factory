import React from 'react';
import { Modal, Typography, Space, Button, Tooltip } from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe } from 'react-icons/si';
import { CreditCard, Wallet } from 'lucide-react';

const { Text } = Typography;

const PaymentMethodSelectModal = ({
  t,
  visible,
  onCancel,
  topUpCount,
  renderTopUpCount,
  payMethods = [],
  enableOnlineTopUp,
  enableStripeTopUp,
  onSelect,
  paymentLoading,
  payWay,
}) => {
  const epayMethods = payMethods.filter((m) => m.type !== 'waffo');

  const renderPayIcon = (payMethod) => {
    if (payMethod.type === 'alipay') {
      return <SiAlipay size={20} color='#1677FF' />;
    }
    if (payMethod.type === 'wxpay') {
      return <SiWechat size={20} color='#07C160' />;
    }
    if (payMethod.type === 'stripe') {
      return <SiStripe size={20} color='#635BFF' />;
    }
    return (
      <CreditCard
        size={20}
        color={payMethod.color || 'var(--semi-color-text-2)'}
      />
    );
  };

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <Wallet className='mr-2' size={18} />
          {t('选择支付方式')}
        </div>
      }
      visible={visible}
      onCancel={onCancel}
      footer={null}
      maskClosable
      size='small'
      centered
    >
      <div className='space-y-4 pb-2'>
        <Text type='tertiary' size='small'>
          {t('充值数量')}：{renderTopUpCount(topUpCount)}
        </Text>
        <Space vertical spacing='medium' style={{ width: '100%' }}>
          {epayMethods.map((payMethod) => {
            const minTopupVal = Number(payMethod.min_topup) || 0;
            const isStripe = payMethod.type === 'stripe';
            const disabled =
              (!enableOnlineTopUp && !isStripe) ||
              (!enableStripeTopUp && isStripe) ||
              minTopupVal > Number(topUpCount || 0);

            const buttonEl = (
              <Button
                key={payMethod.type}
                theme='outline'
                type='tertiary'
                block
                size='large'
                onClick={() => onSelect(payMethod.type)}
                disabled={disabled}
                loading={paymentLoading && payWay === payMethod.type}
                icon={renderPayIcon(payMethod)}
                className='!rounded-xl !justify-start !px-4 !py-3'
              >
                {payMethod.name}
              </Button>
            );

            if (disabled && minTopupVal > Number(topUpCount || 0)) {
              return (
                <Tooltip
                  key={payMethod.type}
                  content={
                    t('此支付方式最低充值金额为') + ' ' + minTopupVal
                  }
                >
                  <span style={{ display: 'block', width: '100%' }}>
                    {buttonEl}
                  </span>
                </Tooltip>
              );
            }

            return buttonEl;
          })}
        </Space>
      </div>
    </Modal>
  );
};

export default PaymentMethodSelectModal;
