import { getPayMethodDisplayName } from '../../../helpers';
import { Modal, Typography, Space, Button, Tooltip } from '@douyinfe/semi-ui';
import { SiStripe } from 'react-icons/si';
import { CreditCard, Wallet } from 'lucide-react';
import {
  AlipayPayLogo,
  WeChatPayLogo,
} from '../PaymentBrandIcons';

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
  activePaymentKey = '',
}) => {
  const epayMethods = payMethods.filter((m) => m.type !== 'waffo');

  const renderPayIcon = (payMethod) => {
    if (payMethod.type === 'alipay') {
      return <AlipayPayLogo size={22} />;
    }
    if (payMethod.type === 'wxpay') {
      return <WeChatPayLogo size={22} />;
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
          {t('充值金额')}：{renderTopUpCount(topUpCount)}
        </Text>
        <Space vertical spacing='medium' style={{ width: '100%' }}>
          {epayMethods.map((payMethod) => {
            const minTopupVal = Number(payMethod.min_topup) || 0;
            const maxTopupVal = Number(payMethod.max_topup) || 0;
            const isStripe = payMethod.type === 'stripe';
            const countVal = Number(topUpCount || 0);
            const belowMin = minTopupVal > countVal;
            const aboveMax = maxTopupVal > 0 && maxTopupVal < countVal;
            const disabled =
              (!enableOnlineTopUp && !isStripe) ||
              (!enableStripeTopUp && isStripe) ||
              belowMin ||
              aboveMax;

            const buttonEl = (
              <Button
                key={payMethod.type}
                theme='outline'
                type='tertiary'
                block
                size='large'
                onClick={() => onSelect(payMethod.type)}
                disabled={disabled}
                loading={activePaymentKey === payMethod.type}
                icon={renderPayIcon(payMethod)}
                className='!rounded-xl !justify-start !px-4 !py-3'
              >
                {getPayMethodDisplayName(payMethod, t)}
              </Button>
            );

            if (disabled && (belowMin || aboveMax)) {
              return (
                <Tooltip
                  key={payMethod.type}
                  content={
                    belowMin
                      ? t('此支付方式最低充值金额为') + ' ' + minTopupVal
                      : t('充值数量不能大于') + maxTopupVal
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
