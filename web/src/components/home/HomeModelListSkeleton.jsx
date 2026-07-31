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

import React from 'react';

const FILTER_ROWS = [4, 5, 3];
const CARD_COUNT = 12;

const SkeletonBlock = ({ className = '' }) => (
  <span className={`home-model-skeleton-block ${className}`} />
);

const HomeModelListSkeleton = ({ label = 'Loading models' }) => (
  <div
    className='home-model-skeleton'
    role='status'
    aria-live='polite'
    aria-busy='true'
  >
    <span className='sr-only'>{label}</span>
    <style>{`
      .home-model-skeleton {
        width: 100%;
        min-height: 420px;
        animation: home-model-skeleton-enter 180ms ease-out both;
      }
      .home-model-skeleton-layout {
        display: flex;
        width: 100%;
        gap: 1rem;
      }
      .home-model-skeleton-sidebar {
        width: 300px;
        min-width: 300px;
        padding: 1rem;
      }
      .home-model-skeleton-heading {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 1rem;
      }
      .home-model-skeleton-tools {
        display: grid;
        gap: 0.5rem;
        margin-bottom: 1.25rem;
      }
      .home-model-skeleton-filter + .home-model-skeleton-filter {
        margin-top: 1rem;
      }
      .home-model-skeleton-pills {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
        margin-top: 0.625rem;
      }
      .home-model-skeleton-content {
        min-width: 0;
        flex: 1;
        padding: 0.5rem;
      }
      .home-model-skeleton-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
        align-items: stretch;
        gap: 0.75rem;
      }
      .home-model-skeleton-card {
        min-width: 0;
        min-height: 278px;
        padding: 1.125rem;
        overflow: hidden;
        border: 1px solid rgba(59, 130, 246, 0.1);
        border-radius: 10px;
        background: rgba(255, 255, 255, 0.62);
      }
      .home-model-skeleton-card-head {
        display: flex;
        align-items: flex-start;
        gap: 0.75rem;
      }
      .home-model-skeleton-card-title {
        display: grid;
        min-width: 0;
        flex: 1;
        gap: 0.5rem;
        padding-top: 0.125rem;
      }
      .home-model-skeleton-card-action {
        flex: 0 0 auto;
      }
      .home-model-skeleton-price {
        display: grid;
        gap: 0.5rem;
        margin-top: 1.125rem;
        padding: 0.75rem;
        border: 1px solid rgba(59, 130, 246, 0.1);
        border-radius: 8px;
        background: rgba(255, 255, 255, 0.42);
      }
      .home-model-skeleton-price-row {
        display: grid;
        grid-template-columns: 0.75fr 1fr 1fr 0.65fr;
        gap: 0.5rem;
      }
      .home-model-skeleton-card-foot {
        display: flex;
        justify-content: space-between;
        gap: 0.75rem;
        margin-top: 1rem;
      }
      .home-model-skeleton-block {
        position: relative;
        display: block;
        overflow: hidden;
        border-radius: 6px;
        background: rgba(100, 116, 139, 0.13);
      }
      .home-model-skeleton-block::after {
        content: '';
        position: absolute;
        inset: 0;
        transform: translateX(-100%);
        background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.72), transparent);
        animation: home-model-skeleton-shimmer 1.55s ease-in-out infinite;
      }
      .home-model-skeleton-line-sm { width: 34%; height: 12px; }
      .home-model-skeleton-line-md { width: 62%; height: 15px; }
      .home-model-skeleton-line-lg { width: 78%; height: 20px; }
      .home-model-skeleton-input { width: 100%; height: 32px; border-radius: 8px; }
      .home-model-skeleton-pill { width: 70px; height: 30px; border-radius: 8px; }
      .home-model-skeleton-pill:nth-child(2n) { width: 88px; }
      .home-model-skeleton-avatar { width: 42px; height: 42px; flex: 0 0 auto; border-radius: 9px; }
      .home-model-skeleton-icon { width: 28px; height: 28px; border-radius: 7px; }
      .home-model-skeleton-cell { width: 100%; height: 12px; }
      .home-model-skeleton-tag { width: 72px; height: 24px; border-radius: 999px; }
      .home-model-skeleton-detail { width: 30px; height: 24px; border-radius: 7px; }
      html.dark .home-model-skeleton-card {
        border-color: var(--semi-color-border);
        background: var(--semi-color-bg-2);
      }
      html.dark .home-model-skeleton-price {
        border-color: rgba(255, 255, 255, 0.08);
        background: rgba(255, 255, 255, 0.035);
      }
      html.dark .home-model-skeleton-block {
        background: rgba(148, 163, 184, 0.16);
      }
      html.dark .home-model-skeleton-block::after {
        background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.09), transparent);
      }
      @keyframes home-model-skeleton-shimmer {
        100% { transform: translateX(100%); }
      }
      @keyframes home-model-skeleton-enter {
        from { opacity: 0; }
        to { opacity: 1; }
      }
      @media (max-width: 767px) {
        .home-model-skeleton-layout { flex-direction: column; }
        .home-model-skeleton-sidebar {
          width: 100%;
          min-width: 100%;
          padding: 0 0 0.25rem;
        }
        .home-model-skeleton-content { padding: 0; }
        .home-model-skeleton-grid { grid-template-columns: minmax(0, 1fr); }
        .home-model-skeleton-card { min-height: 260px; }
        .home-model-skeleton-card:nth-child(n + 5) { display: none; }
      }
      @media (prefers-reduced-motion: reduce) {
        .home-model-skeleton { animation: none; }
        .home-model-skeleton-block::after { animation: none; }
      }
    `}</style>

    <div className='home-model-skeleton-layout' aria-hidden='true'>
      <aside className='home-model-skeleton-sidebar'>
        <div className='home-model-skeleton-heading'>
          <SkeletonBlock className='home-model-skeleton-line-md' />
          <SkeletonBlock className='home-model-skeleton-detail' />
        </div>
        <div className='home-model-skeleton-tools'>
          <SkeletonBlock className='home-model-skeleton-input' />
          <SkeletonBlock className='home-model-skeleton-input' />
        </div>
        {FILTER_ROWS.map((pillCount, rowIndex) => (
          <div className='home-model-skeleton-filter' key={rowIndex}>
            <SkeletonBlock className='home-model-skeleton-line-sm' />
            <div className='home-model-skeleton-pills'>
              {Array.from({ length: pillCount }).map((_, pillIndex) => (
                <SkeletonBlock
                  className='home-model-skeleton-pill'
                  key={pillIndex}
                />
              ))}
            </div>
          </div>
        ))}
      </aside>

      <div className='home-model-skeleton-content'>
        <div className='home-model-skeleton-grid'>
          {Array.from({ length: CARD_COUNT }).map((_, cardIndex) => (
            <div className='home-model-skeleton-card' key={cardIndex}>
              <div className='home-model-skeleton-card-head'>
                <SkeletonBlock className='home-model-skeleton-avatar' />
                <div className='home-model-skeleton-card-title'>
                  <SkeletonBlock className='home-model-skeleton-line-lg' />
                  <SkeletonBlock className='home-model-skeleton-line-md' />
                </div>
                <SkeletonBlock className='home-model-skeleton-icon home-model-skeleton-card-action' />
              </div>
              <div className='home-model-skeleton-price'>
                {Array.from({ length: 3 }).map((_, rowIndex) => (
                  <div className='home-model-skeleton-price-row' key={rowIndex}>
                    {Array.from({ length: 4 }).map((__, cellIndex) => (
                      <SkeletonBlock
                        className='home-model-skeleton-cell'
                        key={cellIndex}
                      />
                    ))}
                  </div>
                ))}
              </div>
              <div className='home-model-skeleton-card-foot'>
                <SkeletonBlock className='home-model-skeleton-tag' />
                <SkeletonBlock className='home-model-skeleton-detail' />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  </div>
);

export default HomeModelListSkeleton;
