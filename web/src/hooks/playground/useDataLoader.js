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

import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  processModelsData,
  processGroupsData,
  showError,
} from '../../helpers';
import { API_ENDPOINTS } from '../../constants/playground.constants';

/**
 * 判断「全部类型」未选（与 vendor_id=0 的已选相区分，0 为合法值）
 * @param {string|number|undefined|null} v
 * @returns {boolean}
 */
const isTypeSelectionEmpty = (v) => v === '' || v == null || v === undefined;

/**
 * 规范化操練场「模型类型」的值为数字或空（全部类型）
 * @param {string|number|undefined|null} v
 * @returns {string|number}
 */
const normalizeTypeId = (v) => {
  if (isTypeSelectionEmpty(v)) return '';
  const n = Number(v);
  return Number.isNaN(n) ? '' : n;
};

/**
 * 与模型广场一致：按「当前选中的类型」在客户端筛模型；优先 vendor_id，其次与下拉项 label 与 item.vendor 对齐（与广场按 vendor 名称一致）
 * @param {{ model_name: string, vendor_id?: number, vendor?: string }[]} listForPlayground
 * @param {string|number} effectiveType
 * @param {Array<{ label: string, value: string|number }>} typeOptions
 * @returns {typeof listForPlayground}
 */
const filterPlaygroundModelsByType = (
  listForPlayground,
  effectiveType,
  typeOptions,
) => {
  if (effectiveType === '' || isTypeSelectionEmpty(effectiveType)) {
    return listForPlayground;
  }
  const eff = Number(effectiveType);
  const typeOpt = (typeOptions || []).find(
    (o) => o && o.value !== '' && Number(o.value) === eff,
  );
  const labelForType = String(typeOpt?.label || '').trim();
  return listForPlayground.filter((item) => {
    const vid = Number(item?.vendor_id ?? 0);
    if (Number.isFinite(eff) && vid === eff) {
      return true;
    }
    if (labelForType && String(item?.vendor || '').trim() === labelForType) {
      return true;
    }
    return false;
  });
};

/**
 * 操练场数据加载：拉取用户模型与类型（与模型广场同源元数据）、按「全部/类型」在客户端筛模型（同模型广场逻辑），分组单独加载。
 * @param {{ user?: object }} userState 已登录用户状态
 * @param {{ model?: string, model_type?: string|number, group?: string, specific_channel_id?: string|number }} inputs 当前表单/配置
 * @param {Array<{ label: string, value: string|number }>} modelTypes 类型下拉项（与接口同步后的状态，用于按类型重算模型列表）
 * @param {function(string, unknown): void} handleInputChange 更新单项输入
 * @param {import('react').Dispatch<import('react').SetStateAction<any[]>>} setModels 设置模型下拉选项
 * @param {import('react').Dispatch<import('react').SetStateAction<any[]>>} setModelTypes 设置模型类型下拉
 * @param {import('react').Dispatch<import('react').SetStateAction<any[]>>} setSupplierOptions 设置渠道下拉
 * @param {import('react').Dispatch<import('react').SetStateAction<any[]>>} setGroups 设置分组下拉
 * @returns {{ loadModels: function, loadGroups: function }}
 */
