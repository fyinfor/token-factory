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

import React, { useEffect, useRef, useState } from 'react';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const EMPTY_STATE = {
  visible: false,
  left: 0,
  bottom: 0,
  width: 0,
  innerWidth: 0,
};

const SCROLLABLE_CONTAINER_SELECTORS = [
  '.semi-modal-body',
  '.semi-sidesheet-body',
  '.semi-card-body',
  '.semi-tabs-content',
  '.semi-layout-content',
  'main',
];

const HIDDEN_CONTAINER_SELECTORS = [
  '.semi-modal-displayNone',
  '.semi-sidesheet-hidden',
  '[aria-hidden="true"]',
  '[hidden]',
];

const FLOATING_LAYER_SELECTORS = [
  '.semi-modal:not(.semi-modal-displayNone)',
  '.semi-sidesheet:not(.semi-sidesheet-hidden)',
];

const getVisibleFloatingLayer = (body) => {
  const layer = FLOATING_LAYER_SELECTORS.map((selector) =>
    body.closest(selector),
  )
    .filter(Boolean)
    .find((element) => {
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      return (
        rect.width > 0 &&
        rect.height > 0 &&
        style.display !== 'none' &&
        style.visibility !== 'hidden'
      );
    });

  return layer || null;
};

const getLayerPriority = (body) => {
  const floatingLayer = getVisibleFloatingLayer(body);
  if (!floatingLayer) return 0;
  return floatingLayer.classList.contains('semi-modal') ? 3 : 2;
};

const getLayerZIndex = (body) => {
  const layer = getVisibleFloatingLayer(body);
  if (!layer) return 0;

  let current = layer;
  while (current && current !== document.body) {
    const zIndex = Number.parseInt(window.getComputedStyle(current).zIndex, 10);
    if (Number.isFinite(zIndex)) return zIndex;
    current = current.parentElement;
  }

  return 0;
};

const isElementVisible = (element) => {
  if (!element || element.closest(HIDDEN_CONTAINER_SELECTORS.join(','))) {
    return false;
  }

  const rect = element.getBoundingClientRect();
  const style = window.getComputedStyle(element);
  return (
    rect.width > 0 &&
    rect.height > 0 &&
    style.display !== 'none' &&
    style.visibility !== 'hidden'
  );
};

const getScrollParent = (element) => {
  let current = element?.parentElement;
  while (current && current !== document.body) {
    const style = window.getComputedStyle(current);
    if (/(auto|scroll|overlay)/.test(`${style.overflowY}${style.overflow}`)) {
      return current;
    }
    current = current.parentElement;
  }
  return window;
};

const getScrollBoundary = (tableBody) => {
  const matched = SCROLLABLE_CONTAINER_SELECTORS.map((selector) =>
    tableBody.closest(selector),
  )
    .filter(Boolean)
    .find((element) => {
      const rect = element.getBoundingClientRect();
      return rect.height > 0 && rect.width > 0;
    });

  return matched || getScrollParent(tableBody);
};

const getBoundaryRect = (boundary) => {
  if (!boundary || boundary === window) {
    return {
      top: 0,
      bottom: window.innerHeight,
      left: 0,
      right: window.innerWidth,
    };
  }

  return boundary.getBoundingClientRect();
};

const pickVisibleTableBody = () => {
  const bodies = Array.from(
    document.querySelectorAll('.semi-table-body'),
  ).filter(isElementVisible);

  return bodies
    .map((body) => {
      const bodyRect = body.getBoundingClientRect();
      const boundary = getScrollBoundary(body);
      const boundaryRect = getBoundaryRect(boundary);
      const visibleTop = Math.max(bodyRect.top, boundaryRect.top, 0);
      const visibleBottom = Math.min(
        bodyRect.bottom,
        boundaryRect.bottom,
        window.innerHeight,
      );
      const visibleLeft = Math.max(bodyRect.left, boundaryRect.left, 0);
      const visibleRight = Math.min(
        bodyRect.right,
        boundaryRect.right,
        window.innerWidth,
      );
      const visibleHeight = visibleBottom - visibleTop;
      const visibleWidth = visibleRight - visibleLeft;
      const originalScrollbarVisible = bodyRect.bottom <= visibleBottom + 1;
      const hasHorizontalOverflow = body.scrollWidth > body.clientWidth + 1;
      const layerPriority = getLayerPriority(body);
      const layerZIndex = getLayerZIndex(body);

      return {
        body,
        boundary,
        visibleTop,
        visibleBottom,
        visibleLeft,
        visibleRight,
        visibleHeight,
        visibleWidth,
        originalScrollbarVisible,
        hasHorizontalOverflow,
        layerPriority,
        layerZIndex,
      };
    })
    .filter(
      (item) =>
        item.hasHorizontalOverflow &&
        !item.originalScrollbarVisible &&
        item.visibleHeight > 24 &&
        item.visibleWidth > 24 &&
        item.body.clientWidth > 24,
    )
    .sort((a, b) => {
      if (b.layerPriority !== a.layerPriority) {
        return b.layerPriority - a.layerPriority;
      }
      if (b.layerZIndex !== a.layerZIndex) {
        return b.layerZIndex - a.layerZIndex;
      }
      if (b.visibleBottom !== a.visibleBottom) {
        return b.visibleBottom - a.visibleBottom;
      }
      return b.visibleHeight - a.visibleHeight;
    })[0];
};

