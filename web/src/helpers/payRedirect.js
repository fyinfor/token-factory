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

/**
 * 支付跳转相关工具（微信 deep link、移动端 H5、唤起等）。
 */

/** 检测是否为移动设备（手机/平板）。 */
export function isMobileDevice() {
  if (typeof navigator === 'undefined') {
    return false;
  }
  return /android|iphone|ipad|ipod|mobile/i.test(navigator.userAgent || '');
}

/** 返回后端统一下单用的客户端场景标识。 */
export function getClientDevice() {
  return isMobileDevice() ? 'mobile' : 'pc';
}

/** 检测是否在微信内置浏览器中打开。 */
export function isWeChatInAppBrowser() {
  if (typeof navigator === 'undefined') {
    return false;
  }
  return /MicroMessenger/i.test(navigator.userAgent || '');
}

/**
 * 判断 URL 是否为微信 App 支付 deep link。
 * @param {unknown} url 支付跳转地址
 * @returns {boolean}
 */
export function isWeixinPayDeepLink(url) {
  if (typeof url !== 'string') {
    return false;
  }
  const low = url.trim().toLowerCase();
  return low.startsWith('weixin://') || low.startsWith('weixinpay://');
}

/** 判断是否为可在浏览器直接打开的 HTTP(S) 支付链接。 */
export function isHttpPayURL(url) {
  if (typeof url !== 'string') {
    return false;
  }
  const low = url.trim().toLowerCase();
  return low.startsWith('http://') || low.startsWith('https://');
}

/**
 * 解析支付链接的前端拉起方式。
 * @param {unknown} url 支付跳转地址
 * @returns {'form'|'qr'|'app'|'h5'} form=表单 POST；qr=PC 扫码；app=移动端 scheme；h5=移动端 H5 跳转
 */
export function resolvePayRedirectMode(url) {
  if (typeof url !== 'string' || !url.trim()) {
    return 'form';
  }
  const trimmed = url.trim();
  if (isHttpPayURL(trimmed)) {
    return isMobileDevice() ? 'h5' : 'form';
  }
  if (isWeixinPayDeepLink(trimmed)) {
    // 手机未开通 H5 时网关仅返回 weixin://，与 PC 一样展示二维码供微信扫码，避免浏览器打开 scheme 失败。
    return 'qr';
  }
  return 'form';
}

/**
 * 在移动端通过 hidden iframe 尝试唤起微信（避免部分浏览器把 weixin:// 写进地址栏）。
 * @param {string} url weixin:// 或 weixinpay:// 链接
 */
export function openWeixinPayDeepLink(url) {
  if (!url || typeof url !== 'string') {
    return;
  }
  const iframe = document.createElement('iframe');
  iframe.style.cssText =
    'display:none;width:0;height:0;border:0;position:absolute;left:-9999px';
  iframe.src = url;
  document.body.appendChild(iframe);
  window.setTimeout(() => {
    try {
      document.body.removeChild(iframe);
    } catch {
      /* ignore */
    }
  }, 3000);
}

/**
 * 移动端跳转微信 H5 收银台（https://wx.tenpay.com/...）。
 * @param {string} url H5 支付链接
 */
export function openMobileH5Pay(url) {
  if (!url || typeof url !== 'string') {
    return;
  }
  window.location.assign(url);
}

/** @deprecated 使用 resolvePayRedirectMode */
export function resolveWeixinEpayRedirectMode(url) {
  const mode = resolvePayRedirectMode(url);
  if (mode === 'h5') {
    return 'form';
  }
  return mode;
}
