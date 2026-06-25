import React, { useEffect, useMemo, useState } from 'react';
import { Modal, Button, Typography, Space, Banner } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { ExternalLink } from 'lucide-react';
import { copy, showSuccess } from '../../../helpers';

const { Text } = Typography;

const RED_TEXT_STYLE = {
  color: 'var(--semi-color-danger)',
  margin: 0,
  fontWeight: 600,
};

const WALLET_LINKS = {
  binance: {
    name: 'Binance Wallet',
    url: 'https://www.binance.com/web3wallet',
  },
  metamask: {
    name: 'MetaMask',
    url: 'https://metamask.io/download/',
  },
  eoraptor: {
    name: 'Eoraptor Wallet',
    url: 'https://ts-apk.s3.ap-east-1.amazonaws.com/ts-apk/EORAPTORWallet.apk',
  },
};

const EXCHANGE_LINKS = {
  eoraptor: {
    name: 'Eoraptor',
    url: 'https://web.eoraptor.org/',
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

          <div style={{ width: '100%', textAlign: 'center' }}>
            {ucoinResult.network && (
              <p style={RED_TEXT_STYLE}>
                {t('网络')}：{ucoinResult.network}
              </p>
            )}
            {(ucoinResult.currency || ucoinResult.coin) && (
              <p style={RED_TEXT_STYLE}>
                {t('币种')}：{ucoinResult.currency || ucoinResult.coin}
              </p>
            )}
            {ucoinResult.min_topup != null && (
              <p style={RED_TEXT_STYLE}>
                {t('最小充值金额')}：{ucoinResult.min_topup}
              </p>
            )}
          </div>

          <Banner
            type='warning'
            closeIcon={null}
            style={{ width: '100%' }}
            description={t(
              'USDT 转账到账后，系统自动入账可能存在约 5 分钟延迟，请勿重复转账，请耐心等待。',
            )}
          />

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
                {t(
                  '未检测到浏览器钱包插件，可安装后使用，也可直接用上方二维码转账。',
                )}
              </Text>
              <Space wrap>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.binance.url, '_blank')
                  }
                >
                  {WALLET_LINKS.binance.name}
                </Button>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.metamask.url, '_blank')
                  }
                >
                  {WALLET_LINKS.metamask.name}
                </Button>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.eoraptor.url, '_blank')
                  }
                >
                  {WALLET_LINKS.eoraptor.name}
                </Button>
              </Space>
              <div
                style={{
                  marginTop: 12,
                  paddingTop: 12,
                  borderTop: '1px solid var(--semi-color-border)',
                }}
              >
                <Text
                  strong
                  style={{
                    display: 'block',
                    marginBottom: 8,
                    color: 'var(--semi-color-danger)',
                  }}
                >
                  {t('交易推荐')}
                </Text>
                <Button
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(EXCHANGE_LINKS.eoraptor.url, '_blank')
                  }
                >
                  {EXCHANGE_LINKS.eoraptor.name}
                </Button>
              </div>
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