export const useDataLoader = (
  userState,
  inputs,
  modelTypes,
  handleInputChange,
  setModels,
  setModelTypes,
  setSupplierOptions,
  setGroups,
) => {
  const { t } = useTranslation();
  /**
   * 接口返回的带 vendor 信息的原始模型行，供「仅客户端」按类型过滤（与模型广场一致、且避免与请求竞态导致类型与列表不同步）
   */
  const [playgroundRawModels, setPlaygroundRawModels] = useState([]);

  /**
   * 拉取 scene=playground 模型列表，写入「类型选项」与原始行；不依赖当前 model_type，避免重复请求与请求返回顺序错配。
   */
  const loadModels = useCallback(async () => {
    try {
      const res = await API.get(
        `${API_ENDPOINTS.USER_MODELS}?scene=playground`,
      );
      const { success, message, data } = res.data;

      if (success) {
        const items = Array.isArray(data) ? data : data?.items || [];
        const vendorOptionsFromAPI = Array.isArray(data?.vendor_options)
          ? data.vendor_options
          : [];
        const normalizedItems = items.map((item) => {
          if (typeof item === 'string') {
            return {
              model_name: item,
              vendor_id: 0,
              vendor: '',
              tested_success: true,
            };
          }
          return item;
        });
        // 后端已仅返回「渠道单测最近一次成功」的模型；下列表与类型均基于此子集
        const listForPlayground = normalizedItems.filter(
          (item) => item?.model_name,
        );
        setPlaygroundRawModels(listForPlayground);
        const typeOptionsFromAPI = vendorOptionsFromAPI
          .map((item) => ({
            label: String(item?.name || '').trim(),
            value: Number(item?.id),
          }))
          .filter(
            (item) =>
              item.label !== '' &&
              Number.isFinite(item.value) &&
              (item.value > 0 || item.value === 0),
          );
        // 兜底：若后端未返回 vendor_options，从当前用户可用模型项（全量项）提取类型（与广场「从数据里出现的供应商」一致；含未关联 id=0）
        const fallbackTypeMap = new Map();
        normalizedItems.forEach((item) => {
          const vendorName = String(item?.vendor || '').trim();
          const vendorID = Number(item?.vendor_id ?? 0);
          if (Number.isFinite(vendorID) && vendorID === 0) {
            fallbackTypeMap.set(0, {
              label: t('未知模型类型'),
              value: 0,
            });
            return;
          }
          if (vendorName !== '' && Number.isFinite(vendorID) && vendorID > 0) {
            fallbackTypeMap.set(vendorID, {
              label: vendorName,
              value: vendorID,
            });
          }
        });
        const rawTypeOptions =
          typeOptionsFromAPI.length > 0
            ? typeOptionsFromAPI
            : Array.from(fallbackTypeMap.values());
        rawTypeOptions.sort((a, b) =>
          String(a.label).localeCompare(String(b.label), 'zh-Hans-CN'),
        );
        const modelTypeOptions = [{ label: t('全部类型'), value: '' }].concat(
          rawTypeOptions,
        );
        setModelTypes(modelTypeOptions);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('加载模型失败'));
    }
  }, [setModelTypes, t]);

  /**
   * 在原始模型行与「模型类型」上按模型广场方式本地筛出模型下拉里模型名；不发起额外请求
   */
  useEffect(() => {
    if (!userState?.user) {
      setPlaygroundRawModels([]);
      setModels([]);
      setSupplierOptions([{ label: `${t('随机')} (${t('默认')})`, value: '' }]);
      return;
    }
    if (playgroundRawModels.length === 0) {
      setModels([]);
      setSupplierOptions([{ label: `${t('随机')} (${t('默认')})`, value: '' }]);
      return;
    }
    const typeOptions =
      modelTypes && modelTypes.length > 0
        ? modelTypes
        : [{ label: t('全部类型'), value: '' }];

    const selectedType = normalizeTypeId(inputs.model_type);
    const hasSelectedType = typeOptions.some((option) => {
      if (option.value === '' && selectedType === '') return true;
      if (option.value === '' || isTypeSelectionEmpty(selectedType))
        return false;
      return Number(option.value) === selectedType;
    });
    if (!hasSelectedType && !isTypeSelectionEmpty(selectedType)) {
      handleInputChange('model_type', '');
    }
    const effectiveType = hasSelectedType ? selectedType : '';
    const filteredItems = filterPlaygroundModelsByType(
      playgroundRawModels,
      effectiveType,
      typeOptions,
    );
    const { modelOptions, selectedModel } = processModelsData(
      filteredItems.map((item) => item.model_name),
      inputs.model,
    );
    setModels(modelOptions);
    if (selectedModel !== inputs.model) {
      handleInputChange('model', selectedModel);
    }
    const selectedModelName =
      selectedModel !== undefined && selectedModel !== null
        ? selectedModel
        : inputs.model;
    const selectedModelRow = filteredItems.find(
      (item) => item?.model_name === selectedModelName,
    );
    const supplierOptionsRaw = Array.isArray(selectedModelRow?.channel_options)
      ? selectedModelRow.channel_options
      : [];
    const supplierOptions = [
      { label: `${t('随机')} (${t('默认')})`, value: '' },
      ...supplierOptionsRaw
        .map((option) => {
          const id = Number(option?.id);
          if (!Number.isFinite(id) || id <= 0) {
            return null;
          }
          const name = String(option?.name || '').trim() || `渠道#${id}`;
          const channelNo = String(option?.channel_no || '').trim();
          const label = channelNo ? `${name} (${channelNo})` : name;
          return {
            label,
            value: id,
          };
        })
        .filter(Boolean),
    ];
    setSupplierOptions(supplierOptions);
    const selectedSupplierID = inputs.specific_channel_id;
    const hasSelectedSupplier = supplierOptions.some((option) => {
      if (
        option.value === '' &&
        (selectedSupplierID === '' || selectedSupplierID == null)
      ) {
        return true;
      }
      if (
        option.value === '' ||
        selectedSupplierID === '' ||
        selectedSupplierID == null
      ) {
        return false;
      }
      return Number(option.value) === Number(selectedSupplierID);
    });
    if (!hasSelectedSupplier) {
      handleInputChange('specific_channel_id', '');
    }
  }, [
    userState?.user,
    playgroundRawModels,
    modelTypes,
    inputs.model_type,
    inputs.model,
    inputs.specific_channel_id,
    t,
    setModels,
    setSupplierOptions,
    handleInputChange,
  ]);

  /**
   * 拉取用户可用分组，并校正当前选中分组
   */
  const loadGroups = useCallback(async () => {
    try {
      const res = await API.get(API_ENDPOINTS.USER_GROUPS);
      const { success, message, data } = res.data;

      if (success) {
        const userGroup =
          userState?.user?.group ||
          JSON.parse(localStorage.getItem('user'))?.group;
        const groupOptions = processGroupsData(data, userGroup);
        setGroups(groupOptions);

        const hasCurrentGroup = groupOptions.some(
          (option) => option.value === inputs.group,
        );
        if (!hasCurrentGroup) {
          handleInputChange('group', groupOptions[0]?.value || '');
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('加载分组失败'));
    }
  }, [userState, inputs.group, handleInputChange, setGroups, t]);

  /**
   * 用户登录后自动加载模型列表与分组；依赖 loadModels / loadGroups 以在 model_type 等变化时重算筛选结果
   */
  useEffect(() => {
    if (userState?.user) {
      loadModels();
      loadGroups();
    }
  }, [userState?.user, loadModels, loadGroups]);

  return {
    loadModels,
    loadGroups,
  };
};
