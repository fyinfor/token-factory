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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Highlight, Input, Modal } from '@douyinfe/semi-ui';
import { Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import {
  CONSOLE_SEARCH_ITEMS,
  QUICK_SEARCH_FEATURED_IDS,
} from '../../../constants/consoleSearch.constants';
import { SETTING_SEARCH_ITEMS } from '../../../constants/setting.constants';
import { USER_ROLES } from '../../../constants/user.constants';

const normalizeSearchText = (value) =>
  String(value || '')
    .trim()
    .toLowerCase();

const includesQuery = (value, normalizedQuery) =>
  normalizeSearchText(value).includes(normalizedQuery);

const getShortcutLabel = () => {
  if (typeof navigator === 'undefined') return 'Ctrl K';

  const platform =
    navigator.userAgentData?.platform ||
    navigator.platform ||
    navigator.userAgent;
  return /mac|iphone|ipad|ipod/i.test(platform) ? '⌘ K' : 'Ctrl K';
};

const SettingsQuickSearch = ({ userState }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [visible, setVisible] = useState(false);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const resultRefs = useRef([]);
  const normalizedQuery = normalizeSearchText(query);
  const isRootUser = Number(userState?.user?.role) >= USER_ROLES.ROOT;
  const shortcutLabel = useMemo(getShortcutLabel, []);

  const searchItems = useMemo(
    () => [
      ...CONSOLE_SEARCH_ITEMS.map((item) => ({
        ...item,
        rawLabel: item.label,
        rawPath: item.pathLabels.join(' / '),
        rawKeywords: item.keywords,
        label: t(item.label),
        path: item.pathLabels.map((label) => t(label)).join(' / '),
        keywords: item.keywords.map((keyword) => t(keyword)),
      })),
      ...SETTING_SEARCH_ITEMS.map((item) => {
        const pathLabels = ['系统设置', item.categoryLabel, item.pageLabel];

        return {
          id: `setting:${item.categoryKey}:${item.pageKey}:${item.key}`,
          type: 'setting',
          url: item.url,
          rawLabel: item.label,
          rawPath: pathLabels.join(' / '),
          rawKeywords: item.keywords,
          label: t(item.label),
          path: pathLabels.map((label) => t(label)).join(' / '),
          keywords: item.keywords.map((keyword) => t(keyword)),
        };
      }),
    ],
    [t],
  );

  const featuredResults = useMemo(() => {
    const itemsById = new Map(searchItems.map((item) => [item.id, item]));

    return QUICK_SEARCH_FEATURED_IDS.map((id) => itemsById.get(id))
      .filter(Boolean)
      .map((item) => ({
        ...item,
        displayTitle: item.label,
        displayPath: item.path,
        matchedKeywords: [],
      }));
  }, [searchItems]);

  const results = useMemo(() => {
    if (!normalizedQuery) return [];

    return searchItems
      .map((item) => {
        const searchableValues = [
          item.rawLabel,
          item.rawPath,
          ...item.rawKeywords,
          item.label,
          item.path,
          ...item.keywords,
        ].map(normalizeSearchText);

        if (
          !searchableValues.some((value) => value.includes(normalizedQuery))
        ) {
          return null;
        }

        const labelSearchText = normalizeSearchText(item.label);
        const rawLabelSearchText = normalizeSearchText(item.rawLabel);
        const pathSearchText = normalizeSearchText(item.path);
        const rawPathSearchText = normalizeSearchText(item.rawPath);
        const score =
          labelSearchText === normalizedQuery ||
          rawLabelSearchText === normalizedQuery
            ? 0
            : labelSearchText.startsWith(normalizedQuery) ||
                rawLabelSearchText.startsWith(normalizedQuery)
              ? 1
              : labelSearchText.includes(normalizedQuery) ||
                  rawLabelSearchText.includes(normalizedQuery)
                ? 2
                : pathSearchText.includes(normalizedQuery) ||
                    rawPathSearchText.includes(normalizedQuery)
                  ? 3
                  : 4;
        const displayTitle = includesQuery(item.label, normalizedQuery)
          ? item.label
          : includesQuery(item.rawLabel, normalizedQuery)
            ? item.rawLabel
            : item.label;
        const displayPath = includesQuery(item.path, normalizedQuery)
          ? item.path
          : includesQuery(item.rawPath, normalizedQuery)
            ? item.rawPath
            : item.path;
        const matchedKeywords = [...item.keywords, ...item.rawKeywords].filter(
          (keyword, index, values) =>
            includesQuery(keyword, normalizedQuery) &&
            values.indexOf(keyword) === index,
        );

        return {
          ...item,
          displayTitle,
          displayPath,
          matchedKeywords,
          score,
        };
      })
      .filter(Boolean)
      .sort((left, right) =>
        left.score === right.score
          ? left.type === right.type
            ? left.label.localeCompare(right.label)
            : left.type === 'console'
              ? -1
              : 1
          : left.score - right.score,
      )
      .slice(0, 30);
  }, [normalizedQuery, searchItems]);

  const displayResults = normalizedQuery ? results : featuredResults;

  useEffect(() => {
    setActiveIndex(0);
  }, [normalizedQuery, visible]);

  useEffect(() => {
    resultRefs.current[activeIndex]?.scrollIntoView({ block: 'nearest' });
  }, [activeIndex]);

  useEffect(() => {
    if (!isRootUser) return undefined;

    const handleShortcut = (event) => {
      if (
        event.defaultPrevented ||
        event.isComposing ||
        event.altKey ||
        event.shiftKey ||
        (!event.ctrlKey && !event.metaKey) ||
        event.key.toLowerCase() !== 'k'
      ) {
        return;
      }

      event.preventDefault();
      setVisible(true);
    };

    window.addEventListener('keydown', handleShortcut);
    return () => window.removeEventListener('keydown', handleShortcut);
  }, [isRootUser]);

  if (!isRootUser) {
    return null;
  }

  const closeModal = () => {
    setVisible(false);
    setQuery('');
  };

  const openResult = (result) => {
    navigate(result.url);
    closeModal();
  };

  const handleSearchKeyDown = (event) => {
    if (displayResults.length === 0) return;

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex((current) => (current + 1) % displayResults.length);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex(
        (current) =>
          (current - 1 + displayResults.length) % displayResults.length,
      );
    } else if (event.key === 'Enter') {
      event.preventDefault();
      openResult(displayResults[activeIndex] || displayResults[0]);
    }
  };

  const highlightProps = {
    searchWords: normalizedQuery ? [query.trim()] : [],
    autoEscape: true,
    caseSensitive: false,
    highlightClassName: 'settings-search-highlight',
  };

  return (
    <>
      <button
        type='button'
        className='settings-quick-search-trigger'
        onClick={() => setVisible(true)}
        aria-label={t('搜索功能')}
        title={`${t('搜索功能')} (${shortcutLabel})`}
        aria-haspopup='dialog'
        aria-expanded={visible}
        aria-keyshortcuts='Control+K Meta+K'
      >
        <Search size={15} strokeWidth={2} />
        <span className='settings-quick-search-trigger-label'>
          {t('搜索功能')}
        </span>
        <kbd className='settings-quick-search-shortcut'>{shortcutLabel}</kbd>
      </button>

      <Modal
        className='settings-quick-search-modal'
        title={t('功能快捷搜索')}
        visible={visible}
        onCancel={closeModal}
        footer={null}
        centered
        width='min(680px, calc(100vw - 24px))'
        bodyStyle={{ padding: '12px 20px 20px' }}
      >
        <Input
          className='settings-search-input'
          autoFocus
          value={query}
          onChange={setQuery}
          onKeyDown={handleSearchKeyDown}
          prefix={<Search size={16} />}
          placeholder={t('输入功能名称或关键字')}
          showClear
          size='large'
        />

        <div className='settings-search-results' role='listbox'>
          {!normalizedQuery && (
            <div className='settings-search-results-label'>{t('常用功能')}</div>
          )}
          {normalizedQuery && results.length === 0 ? (
            <div className='settings-search-empty'>{t('未找到匹配的功能')}</div>
          ) : (
            displayResults.map((result, index) => (
              <button
                type='button'
                key={result.id}
                ref={(element) => {
                  resultRefs.current[index] = element;
                }}
                className={`settings-search-result${index === activeIndex ? ' is-active' : ''}`}
                onClick={() => openResult(result)}
                onMouseEnter={() => setActiveIndex(index)}
                role='option'
                aria-selected={index === activeIndex}
              >
                <span className='settings-search-result-title'>
                  <Highlight
                    {...highlightProps}
                    sourceString={result.displayTitle}
                  />
                </span>
                <span className='settings-search-result-path'>
                  <Highlight
                    {...highlightProps}
                    sourceString={result.displayPath}
                  />
                </span>
                {result.matchedKeywords.length > 0 && (
                  <span className='settings-search-result-keywords'>
                    <Highlight
                      {...highlightProps}
                      sourceString={result.matchedKeywords.join(' / ')}
                    />
                  </span>
                )}
              </button>
            ))
          )}
        </div>
      </Modal>
    </>
  );
};

export default SettingsQuickSearch;