const StickyTableScrollbar = () => {
  const isMobile = useIsMobile();
  const scrollbarRef = useRef(null);
  const tableBodyRef = useRef(null);
  const scrollBoundaryRef = useRef(null);
  const frameRef = useRef(0);
  const mutationObserverRef = useRef(null);
  const resizeObserverRef = useRef(null);
  const [state, setState] = useState(EMPTY_STATE);

  useEffect(() => {
    if (isMobile) {
      setState(EMPTY_STATE);
      return undefined;
    }

    const scrollbar = scrollbarRef.current;
    if (!scrollbar) return undefined;

    const setNextState = (nextState) => {
      setState((prev) => {
        const unchanged =
          prev.visible === nextState.visible &&
          Math.abs(prev.left - nextState.left) < 1 &&
          Math.abs(prev.bottom - nextState.bottom) < 1 &&
          Math.abs(prev.width - nextState.width) < 1 &&
          Math.abs(prev.innerWidth - nextState.innerWidth) < 1;

        return unchanged ? prev : nextState;
      });
    };

    const detachTableBodyScroll = () => {
      if (tableBodyRef.current) {
        tableBodyRef.current.removeEventListener('scroll', syncFromTableBody);
      }
      if (scrollBoundaryRef.current && scrollBoundaryRef.current !== window) {
        scrollBoundaryRef.current.removeEventListener(
          'scroll',
          scheduleMeasure,
        );
      }
      tableBodyRef.current = null;
      scrollBoundaryRef.current = null;
    };

    const attachTableBodyScroll = (body, boundary) => {
      if (
        tableBodyRef.current === body &&
        scrollBoundaryRef.current === boundary
      ) {
        return;
      }

      detachTableBodyScroll();
      tableBodyRef.current = body;
      scrollBoundaryRef.current = boundary;
      body.addEventListener('scroll', syncFromTableBody, { passive: true });
      if (boundary && boundary !== window) {
        boundary.addEventListener('scroll', scheduleMeasure, {
          passive: true,
        });
      }
    };

    const hideScrollbar = () => {
      detachTableBodyScroll();
      setNextState(EMPTY_STATE);
    };

    function syncFromTableBody() {
      const body = tableBodyRef.current;
      if (body && scrollbar.scrollLeft !== body.scrollLeft) {
        scrollbar.scrollLeft = body.scrollLeft;
      }
    }

    function syncToTableBody() {
      const body = tableBodyRef.current;
      if (body && body.scrollLeft !== scrollbar.scrollLeft) {
        body.scrollLeft = scrollbar.scrollLeft;
      }
    }

    function measure() {
      const active = pickVisibleTableBody();

      if (!active) {
        hideScrollbar();
        return;
      }

      attachTableBodyScroll(active.body, active.boundary);
      setNextState({
        visible: true,
        left: active.visibleLeft,
        bottom: Math.max(window.innerHeight - active.visibleBottom, 0),
        width: active.visibleWidth,
        innerWidth: active.body.scrollWidth,
      });

      if (scrollbar.scrollLeft !== active.body.scrollLeft) {
        scrollbar.scrollLeft = active.body.scrollLeft;
      }
    }

    function scheduleMeasure() {
      if (frameRef.current) cancelAnimationFrame(frameRef.current);
      frameRef.current = requestAnimationFrame(measure);
    }

    scheduleMeasure();
    scrollbar.addEventListener('scroll', syncToTableBody, { passive: true });
    window.addEventListener('resize', scheduleMeasure);
    window.addEventListener('scroll', scheduleMeasure, true);

    if (typeof ResizeObserver !== 'undefined') {
      resizeObserverRef.current = new ResizeObserver(scheduleMeasure);
      resizeObserverRef.current.observe(document.body);
    }

    if (typeof MutationObserver !== 'undefined') {
      mutationObserverRef.current = new MutationObserver(scheduleMeasure);
      mutationObserverRef.current.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ['class', 'style'],
      });
    }

    return () => {
      if (frameRef.current) cancelAnimationFrame(frameRef.current);
      detachTableBodyScroll();
      scrollbar.removeEventListener('scroll', syncToTableBody);
      window.removeEventListener('resize', scheduleMeasure);
      window.removeEventListener('scroll', scheduleMeasure, true);
      resizeObserverRef.current?.disconnect();
      mutationObserverRef.current?.disconnect();
    };
  }, [isMobile]);

  if (isMobile) return null;

  return (
    <div
      ref={scrollbarRef}
      aria-hidden='true'
      className={`sticky-table-horizontal-scrollbar ${
        state.visible ? 'is-visible' : ''
      }`}
      style={{
        left: state.left,
        bottom: state.bottom,
        width: state.width,
      }}
    >
      <div style={{ width: state.innerWidth }} />
    </div>
  );
};

export default StickyTableScrollbar;
