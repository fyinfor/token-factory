/*
Copyright (C) 2025 QuantumNous

支付宝 / 微信支付 / Antom 官方品牌 Logo（静态资源）
*/

import React from 'react';

const ALIPAY_LOGO_SRC = '/payment-brands/alipay.png';
const WECHAT_PAY_LOGO_SRC = '/payment-brands/wechatpay.png';
const ERP_LOGO_SRC = '/payment-brands/erp.png';
const ANTOM_LOGO_SRC = '/payment-brands/antom.svg';

function PayBrandImage({ src, size, width, height, className, alt }) {
  const w = width ?? size;
  const h = height ?? size;
  return (
    <img
      src={src}
      alt={alt}
      width={w}
      height={h}
      className={className}
      aria-hidden={!alt}
      draggable={false}
      style={{
        display: 'block',
        width: w,
        height: h,
        objectFit: 'contain',
        borderRadius: 6,
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

export function AntomPayLogo({ size = 24, className }) {
  return (
    <PayBrandImage
      src={ANTOM_LOGO_SRC}
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
