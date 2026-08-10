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

import React, { useContext, useMemo } from 'react';
import { useHomeBannerModelFocus } from './use-home-banner-model-focus';
import {
  Input,
  ImagePreview,
  Button,
  Collapsible,
  Select,
} from '@douyinfe/semi-ui';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { IconSearch } from '@douyinfe/semi-icons';
import PricingVendors from '../table/model-pricing/filter/PricingVendors';
import PricingProviderType from '../table/model-pricing/filter/PricingProviderType';
import PricingQuotaTypes from '../table/model-pricing/filter/PricingQuotaTypes';
import PricingTags from '../table/model-pricing/filter/PricingTags';
import PricingEndpointTypes from '../table/model-pricing/filter/PricingEndpointTypes';
import PricingCardView from '../table/model-pricing/view/card/PricingCardView';
import ModelDetailSideSheet from '../table/model-pricing/modal/ModelDetailSideSheet';
import { useModelPricingData } from '../../hooks/model-pricing/useModelPricingData';
import { usePricingFilterCounts } from '../../hooks/model-pricing/usePricingFilterCounts';
import { StatusContext } from '../../context/Status';
import { UserContext } from '../../context/User';
import { LIVE_HOT_FILTER } from '../table/model-pricing/utils/modelHeat';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import HomeModelListSkeleton from './HomeModelListSkeleton';

const EMPTY_SELECTED_ROW_KEYS = [];
const ignoreSelectedRows = () => {};

