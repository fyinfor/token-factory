/*
Copyright (C) 2025 QuantumNous

支付宝 / 微信支付官方品牌 Logo（静态资源）
*/

import React from 'react';

const ALIPAY_LOGO_SRC = '/payment-brands/alipay.png';
const WECHAT_PAY_LOGO_SRC = '/payment-brands/wechatpay.png';
const ERP_LOGO_SRC = '/payment-brands/erp.png';

function PayBrandImage({ src, size, className, alt }) {
  return (
    <img
      src={src}
      alt={alt}
      width={size}
      height={size}
      className={className}
      aria-hidden={!alt}
      draggable={false}
      style={{
        display: 'block',
        width: size,
        height: size,
        objectFit: 'contain',
      }}
    />
  );
}

export function AlipayPayLogo({ size = 24, className }) {
  return (
    <PayBrandImage
      src={ALIPAY_LOGO_SRC}
      size={size}
      className={className}
      alt=''
    />
  );
}

export function WeChatPayLogo({ size = 24, className }) {
  return (
    <PayBrandImage
      src={WECHAT_PAY_LOGO_SRC}
      size={size}
      className={className}
      alt=''
    />
  );
}

export function ErpPayLogo({ size = 24, className }) {
  return (
    <PayBrandImage
      src={ERP_LOGO_SRC}
      size={size}
      className={className}
      alt=''
    />
  );
}

export function isLogoOnlyPayMethod(type) {
  return type === 'alipay' || type === 'wxpay';
}
