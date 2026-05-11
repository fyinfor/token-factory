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

/**
 * 右侧轻拟物 3D 风装饰（低饱和奶油橙 / 柔蓝），无外链资源。
 */
const HomeBannerIllustration = ({ className = '' }) => (
  <div
    className={`pointer-events-none select-none ${className}`}
    aria-hidden
  >
    <svg
      viewBox='0 0 200 140'
      className='h-full w-full max-h-[140px] drop-shadow-sm'
      fill='none'
      xmlns='http://www.w3.org/2000/svg'
    >
      <defs>
        <linearGradient
          id='bb-chip'
          x1='0%'
          y1='0%'
          x2='100%'
          y2='100%'
        >
          <stop offset='0%' stopColor='#FFE4CC' />
          <stop offset='100%' stopColor='#FFD4A8' />
        </linearGradient>
        <linearGradient id='bb-cube' x1='0%' y1='0%' x2='0%' y2='100%'>
          <stop offset='0%' stopColor='#FFF9F5' />
          <stop offset='100%' stopColor='#F5E6D8' />
        </linearGradient>
        <linearGradient id='bb-flow' x1='0%' y1='50%' x2='100%' y2='50%'>
          <stop offset='0%' stopColor='#E8F0FE' stopOpacity='0.9' />
          <stop offset='100%' stopColor='#FDE8D4' stopOpacity='0.85' />
        </linearGradient>
        <filter
          id='bb-soft'
          x='-20%'
          y='-20%'
          width='140%'
          height='140%'
        >
          <feGaussianBlur in='SourceGraphic' stdDeviation='0.8' />
        </filter>
      </defs>
      {/* 底座平台 */}
      <ellipse
        cx='100'
        cy='118'
        rx='72'
        ry='10'
        fill='#F0E6DC'
        opacity='0.65'
      />
      {/* GPU / 模块块 */}
      <g transform='translate(28 36)'>
        <rect
          x='0'
          y='18'
          width='56'
          height='44'
          rx='10'
          fill='url(#bb-chip)'
          stroke='#E8D5C4'
          strokeWidth='1'
        />
        <rect
          x='8'
          y='26'
          width='40'
          height='6'
          rx='2'
          fill='#FFFDFB'
          opacity='0.9'
        />
        <rect
          x='8'
          y='38'
          width='28'
          height='5'
          rx='2'
          fill='#FFFDFB'
          opacity='0.55'
        />
      </g>
      {/* Token 小立方 */}
      <g transform='translate(118 52)'>
        <path
          d='M22 8 L38 16 L38 36 L22 44 L6 36 L6 16 Z'
          fill='url(#bb-cube)'
          stroke='#E5D4C8'
          strokeWidth='0.8'
        />
        <path
          d='M6 16 L22 8 L38 16 L22 24 Z'
          fill='#FFFCF8'
          opacity='0.95'
        />
        <text
          x='22'
          y='30'
          textAnchor='middle'
          fontSize='11'
          fontWeight='700'
          fill='#C2410C'
          opacity='0.75'
        >
          AI
        </text>
      </g>
      {/* 数据流弧线 */}
      <path
        d='M 20 78 Q 100 48 175 70'
        stroke='url(#bb-flow)'
        strokeWidth='5'
        strokeLinecap='round'
        opacity='0.55'
        filter='url(#bb-soft)'
      />
      {/* 节点圆 */}
      <circle cx='42' cy='24' r='5' fill='#FDE8D4' stroke='#F5D0B8' />
      <circle cx='158' cy='32' r='4' fill='#E8F0FE' stroke='#D4E2FC' />
      <circle cx='168' cy='88' r='5' fill='#FFF4EC' stroke='#FFDCC4' />
    </svg>
  </div>
);

export default HomeBannerIllustration;