const HomeModelList = () => {
  const isMobile = useIsMobile();
  const pricingData = useModelPricingData({
    defaultSortKey: 'hot',
    mergeChannelsByModel: true,
  });
  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);
  const showInitialSkeleton = useMinimumLoadingTime(pricingData.loading, 450);

  const headerNavHomeConfig = useMemo(() => {
    try {
      const config = statusState?.status?.HeaderNavModules;
      if (!config) return null;
      const modules = JSON.parse(config);
      const home = modules?.home;
      if (typeof home === 'object' && home !== null) {
        return home;
      }
    } catch {}
    return null;
  }, [statusState?.status?.HeaderNavModules]);

  const blurPricing = useMemo(() => {
    if (userState?.user) return false;
    return !!headerNavHomeConfig?.blurPricing;
  }, [headerNavHomeConfig, userState?.user]);

  const showCostPrice = useMemo(
    () => !!headerNavHomeConfig?.showCostPrice,
    [headerNavHomeConfig],
  );

  const {
    quotaTypeModels,
    endpointTypeModels,
    vendorModels,
    tagModels,
    supplierTypeModels,
  } = usePricingFilterCounts({
    models: pricingData.models,
    filterGroup: pricingData.filterGroup,
    filterQuotaType: pricingData.filterQuotaType,
    filterEndpointType: pricingData.filterEndpointType,
    filterVendor: pricingData.filterVendor,
    filterTag: pricingData.filterTag,
    filterSupplier: pricingData.filterSupplier,
    filterSupplierType: pricingData.filterSupplierType,
    searchValue: pricingData.searchValue,
    hotChannelScoreMap: pricingData.hotChannelScoreMap,
  });

  React.useEffect(() => {
    pricingData.setPageSize(40);
  }, []);

  useHomeBannerModelFocus({
    loading: pricingData.loading,
    setSearchValue: pricingData.setSearchValue,
  });

  const handleResetFilters = () => {
    pricingData.setSearchValue('');
    pricingData.setFilterVendor('all');
    pricingData.setFilterQuotaType('all');
    pricingData.setFilterTag('all');
    pricingData.setFilterEndpointType('all');
    pricingData.setFilterSupplierType?.('all');
    pricingData.setFilterSupplier && pricingData.setFilterSupplier('all');
    pricingData.setSortKey && pricingData.setSortKey('hot');
    pricingData.setCurrentPage(1);
  };

  const handleFilterVendorChange = (vendor) => {
    pricingData.setFilterVendor(vendor);
    if (vendor === 'all') {
      pricingData.setSortKey?.('hot');
    } else {
      pricingData.setSortKey?.('discount');
    }
  };

  const handleFilterTagChange = (tag) => {
    pricingData.setFilterTag(tag);
    if (tag === LIVE_HOT_FILTER) {
      pricingData.setSortKey?.('hot');
    }
  };

  const sortSelectValue =
    !pricingData.sortKey || pricingData.sortKey === 'default'
      ? 'hot'
      : pricingData.sortKey;

  const sortOptions = [
    // { value: 'default', label: pricingData.t('默认') },
    { value: 'hot', label: pricingData.t('热门') },
    { value: 'price', label: pricingData.t('价格') },
    { value: 'discount', label: pricingData.t('折扣率') },
    { value: 'supplier_grade', label: pricingData.t('供应商等级') },
    { value: 'latency', label: pricingData.t('时延') },
  ];

  if (showInitialSkeleton) {
    return <HomeModelListSkeleton label={pricingData.t('加载中...')} />;
  }

  return (
    <div id='home-models' className='w-full home-model-list'>
      <style>{`
        .home-model-cards-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
          gap: 0.75rem;
          align-items: stretch;
        }
        .home-model-card {
          position: relative;
          isolation: isolate;
          overflow: hidden;
          width: 100%;
          min-width: 0;
          min-height: 278px;
          border-radius: 10px !important;
          --home-card-accent-a: rgba(14, 165, 233, 0.07);
          --home-card-accent-b: rgba(99, 102, 241, 0.055);
          --home-card-accent-c: rgba(45, 212, 191, 0.06);
          border-color: rgba(59, 130, 246, 0.1) !important;
          background:
            linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 252, 0.82)),
            radial-gradient(circle at 12% 0%, var(--home-card-accent-a), transparent 42%),
            radial-gradient(circle at 98% 18%, var(--home-card-accent-b), transparent 38%),
            radial-gradient(circle at 70% 112%, var(--home-card-accent-c), transparent 42%),
            linear-gradient(135deg, rgba(248, 250, 252, 0.82), rgba(255, 255, 255, 0.9) 48%, rgba(248, 250, 252, 0.78)),
            var(--semi-color-bg-2);
          box-shadow: none;
          transition:
            background 0.3s ease,
            border-color 0.3s ease,
            box-shadow 0.3s ease,
            transform 0.3s ease;
          --model-price-glass-bg: rgba(255, 255, 255, 0.7);
          --model-price-glass-head-bg: rgba(255, 255, 255, 0.52);
          --model-price-glass-row-bg: rgba(255, 255, 255, 0.34);
          --model-price-glass-border: rgba(59, 130, 246, 0.18);
          --model-price-glass-line: rgba(59, 130, 246, 0.12);
          --model-price-glass-highlight: rgba(255, 255, 255, 0.78);
          --model-price-glass-head-text: rgba(30, 41, 59, 0.66);
          --model-price-glass-text: #1e293b;
          --model-price-glass-price: #2563eb;
        }
        .home-model-card:nth-child(4n + 2) {
          --home-card-accent-a: rgba(168, 85, 247, 0.06);
          --home-card-accent-b: rgba(14, 165, 233, 0.07);
          --home-card-accent-c: rgba(236, 72, 153, 0.05);
        }
        .home-model-card:nth-child(4n + 3) {
          --home-card-accent-a: rgba(45, 212, 191, 0.07);
          --home-card-accent-b: rgba(59, 130, 246, 0.06);
          --home-card-accent-c: rgba(132, 204, 22, 0.045);
        }
        .home-model-card:nth-child(4n + 4) {
          --home-card-accent-a: rgba(99, 102, 241, 0.06);
          --home-card-accent-b: rgba(236, 72, 153, 0.05);
          --home-card-accent-c: rgba(34, 211, 238, 0.06);
        }
        .home-model-card:hover {
          border-color: rgba(37, 99, 235, 0.34) !important;
          background:
            linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(239, 246, 255, 0.9)),
            radial-gradient(circle at 12% 0%, rgba(37, 99, 235, 0.1), transparent 42%),
            radial-gradient(circle at 98% 18%, rgba(37, 99, 235, 0.08), transparent 38%),
            radial-gradient(circle at 70% 112%, rgba(37, 99, 235, 0.07), transparent 42%),
            var(--semi-color-bg-2);
          box-shadow: 0 14px 34px rgba(37, 99, 235, 0.14) !important;
          transform: translateY(-2px);
        }
        .home-model-card .semi-card-body {
          position: relative;
          z-index: 1;
        }
        .home-model-card-title {
          color: var(--semi-color-text-0);
          letter-spacing: 0;
        }
        .home-model-title-meta {
          display: flex;
          align-items: center;
          gap: 0.375rem;
          min-width: 0;
          margin-top: 0.45rem;
          overflow: hidden;
          white-space: nowrap;
        }
        .home-model-meta-chip,
        .home-model-suffix-chip {
          max-width: 118px;
          min-width: 0;
        }
        .home-model-meta-separator {
          color: var(--semi-color-text-3);
          font-size: 12px;
          line-height: 1;
        }
        .home-model-hot-pill {
          position: relative;
          display: inline-flex;
          flex-shrink: 0;
          align-items: center;
          justify-content: center;
          height: 24px;
          padding: 0 10px;
          overflow: hidden;
          border: 1px solid rgba(251, 191, 36, 0.36);
          border-radius: 999px;
          background: linear-gradient(135deg, #fbbf24, #f97316 48%, #ef4444);
          box-shadow: 0 8px 20px rgba(249, 115, 22, 0.2);
          color: #fff;
          font-size: 12px;
          font-weight: 900;
          letter-spacing: 0;
          line-height: 24px;
          white-space: nowrap;
        }
        .home-model-hot-pill::before {
          content: "";
          position: absolute;
          inset: -45% -80%;
          background: linear-gradient(115deg, transparent 38%, rgba(255, 255, 255, 0.86) 50%, transparent 62%);
          opacity: 0;
          transform: translateX(-70%) rotate(8deg);
          animation: home-model-hot-pill-sheen 2.8s ease-in-out infinite;
        }
        .home-model-hot-pill-text {
          position: relative;
          z-index: 1;
        }
        .home-model-discount-tag {
          flex-shrink: 0;
          height: 24px;
          border: none !important;
          background: linear-gradient(135deg, #f97316, #ef4444) !important;
          color: #fff !important;
          font-size: 12px !important;
          font-weight: 900 !important;
          box-shadow: 0 8px 18px rgba(249, 115, 22, 0.22);
        }
        .home-model-discount-tag .semi-tag-content {
          color: #fff !important;
        }
        .home-model-route-chip {
          height: 23px;
          max-width: 168px;
          min-width: 0;
          border: none;
          border-radius: 999px;
          padding: 0 9px;
          display: inline-flex;
          align-items: center;
          gap: 0.25rem;
          overflow: hidden;
          background: linear-gradient(135deg, #ecfccb, #e0f2fe);
          color: #4d7c0f;
          font-size: 12px;
          font-weight: 800;
          line-height: 23px;
          white-space: nowrap;
        }
        button.home-model-route-chip:not(:disabled) {
          cursor: pointer;
        }
        button.home-model-route-chip:disabled {
          cursor: default;
          opacity: 1;
        }
        button.home-model-route-chip:not(:disabled):hover {
          background: linear-gradient(135deg, #d9f99d, #bae6fd);
        }
        .home-model-route-chip-supplier,
        .home-model-route-chip-suffix {
          min-width: 0;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        .home-model-route-chip-supplier {
          color: #4d7c0f;
        }
        .home-model-route-chip-dot {
          flex-shrink: 0;
          color: var(--semi-color-text-3);
        }
        .home-model-route-chip-suffix {
          flex-shrink: 0;
          color: #0369a1;
        }
        .home-model-price-block {
          color: var(--semi-color-text-0);
        }
        .home-model-description-slot {
          flex: 1 1 0;
          min-height: 0;
          margin-top: 0.75rem;
          overflow: hidden;
        }
        .home-model-description {
          display: -webkit-box;
          overflow: hidden;
          color: var(--semi-color-text-2);
          font-size: 12px;
          line-height: 18px;
          overflow-wrap: anywhere;
          -webkit-box-orient: vertical;
        }
        .home-model-bottom-tags {
          display: flex;
          min-width: 0;
          flex: 1;
          align-items: center;
          gap: 0.375rem;
          overflow: hidden;
          white-space: nowrap;
        }
        .home-model-extra-tag {
          min-width: 0;
        }
        .home-model-more-tag {
          border: 1px solid var(--semi-color-border) !important;
          background: var(--semi-color-fill-0) !important;
          color: var(--semi-color-text-1) !important;
          font-weight: 700 !important;
          cursor: default;
        }
        .home-model-tag-popover {
          display: flex;
          max-width: min(420px, calc(100vw - 32px));
          flex-wrap: wrap;
          gap: 0.375rem;
          padding: 0.25rem;
        }
        .home-model-tag-popover-item {
          max-width: 100%;
        }
        .home-model-detail-btn {
          padding-right: 7px !important;
          border-color: rgba(var(--semi-blue-5), 0.32) !important;
          background: rgba(var(--semi-blue-0), 0.42) !important;
          color: var(--semi-color-primary) !important;
          font-weight: 700 !important;
        }
        .home-model-detail-btn:hover {
          border-color: rgba(var(--semi-blue-5), 0.48) !important;
          background: rgba(var(--semi-blue-0), 0.68) !important;
        }
        .home-model-detail-arrow {
          flex-shrink: 0;
          transition: transform 180ms ease;
        }
        .home-model-detail-btn:hover .home-model-detail-arrow {
          transform: translateX(2px);
        }
        .home-model-card-hot {
          border-color: rgba(59, 130, 246, 0.4) !important;
          background:
            linear-gradient(120deg, rgba(255, 255, 255, 0.88), rgba(248, 250, 252, 0.78)),
            radial-gradient(circle at 10% 10%, rgba(34, 211, 238, 0.34), transparent 32%),
            radial-gradient(circle at 92% 8%, rgba(99, 102, 241, 0.3), transparent 32%),
            radial-gradient(circle at 72% 96%, rgba(236, 72, 153, 0.24), transparent 34%),
            linear-gradient(130deg, rgba(224, 242, 254, 0.96), rgba(237, 233, 254, 0.88) 42%, rgba(204, 251, 241, 0.92) 72%, rgba(252, 231, 243, 0.78)),
            var(--semi-color-bg-2);
          background-size: 100% 100%, 170% 170%, 165% 165%, 170% 170%, 320% 320%, 100% 100%;
          animation: home-model-card-aurora 7.2s ease-in-out infinite;
          box-shadow:
            0 18px 44px rgba(59, 130, 246, 0.16),
            0 0 0 1px rgba(255, 255, 255, 0.72) inset,
            0 10px 28px rgba(15, 23, 42, 0.08) !important;
        }
        .home-model-card-hot:hover {
          border-color: rgba(37, 99, 235, 0.58) !important;
          box-shadow:
            0 22px 48px rgba(59, 130, 246, 0.2),
            0 0 0 1px rgba(255, 255, 255, 0.72) inset,
            0 14px 32px rgba(15, 23, 42, 0.1) !important;
        }
        .home-model-card-hot::before {
          content: "";
          position: absolute;
          inset: -64%;
          z-index: 0;
          pointer-events: none;
          background: conic-gradient(from 90deg, transparent 0deg, rgba(34, 211, 238, 0.28) 48deg, rgba(99, 102, 241, 0.22) 104deg, rgba(236, 72, 153, 0.18) 156deg, transparent 228deg, transparent 360deg);
          opacity: 0.72;
          filter: blur(10px);
          animation: home-model-card-ribbon 8.5s linear infinite;
        }
        .home-model-card-hot::after {
          content: "";
          position: absolute;
          top: -82px;
          right: -70px;
          z-index: 0;
          width: 210px;
          height: 210px;
          border-radius: 999px;
          pointer-events: none;
          background: radial-gradient(circle, rgba(34, 211, 238, 0.34), rgba(99, 102, 241, 0.18) 38%, rgba(236, 72, 153, 0.1) 58%, transparent 68%);
          filter: blur(2px);
          animation: home-model-card-orbit 7.8s ease-in-out infinite;
        }
        html.dark .home-model-card-hot {
          border-color: rgba(34, 211, 238, 0.34) !important;
          background:
            linear-gradient(120deg, rgba(15, 23, 42, 0.97), rgba(17, 24, 39, 0.94)),
            radial-gradient(circle at 10% 10%, rgba(6, 182, 212, 0.34), transparent 34%),
            radial-gradient(circle at 92% 4%, rgba(124, 58, 237, 0.32), transparent 34%),
            radial-gradient(circle at 72% 96%, rgba(244, 63, 94, 0.18), transparent 34%),
            linear-gradient(130deg, rgba(8, 47, 73, 0.78), rgba(49, 46, 129, 0.66) 42%, rgba(15, 118, 110, 0.44) 72%, rgba(131, 24, 67, 0.34)),
            var(--semi-color-bg-2);
          background-size: 100% 100%, 175% 175%, 165% 165%, 170% 170%, 340% 340%, 100% 100%;
          box-shadow:
            0 18px 46px rgba(34, 211, 238, 0.16),
            0 0 0 1px rgba(125, 211, 252, 0.06) inset,
            0 12px 30px rgba(0, 0, 0, 0.36) !important;
        }
        html.dark .home-model-card-hot:hover {
          border-color: rgba(96, 165, 250, 0.5) !important;
          box-shadow:
            0 22px 50px rgba(34, 211, 238, 0.2),
            0 0 0 1px rgba(125, 211, 252, 0.06) inset,
            0 16px 36px rgba(0, 0, 0, 0.4) !important;
        }
        html.dark .home-model-card-hot::after {
          background: radial-gradient(circle, rgba(34, 211, 238, 0.26), rgba(168, 85, 247, 0.18) 42%, rgba(244, 63, 94, 0.08) 58%, transparent 68%);
        }
        html.dark .home-model-card {
          border-color: var(--semi-color-border) !important;
          background: var(--semi-color-bg-2);
          --model-price-glass-bg: rgba(255, 255, 255, 0.07);
          --model-price-glass-head-bg: rgba(255, 255, 255, 0.09);
          --model-price-glass-row-bg: rgba(255, 255, 255, 0.045);
          --model-price-glass-border: rgba(255, 255, 255, 0.12);
          --model-price-glass-line: rgba(255, 255, 255, 0.08);
          --model-price-glass-highlight: rgba(255, 255, 255, 0.1);
          --model-price-glass-head-text: var(--semi-color-text-2);
          --model-price-glass-text: var(--semi-color-text-0);
          --model-price-glass-price: var(--semi-color-primary);
        }
        html.dark .home-model-card:hover {
          border-color: rgba(96, 165, 250, 0.36) !important;
          background:
            linear-gradient(135deg, rgba(15, 23, 42, 0.98), rgba(30, 41, 59, 0.92)),
            radial-gradient(circle at 12% 0%, rgba(96, 165, 250, 0.1), transparent 42%),
            radial-gradient(circle at 98% 18%, rgba(96, 165, 250, 0.08), transparent 38%),
            var(--semi-color-bg-2);
          box-shadow: 0 14px 34px rgba(0, 0, 0, 0.28) !important;
        }
        html.dark .home-model-discount-tag {
          box-shadow: 0 8px 18px rgba(249, 115, 22, 0.12);
        }
        html.dark .home-model-hot-pill {
          border-color: rgba(251, 191, 36, 0.28);
          background: linear-gradient(135deg, #d97706, #ea580c 48%, #dc2626);
          box-shadow: 0 8px 22px rgba(249, 115, 22, 0.14);
        }
        html.dark .home-model-route-chip {
          background: linear-gradient(135deg, rgba(132, 204, 22, 0.16), rgba(14, 165, 233, 0.16));
        }
        html.dark button.home-model-route-chip:not(:disabled):hover {
          background: linear-gradient(135deg, rgba(132, 204, 22, 0.22), rgba(14, 165, 233, 0.2));
        }
        html.dark .home-model-route-chip-supplier {
          color: #bef264;
        }
        html.dark button.home-model-route-chip:not(:disabled):hover .home-model-route-chip-supplier {
          color: #d9f99d;
        }
        html.dark .home-model-route-chip-suffix {
          color: #7dd3fc;
        }
        html.dark button.home-model-route-chip:not(:disabled):hover .home-model-route-chip-suffix {
          color: #bae6fd;
        }
        @keyframes home-model-card-sheen {
          0%,
          42% {
            transform: translateX(-36%) rotate(4deg);
            opacity: 0;
          }
          52% {
            opacity: 0.9;
          }
          68%,
          100% {
            transform: translateX(34%) rotate(4deg);
            opacity: 0;
          }
        }
        @keyframes home-model-hot-pill-sheen {
          0%,
          42% {
            opacity: 0;
            transform: translateX(-70%) rotate(8deg);
          }
          54% {
            opacity: 0.95;
          }
          72%,
          100% {
            opacity: 0;
            transform: translateX(70%) rotate(8deg);
          }
        }
        @keyframes home-model-card-ribbon {
          0% {
            transform: rotate(0deg) scale(1);
          }
          100% {
            transform: rotate(360deg) scale(1);
          }
        }
        @keyframes home-model-card-aurora {
          0%,
          100% {
            background-position:
              0 0,
              0% 0%,
              100% 0%,
              86% 100%,
              0% 50%,
              0 0;
          }
          50% {
            background-position:
              0 0,
              26% 18%,
              74% 18%,
              64% 82%,
              100% 50%,
              0 0;
          }
        }
        @keyframes home-model-card-orbit {
          0%,
          100% {
            transform: translate3d(0, 0, 0) scale(1);
            opacity: 0.82;
          }
          50% {
            transform: translate3d(-18px, 18px, 0) scale(1.08);
            opacity: 0.58;
          }
        }
        @media (prefers-reduced-motion: reduce) {
          .home-model-hot-pill::before {
            animation: none;
            opacity: 0;
            transform: none;
          }
          .home-model-card-hot::before {
            animation: none;
            opacity: 0.28;
            transform: none;
          }
          .home-model-card-hot,
          .home-model-card-hot::after {
            animation: none;
          }
        }
        @media (max-width: 767px) {
          .home-model-cards-grid {
            grid-template-columns: minmax(0, 1fr);
          }
          .home-model-card {
            min-height: 260px;
          }
        }
        .home-model-layout {
          display: flex;
          gap: 1rem;
          width: 100%;
        }
        @media (max-width: 767px) {
          .home-model-layout {
            flex-direction: column;
            gap: 1rem;
          }
        }
        .home-model-sidebar {
          align-self: flex-start;
          width: 300px;
          min-width: 300px;
          max-width: 300px;
          max-height: calc(100vh - 60px);
          flex-shrink: 0;
          display: flex;
          flex-direction: column;
        }
        @media (max-width: 767px) {
          .home-model-sidebar {
            position: static;
            width: 100%;
            min-width: 100%;
            max-width: 100%;
            max-height: none;
          }
        }
        .home-sidebar-header {
          padding: 1rem 1rem 0.5rem 1rem;
        }
        .home-sidebar-tools {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }
        .home-filter-input.semi-input-wrapper,
        .home-filter-select.semi-select {
          position: relative;
          overflow: hidden;
          border: 1px solid rgba(37, 99, 235, 0.2);
          border-radius: 8px;
          background: rgba(37, 99, 235, 0.06) !important;
          box-shadow: none;
          transition:
            border-color 0.18s ease,
            box-shadow 0.18s ease,
            background 0.18s ease;
        }
        .home-filter-input.semi-input-wrapper:hover,
        .home-filter-select.semi-select:hover {
          border-color: rgba(37, 99, 235, 0.32);
          background: rgba(37, 99, 235, 0.08) !important;
        }
        .home-filter-input.semi-input-wrapper:focus-within,
        .home-filter-input.semi-input-wrapper-focus,
        .home-filter-select.semi-select:focus-within,
        .home-filter-select.semi-select-focus,
        .home-filter-select.semi-select-open {
          border-color: rgba(37, 99, 235, 0.45);
          background: rgba(37, 99, 235, 0.1) !important;
          box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.12);
        }
        .home-filter-input .semi-input,
        .home-filter-input .semi-input-default,
        .home-filter-select .semi-select-selection-text {
          color: #1e293b;
          font-weight: 700;
        }
        .home-filter-input .semi-input::placeholder {
          color: rgba(30, 41, 59, 0.5);
          font-weight: 600;
        }
        .home-filter-input .semi-icon,
        .home-filter-select .semi-select-prefix,
        .home-filter-select .semi-select-arrow {
          color: #2563eb;
        }
        .home-filter-select .semi-select-prefix {
          font-weight: 800;
        }
        .home-filter-select-dropdown .semi-select-option {
          border-radius: 7px;
          margin: 2px 4px;
        }
        .home-filter-select-dropdown .semi-select-option:hover {
          background: rgba(37, 99, 235, 0.08);
        }
        .home-filter-select-dropdown .semi-select-option-selected {
          background: rgba(37, 99, 235, 0.12) !important;
          color: #2563eb;
          font-weight: 800;
        }
        html.dark .home-filter-input.semi-input-wrapper,
        html.dark .home-filter-select.semi-select {
          border-color: rgba(96, 165, 250, 0.24);
          background: rgba(96, 165, 250, 0.1) !important;
          box-shadow: none;
        }
        html.dark .home-filter-input.semi-input-wrapper:hover,
        html.dark .home-filter-select.semi-select:hover {
          border-color: rgba(96, 165, 250, 0.4);
          background: rgba(96, 165, 250, 0.13) !important;
        }
        html.dark .home-filter-input.semi-input-wrapper:focus-within,
        html.dark .home-filter-input.semi-input-wrapper-focus,
        html.dark .home-filter-select.semi-select:focus-within,
        html.dark .home-filter-select.semi-select-focus,
        html.dark .home-filter-select.semi-select-open {
          border-color: rgba(96, 165, 250, 0.55);
          background: rgba(96, 165, 250, 0.15) !important;
          box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.15);
        }
        html.dark .home-filter-input .semi-input,
        html.dark .home-filter-input .semi-input-default,
        html.dark .home-filter-select .semi-select-selection-text {
          color: var(--semi-color-text-0);
        }
        html.dark .home-filter-input .semi-input::placeholder {
          color: var(--semi-color-text-2);
        }
        html.dark .home-filter-input .semi-icon,
        html.dark .home-filter-select .semi-select-prefix,
        html.dark .home-filter-select .semi-select-arrow {
          color: #60a5fa;
        }
        html.dark .home-filter-select-dropdown .semi-select-option:hover {
          background: rgba(96, 165, 250, 0.12);
        }
        html.dark .home-filter-select-dropdown .semi-select-option-selected {
          background: rgba(96, 165, 250, 0.18) !important;
          color: #93c5fd;
        }
        .home-sidebar-filters {
          flex: 1;
          overflow-y: auto;
          scrollbar-width: none;
          padding: 0 1rem 1rem 1rem;
        }
        .home-sidebar-filters > [class*="sbg-variant-"] {
          margin-bottom: 0.875rem !important;
        }
        .home-sidebar-filters .sbg-button {
          max-width: 100%;
          height: auto;
          min-height: 32px;
          padding-top: 5px;
          padding-bottom: 5px;
        }
        .home-sidebar-filters .sbg-button .semi-button-content {
          max-width: 100%;
        }
        .home-sidebar-filters .sbg-content {
          width: auto;
          max-width: 100%;
        }
        .home-sidebar-filters .sbg-content > span:not(.sbg-icon):not(.sbg-badge) {
          min-width: 0;
          white-space: normal !important;
          overflow-wrap: anywhere;
          line-height: 1.25;
        }
        .home-sidebar-filters .sbg-badge {
          align-self: center;
        }
        .home-sidebar-filters::-webkit-scrollbar {
          display: none;
        }
        @media (max-width: 767px) {
          .home-sidebar-header {
            padding: 0 0 1rem 0;
            margin-bottom: 0.5rem;
          }
          .home-sidebar-filters {
            padding: 0;
            overflow-y: visible;
          }
        }
        .home-model-content {
          flex: 1;
          min-width: 0;
        }
        .home-search-wrapper {
          display: none !important;
        }
        .home-search-wrapper-mobile {
          display: none !important;
        }
        @media (min-width: 768px) {
          .home-search-wrapper-mobile {
            display: none !important;
          }
        }
        @media (max-width: 767px) {
          .home-search-wrapper {
            display: none !important;
          }
          .home-model-layout {
            display: flex;
            flex-direction: column;
          }
          .home-search-wrapper-mobile {
            order: 1;
            width: 100%;
            margin-bottom: 1rem;
          }
          .home-model-sidebar {
            order: 1;
            padding-left: 0;
            padding-right: 0;
          }
          .home-model-content {
            order: 2;
            padding-left: 0 !important;
            padding-right: 0 !important;
          }
        }
      `}</style>
      <div className='home-model-layout'>
        {/* 移动端搜索框 */}
        <div className='home-search-wrapper-mobile'>
          <div className='flex flex-col gap-2 w-full'>
            <Input
              prefix={<IconSearch />}
              placeholder={pricingData.t('模糊搜索模型名称')}
              value={pricingData.searchValue}
              onCompositionStart={pricingData.handleCompositionStart}
              onCompositionEnd={pricingData.handleCompositionEnd}
              onChange={pricingData.handleChange}
              showClear
              size='large'
              className='home-filter-input'
            />
            <Select
              size='large'
              className='home-filter-select'
              dropdownClassName='home-filter-select-dropdown'
              style={{ width: '100%' }}
              value={sortSelectValue}
              onChange={(v) =>
                pricingData.setSortKey && pricingData.setSortKey(v)
              }
              optionList={sortOptions}
              prefix={pricingData.t('排序')}
            />
          </div>
        </div>

        <div className='home-model-sidebar'>
          <div className='home-sidebar-header'>
            <div className='flex items-center justify-between mb-4'>
              <div className='text-lg font-semibold text-gray-800'>
                {pricingData.t('筛选')}
              </div>
              <Button
                theme='outline'
                type='tertiary'
                onClick={handleResetFilters}
                className='text-gray-500 hover:text-gray-700'
              >
                {pricingData.t('重置')}
              </Button>
            </div>
            <div className='home-sidebar-tools'>
              <Input
                prefix={<IconSearch />}
                placeholder={pricingData.t('模糊搜索模型名称')}
                value={pricingData.searchValue}
                onCompositionStart={pricingData.handleCompositionStart}
                onCompositionEnd={pricingData.handleCompositionEnd}
                onChange={pricingData.handleChange}
                showClear
                size='default'
                className='home-filter-input'
                style={{ width: '100%' }}
              />
              <Select
                size='default'
                className='home-filter-select'
                dropdownClassName='home-filter-select-dropdown'
                style={{ width: '100%' }}
                value={sortSelectValue}
                onChange={(v) =>
                  pricingData.setSortKey && pricingData.setSortKey(v)
                }
                optionList={sortOptions}
                prefix={pricingData.t('热门筛选')}
              />
            </div>
          </div>

          <div className='home-sidebar-filters'>
            <PricingVendors
              filterVendor={pricingData.filterVendor}
              setFilterVendor={handleFilterVendorChange}
              models={vendorModels}
              allModels={pricingData.models}
              loading={pricingData.loading}
              t={pricingData.t}
              layout='inline'
            />

            {/* <PricingQuotaTypes
            filterQuotaType={pricingData.filterQuotaType}
            setFilterQuotaType={pricingData.setFilterQuotaType}
            models={quotaTypeModels}
            loading={pricingData.loading}
            t={pricingData.t}
          /> */}

            <PricingTags
              filterTag={pricingData.filterTag}
              setFilterTag={handleFilterTagChange}
              models={tagModels}
              allModels={pricingData.models}
              loading={pricingData.loading}
              t={pricingData.t}
              layout='inline'
              showLiveHot
              hotChannelScoreMap={pricingData.hotChannelScoreMap}
              filterSupplier={pricingData.filterSupplier}
              filterSupplierType={pricingData.filterSupplierType}
            />

            <PricingProviderType
              filterProviderType={pricingData.filterSupplierType}
              setFilterProviderType={pricingData.setFilterSupplierType}
              models={pricingData.models}
              countModels={supplierTypeModels}
              loading={pricingData.loading}
              t={pricingData.t}
              layout='inline'
            />

            {/* <PricingEndpointTypes
            filterEndpointType={pricingData.filterEndpointType}
            setFilterEndpointType={pricingData.setFilterEndpointType}
            models={endpointTypeModels}
            allModels={pricingData.models}
            loading={pricingData.loading}
            t={pricingData.t}
          /> */}
          </div>
        </div>

        <div className='home-model-content'>
          <div
            className={`home-search-wrapper ${isMobile ? 'w-full mb-4' : 'w-full my-4 rounded-xl'}`}
            style={{ backgroundColor: 'transparent' }}
          >
            <div className='flex items-center gap-2 w-full'>
              <Input
                prefix={<IconSearch />}
                placeholder={pricingData.t('模糊搜索模型名称')}
                value={pricingData.searchValue}
                onCompositionStart={pricingData.handleCompositionStart}
                onCompositionEnd={pricingData.handleCompositionEnd}
                onChange={pricingData.handleChange}
                showClear
                size='large'
                className='flex-1 home-filter-input'
                style={{
                  backgroundColor: 'var(--semi-color-bg-1)',
                  opacity: 1,
                  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                }}
              />
              <Select
                size='large'
                className='home-filter-select'
                dropdownClassName='home-filter-select-dropdown'
                style={{
                  width: 180,
                  backgroundColor: 'var(--semi-color-bg-1)',
                  opacity: 1,
                  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                }}
                value={sortSelectValue}
                onChange={(v) =>
                  pricingData.setSortKey && pricingData.setSortKey(v)
                }
                optionList={sortOptions}
                prefix={pricingData.t('排序')}
              />
            </div>
          </div>

          <div className='home-model-card-wrapper'>
            <PricingCardView
              filteredModels={pricingData.filteredModels}
              loading={false}
              rowSelection={null}
              pageSize={pricingData.pageSize}
              setPageSize={pricingData.setPageSize}
              currentPage={pricingData.currentPage}
              setCurrentPage={pricingData.setCurrentPage}
              selectedGroup={pricingData.selectedGroup}
              groupRatio={pricingData.groupRatio}
              groupModelPrice={pricingData.groupModelPrice}
              groupModelRatio={pricingData.groupModelRatio}
              copyText={pricingData.copyText}
              setModalImageUrl={pricingData.setModalImageUrl}
              setIsModalOpenurl={pricingData.setIsModalOpenurl}
              currency={pricingData.currency}
              siteDisplayType={pricingData.siteDisplayType}
              tokenUnit={pricingData.tokenUnit}
              displayPrice={pricingData.displayPrice}
              channelVideoRatio={pricingData.channelVideoRatio}
              channelVideoCompletionRatio={
                pricingData.channelVideoCompletionRatio
              }
              channelVideoPrice={pricingData.channelVideoPrice}
              showRatio={false}
              t={pricingData.t}
              selectedRowKeys={EMPTY_SELECTED_ROW_KEYS}
              setSelectedRowKeys={ignoreSelectedRows}
              openModelDetail={pricingData.openModelDetail}
              showSizeChanger={false}
              blurPricing={blurPricing}
              homeCardMode
              showModelDescription
              searchValue={pricingData.searchValue}
              hotChannelScoreMap={pricingData.hotChannelScoreMap}
              filterSupplier={pricingData.filterSupplier}
              filterSupplierType={pricingData.filterSupplierType}
            />
          </div>
        </div>
      </div>

      <ImagePreview
        src={pricingData.modalImageUrl}
        visible={pricingData.isModalOpenurl}
        onVisibleChange={(visible) => pricingData.setIsModalOpenurl(visible)}
      />

      <ModelDetailSideSheet
        visible={pricingData.showModelDetail}
        onClose={pricingData.closeModelDetail}
        modelData={pricingData.selectedModel}
        groupRatio={pricingData.groupRatio}
        groupModelPrice={pricingData.groupModelPrice}
        groupModelRatio={pricingData.groupModelRatio}
        usableGroup={pricingData.usableGroup}
        currency={pricingData.currency}
        siteDisplayType={pricingData.siteDisplayType}
        tokenUnit={pricingData.tokenUnit}
        displayPrice={pricingData.displayPrice}
        showRatio={false}
        vendorsMap={pricingData.vendorsMap}
        endpointMap={pricingData.endpointMap}
        autoGroups={pricingData.autoGroups}
        t={pricingData.t}
        selectedGroup={pricingData.selectedGroup}
        blurPricing={blurPricing}
        showCostPrice={showCostPrice}
        channelModelRatioMap={pricingData.channelModelRatio}
        channelModelPriceMap={pricingData.channelModelPrice}
        channelCompletionRatioMap={pricingData.channelCompletionRatio}
        channelCacheRatioMap={pricingData.channelCacheRatio}
        channelCreateCacheRatioMap={pricingData.channelCreateCacheRatio}
        channelImageRatioMap={pricingData.channelImageRatio}
        channelImagePriceMap={pricingData.channelImagePrice}
        channelAudioRatioMap={pricingData.channelAudioRatio}
        channelAudioCompletionRatioMap={pricingData.channelAudioCompletionRatio}
        channelVideoRatioMap={pricingData.channelVideoRatio}
        channelVideoCompletionRatioMap={pricingData.channelVideoCompletionRatio}
        channelVideoPriceMap={pricingData.channelVideoPrice}
        hotChannelScoreMap={pricingData.hotChannelScoreMap}
      />
    </div>
  );
};

export default HomeModelList;
