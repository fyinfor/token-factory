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
import { Link, useLocation } from 'react-router-dom';
import SkeletonWrapper from '../components/SkeletonWrapper';

const Navigation = ({
  mainNavLinks,
  isMobile,
  isLoading,
  userState,
  pricingRequireAuth,
}) => {
  const location = useLocation();

  const isActive = (linkPath) => {
    if (!linkPath) return false;
    const currentPath = location.pathname;
    if (linkPath === '/') {
      return currentPath === '/';
    }
    return currentPath.startsWith(linkPath);
  };

  const renderNavLinks = () => {
    const baseClasses =
      'flex-shrink-0 flex items-center text-sm font-medium transition-all duration-200 ease-in-out rounded-md';
    const spacingClasses = isMobile ? 'px-2 py-1' : 'px-3 py-2';
    const hoverClasses =
      'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700/50 hover:!text-semi-color-text-0 dark:hover:!text-white';

    return mainNavLinks.map((link) => {
      const linkContent = <span>{link.text}</span>;

      if (link.isExternal) {
        const openInNewTab = link.openInNewTab !== false;
        return (
          <a
            key={link.itemKey}
            href={link.externalLink}
            {...(openInNewTab
              ? { target: '_blank', rel: 'noopener noreferrer' }
              : {})}
            className={`${baseClasses} ${spacingClasses} ${hoverClasses} !text-semi-color-text-1 dark:!text-gray-300`}
          >
            {linkContent}
          </a>
        );
      }

      let targetPath = link.to;
      if (link.itemKey === 'console' && !userState.user) {
        targetPath = '/login';
      }
      if (link.itemKey === 'pricing' && pricingRequireAuth && !userState.user) {
        targetPath = '/login';
      }

      const active = isActive(link.to);
      const activeClasses = active
        ? '!bg-semi-color-fill-2 dark:!bg-gray-700 !text-semi-color-text-0 dark:!text-white'
        : '!text-semi-color-text-1 dark:!text-gray-300';

      return (
        <Link
          key={link.itemKey}
          to={targetPath}
          className={`${baseClasses} ${spacingClasses} ${hoverClasses} ${activeClasses}`}
        >
          {linkContent}
        </Link>
      );
    });
  };

  return (
    <nav className='hidden md:flex items-center gap-1 overflow-x-auto whitespace-nowrap scrollbar-hide'>
      <SkeletonWrapper
        loading={isLoading}
        type='navigation'
        count={4}
        width={60}
        height={16}
        isMobile={isMobile}
      >
        {renderNavLinks()}
      </SkeletonWrapper>
    </nav>
  );
};

export default Navigation;
