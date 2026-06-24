import React, { useEffect, useMemo, useState } from 'react';
import { Modal, Button, Typography, Space, Banner } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { ExternalLink } from 'lucide-react';
import { copy, showSuccess } from '../../../helpers';

const { Text } = Typography;

const WALLET_LINKS = {
  binance: {
    name: 'Binance Wallet',
    url: 'https://www.binance.com/web3wallet',
  },
  metamask: {
    name: 'MetaMask',
    url: 'https://metamask.io/download/',
  },
};

function detectBrowserWallets() {
  if (typeof window === 'undefined') {
    return { hasMetaMask: false, hasBinance: false };
  }
  const eth = window.ethereum;
  const hasMetaMask = Boolean(eth?.isMetaMask);
  const hasBinance = Boolean(
    window.BinanceChain || eth?.isBinance || eth?.isBinanceWallet,
  );
  return { hasMetaMask, hasBinance };
}

const UcoinPayResultModal = ({ t, visible, onCancel, ucoinResult }) => {
  const [walletState, setWalletState] = useState({
    hasMetaMask: false,
    hasBinance: false,
  });

  useEffect(() => {
    if (!visible) {
      return;
    }
    setWalletState(detectBrowserWallets());
  }, [visible]);

  const showWalletGuide = useMemo(() => {
    return !walletState.hasMetaMask && !walletState.hasBinance;
  }, [walletState]);

  const handleCopyAddress = () => {
    copy(ucoinResult?.address || '');
    showSuccess(t('已复制到剪贴板'));
  };

  return (
    <Modal
      title={t('U币支付')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      centered
    >
      {ucoinResult && (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 12,
          }}
        >
          <p style={{ textAlign: 'center', margin: 0 }}>
            {t('请向以下地址转账完成充值，到账后将自动入账。')}
          </p>
          {ucoinResult.coin && (
            <p style={{ margin: 0 }}>
              {t('币种')}：{ucoinResult.coin}
            </p>
          )}
          <p style={{ margin: 0 }}>
            {t('充值数量')}：{ucoinResult.amount}
          </p>

          <div style={{ background: '#fff', padding: 12, borderRadius: 8 }}>
            <QRCodeSVG value={ucoinResult.address || ''} size={180} />
          </div>

          <div
            style={{
              width: '100%',
              wordBreak: 'break-all',
              textAlign: 'center',
              fontFamily: 'monospace',
            }}
          >
            {ucoinResult.address}
          </div>

          <Button theme='solid' type='primary' onClick={handleCopyAddress}>
            {t('复制地址')}
          </Button>

          {showWalletGuide ? (
            <div
              style={{
                width: '100%',
                marginTop: 4,
                padding: 12,
                borderRadius: 8,
                background: 'var(--semi-color-fill-0)',
              }}
            >
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                {t('推荐钱包')}
              </Text>
              <Text
                type='tertiary'
                size='small'
                style={{ display: 'block', marginBottom: 10 }}
              >
                {t('未检测到浏览器钱包插件，可安装后使用，也可直接用下方二维码转账。')}
              </Text>
              <Space wrap>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() => window.open(WALLET_LINKS.binance.url, '_blank')}
                >
                  {WALLET_LINKS.binance.name}
                </Button>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() => window.open(WALLET_LINKS.metamask.url, '_blank')}
                >
                  {WALLET_LINKS.metamask.name}
                </Button>
              </Space>
              <Banner
                type='info'
                closeIcon={null}
                style={{ marginTop: 10 }}
                description={t('或使用交易所/手机钱包扫码转账')}
              />
            </div>
          ) : (
            <Text type='tertiary' size='small' style={{ textAlign: 'center' }}>
              {t('或使用交易所/手机钱包扫码转账')}
            </Text>
          )}

          <p
            style={{
              color: 'var(--semi-color-text-2)',
              fontSize: 12,
              margin: 0,
            }}
          >
            {t('订单号')}：{ucoinResult.order_id}
          </p>
        </div>
      )}
    </Modal>
  );
};

export default UcoinPayResultModal;
