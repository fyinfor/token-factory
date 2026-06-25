import React, { useEffect, useMemo, useState } from 'react';
import { Modal, Button, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { ExternalLink } from 'lucide-react';
import { copy, showSuccess } from '../../../helpers';

const { Text } = Typography;

const FIELD_TEXT_STYLE = {
  color: 'var(--semi-color-text-0)',
  margin: 0,
  fontWeight: 600,
};

const NOTICE_BOX_STYLE = {
  width: '100%',
  padding: '10px 12px',
  borderRadius: 8,
  background: '#FFF9E6',
  border: '1px solid #FFE58F',
  color: 'var(--semi-color-text-0)',
  fontSize: 14,
  lineHeight: 1.6,
  textAlign: 'center',
  margin: 0,
  fontFamily: 'inherit',
};

const ROW_ACTIONS_STYLE = {
  display: 'flex',
  flexWrap: 'nowrap',
  alignItems: 'center',
  gap: 8,
  width: '100%',
  overflowX: 'auto',
  WebkitOverflowScrolling: 'touch',
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
          <p style={NOTICE_BOX_STYLE}>
            {t(
              '请选择相同区块链网络转账，选错将导致资产无法到账。区块链交易需区块确认，预计 10 分钟内到账，请勿重复提交。',
            )}
          </p>

          <div style={{ width: '100%', textAlign: 'center' }}>
            {ucoinResult.network && (
              <p style={FIELD_TEXT_STYLE}>
                {t('网络')}：{ucoinResult.network}
              </p>
            )}
            {(ucoinResult.currency || ucoinResult.coin) && (
              <p style={FIELD_TEXT_STYLE}>
                {t('币种')}：{ucoinResult.currency || ucoinResult.coin}
              </p>
            )}
            {ucoinResult.min_topup != null && (
              <p style={FIELD_TEXT_STYLE}>
                {t('最小充值金额')}：{ucoinResult.min_topup}
              </p>
            )}
          </div>

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
              <div style={ROW_ACTIONS_STYLE}>
                <Button
                  size='small'
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.binance.url, '_blank')
                  }
                  style={{ flexShrink: 0 }}
                >
                  {WALLET_LINKS.binance.name}
                </Button>
                <Button
                  size='small'
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.metamask.url, '_blank')
                  }
                  style={{ flexShrink: 0 }}
                >
                  {WALLET_LINKS.metamask.name}
                </Button>
                <Button
                  size='small'
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(WALLET_LINKS.eoraptor.url, '_blank')
                  }
                  style={{ flexShrink: 0 }}
                >
                  {WALLET_LINKS.eoraptor.name}
                </Button>
              </div>
              <div
                style={{
                  ...ROW_ACTIONS_STYLE,
                  marginTop: 12,
                  paddingTop: 12,
                  borderTop: '1px solid var(--semi-color-border)',
                }}
              >
                <Text strong style={{ flexShrink: 0, whiteSpace: 'nowrap' }}>
                  {t('交易推荐')}
                </Text>
                <Button
                  size='small'
                  theme='outline'
                  type='tertiary'
                  icon={<ExternalLink size={14} />}
                  onClick={() =>
                    window.open(EXCHANGE_LINKS.eoraptor.url, '_blank')
                  }
                  style={{ flexShrink: 0 }}
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
